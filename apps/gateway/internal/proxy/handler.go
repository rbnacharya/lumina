package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lumina/gateway/internal/auth"
	"github.com/lumina/gateway/internal/logging"
	"github.com/lumina/gateway/internal/models"
)

const (
	openAIBaseURL    = "https://api.openai.com"
	anthropicBaseURL = "https://api.anthropic.com"
)

// Handler handles LLM proxy requests
type Handler struct {
	keyService  *auth.KeyService
	logPipeline *logging.Pipeline
	httpClient  *http.Client
}

// NewHandler creates a new proxy handler
func NewHandler(keyService *auth.KeyService, logPipeline *logging.Pipeline) *Handler {
	return &Handler{
		keyService:  keyService,
		logPipeline: logPipeline,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// parseModel parses a model string in the format "provider/model"
// Returns provider, actual model name, and error
func parseModel(model string) (provider string, actualModel string, err error) {
	parts := strings.SplitN(model, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid model format: expected 'provider/model', got '%s'", model)
	}
	return parts[0], parts[1], nil
}

// ChatCompletions handles chat completions with unified provider/model format
func (h *Handler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	h.proxyUnified(w, r, "/v1/chat/completions", "chat")
}

// Completions handles completions with unified provider/model format
func (h *Handler) Completions(w http.ResponseWriter, r *http.Request) {
	h.proxyUnified(w, r, "/v1/completions", "completion")
}

// Embeddings handles embeddings with unified provider/model format
func (h *Handler) Embeddings(w http.ResponseWriter, r *http.Request) {
	h.proxyUnified(w, r, "/v1/embeddings", "embedding")
}

// AnthropicMessages handles Anthropic messages API with unified provider/model format
func (h *Handler) AnthropicMessages(w http.ResponseWriter, r *http.Request) {
	h.proxyUnified(w, r, "/v1/messages", "anthropic")
}

// proxyUnified handles all proxy requests with the unified provider/model format
func (h *Handler) proxyUnified(w http.ResponseWriter, r *http.Request, path string, requestType string) {
	ctx := r.Context()
	traceID := uuid.New().String()
	startTime := time.Now()

	// Extract and validate virtual key
	keyConfig, err := h.extractAndValidateKey(ctx, r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	r.Body.Close()

	// Parse request for logging
	var requestData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Extract model (in format "provider/model")
	modelField := extractModel(requestData)
	provider, actualModel, err := parseModel(modelField)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate model is allowed
	if !h.keyService.IsModelAllowed(keyConfig, modelField) {
		h.writeError(w, http.StatusForbidden, fmt.Sprintf("model '%s' is not allowed for this key", modelField))
		return
	}

	// Get API key for the provider
	realAPIKey, err := h.keyService.GetProviderKey(keyConfig, provider)
	fmt.Println("Provider:", provider, "API Key:", realAPIKey)
	if err != nil {
		if err == auth.ErrProviderNotFound {
			h.writeError(w, http.StatusBadRequest, fmt.Sprintf("provider '%s' is not configured for this key", provider))
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to get provider key")
		return
	}

	// Replace model with actual model name (without provider prefix)
	requestData["model"] = actualModel
	modifiedBody, err := json.Marshal(requestData)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to modify request")
		return
	}

	// Check if streaming
	isStreaming := false
	if stream, ok := requestData["stream"].(bool); ok {
		isStreaming = stream
	}

	// Route to appropriate provider
	var targetURL string
	var headers map[string]string

	switch provider {
	case "openai":
		targetURL = openAIBaseURL + path
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + realAPIKey,
		}
	case "anthropic":
		// Anthropic uses different endpoint
		targetURL = anthropicBaseURL + "/v1/messages"
		headers = map[string]string{
			"Content-Type":      "application/json",
			"x-api-key":         realAPIKey,
			"anthropic-version": "2023-06-01",
		}
	default:
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported provider: %s", provider))
		return
	}

	// Create upstream request
	upstreamReq, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(modifiedBody))
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to create upstream request")
		return
	}

	// Set headers
	for key, value := range headers {
		upstreamReq.Header.Set(key, value)
	}

	// Forward request
	resp, err := h.httpClient.Do(upstreamReq)
	if err != nil {
		h.writeError(w, http.StatusBadGateway, "failed to reach upstream")
		return
	}
	defer resp.Body.Close()

	latencyMs := int(time.Since(startTime).Milliseconds())

	if isStreaming {
		h.handleStreamingResponse(w, resp, traceID, keyConfig, requestData, provider, modelField, startTime)
	} else {
		h.handleJSONResponse(w, resp, traceID, keyConfig, requestData, provider, modelField, latencyMs)
	}
}

func (h *Handler) extractAndValidateKey(ctx context.Context, r *http.Request) (*models.KeyConfig, error) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("missing or invalid authorization header")
	}

	virtualKey := strings.TrimPrefix(authHeader, "Bearer ")
	return h.keyService.ValidateKey(ctx, virtualKey)
}

func (h *Handler) handleJSONResponse(w http.ResponseWriter, resp *http.Response, traceID string, keyConfig *models.KeyConfig, requestData map[string]interface{}, provider string, fullModel string, latencyMs int) {
	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.writeError(w, http.StatusBadGateway, "failed to read upstream response")
		return
	}

	// Parse response for logging
	var responseData map[string]interface{}
	json.Unmarshal(respBody, &responseData)

	// Extract usage info
	usage := models.UsageLog{}
	if u, ok := responseData["usage"].(map[string]interface{}); ok {
		if pt, ok := u["prompt_tokens"].(float64); ok {
			usage.PromptTokens = int(pt)
		}
		if ct, ok := u["completion_tokens"].(float64); ok {
			usage.CompletionTokens = int(ct)
		}
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	// Calculate cost using provider
	cost := h.calculateCost(provider, fullModel, usage)

	// Update spend
	go func() {
		ctx := context.Background()
		if err := h.keyService.UpdateSpend(ctx, keyConfig.KeyID, cost, usage.TotalTokens); err != nil {
			slog.Error("failed to update spend", "error", err)
		}
	}()

	// Log the request
	logEntry := &models.LogEntry{
		TraceID:        traceID,
		Timestamp:      time.Now(),
		VirtualKeyName: keyConfig.Name,
		VirtualKeyID:   keyConfig.KeyID,
		UserID:         keyConfig.UserID,
		Request: models.RequestLog{
			Model:    fullModel,
			Provider: provider,
			Messages: requestData["messages"],
		},
		Response: models.ResponseLog{
			Content:    extractContent(responseData),
			Usage:      usage,
			StatusCode: resp.StatusCode,
		},
		Metrics: models.MetricsLog{
			LatencyMs: latencyMs,
			CostUSD:   cost,
		},
	}
	h.logPipeline.Log(logEntry)

	// Write response
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

func (h *Handler) handleStreamingResponse(w http.ResponseWriter, resp *http.Response, traceID string, keyConfig *models.KeyConfig, requestData map[string]interface{}, provider string, fullModel string, startTime time.Time) {
	// Set streaming headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Stream response
	var fullContent strings.Builder
	var usage models.UsageLog

	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			flusher.Flush()

			// Try to extract content from SSE data
			// This is a simplified version - production would parse SSE properly
			fullContent.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}

	latencyMs := int(time.Since(startTime).Milliseconds())

	// Log the streaming request (with partial data)
	logEntry := &models.LogEntry{
		TraceID:        traceID,
		Timestamp:      time.Now(),
		VirtualKeyName: keyConfig.Name,
		VirtualKeyID:   keyConfig.KeyID,
		UserID:         keyConfig.UserID,
		Request: models.RequestLog{
			Model:    fullModel,
			Provider: provider,
			Messages: requestData["messages"],
		},
		Response: models.ResponseLog{
			Content:    "[streaming response]",
			Usage:      usage,
			StatusCode: resp.StatusCode,
		},
		Metrics: models.MetricsLog{
			LatencyMs: latencyMs,
			CostUSD:   0, // Estimated separately for streaming
		},
	}
	h.logPipeline.Log(logEntry)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func extractModel(data map[string]interface{}) string {
	if model, ok := data["model"].(string); ok {
		return model
	}
	return "unknown"
}

func extractContent(data map[string]interface{}) string {
	// OpenAI format
	if choices, ok := data["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content
				}
			}
		}
	}

	// Anthropic format
	if content, ok := data["content"].([]interface{}); ok && len(content) > 0 {
		if item, ok := content[0].(map[string]interface{}); ok {
			if text, ok := item["text"].(string); ok {
				return text
			}
		}
	}

	return ""
}

func (h *Handler) calculateCost(provider string, model string, usage models.UsageLog) float64 {
	// Pricing per 1M tokens (simplified)
	var inputPrice, outputPrice float64

	// Extract just the model name if full format provided
	_, actualModel, err := parseModel(model)
	if err != nil {
		actualModel = model
	}

	switch provider {
	case "openai":
		switch {
		case strings.HasPrefix(actualModel, "gpt-4o"):
			inputPrice = 2.50
			outputPrice = 10.00
		case strings.HasPrefix(actualModel, "gpt-4"):
			inputPrice = 30.00
			outputPrice = 60.00
		case strings.HasPrefix(actualModel, "gpt-3.5"):
			inputPrice = 0.50
			outputPrice = 1.50
		case strings.HasPrefix(actualModel, "o1"):
			inputPrice = 15.00
			outputPrice = 60.00
		default:
			inputPrice = 1.00
			outputPrice = 2.00
		}
	case "anthropic":
		switch {
		case strings.Contains(actualModel, "opus"):
			inputPrice = 15.00
			outputPrice = 75.00
		case strings.Contains(actualModel, "sonnet"):
			inputPrice = 3.00
			outputPrice = 15.00
		case strings.Contains(actualModel, "haiku"):
			inputPrice = 0.25
			outputPrice = 1.25
		default:
			inputPrice = 3.00
			outputPrice = 15.00
		}
	default:
		inputPrice = 1.00
		outputPrice = 2.00
	}

	inputCost := float64(usage.PromptTokens) / 1_000_000 * inputPrice
	outputCost := float64(usage.CompletionTokens) / 1_000_000 * outputPrice

	return inputCost + outputCost
}
