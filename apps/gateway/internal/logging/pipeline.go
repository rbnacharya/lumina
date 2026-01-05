package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/lumina/gateway/internal/models"
)

const (
	indexName     = "lumina-logs"
	batchSize     = 100
	flushInterval = 5 * time.Second
	workerCount   = 10
	channelSize   = 1000
)

// Pipeline handles async logging to OpenSearch
type Pipeline struct {
	opensearchURL string
	httpClient    *http.Client
	logChan       chan *models.LogEntry
	batch         []*models.LogEntry
	batchMu       sync.Mutex
	wg            sync.WaitGroup
	done          chan struct{}
}

// New creates a new logging pipeline
func New(opensearchURL string) (*Pipeline, error) {
	slog.Info("initializing logging pipeline", "opensearch_url", opensearchURL)

	p := &Pipeline{
		opensearchURL: opensearchURL,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		logChan:       make(chan *models.LogEntry, channelSize),
		batch:         make([]*models.LogEntry, 0, batchSize),
		done:          make(chan struct{}),
	}

	// Create index if not exists
	if err := p.createIndex(); err != nil {
		slog.Warn("failed to create index", "error", err)
		// Don't fail - OpenSearch might not be ready yet
	} else {
		slog.Info("OpenSearch index created or already exists", "index", indexName)
	}

	// Start worker pool
	for i := 0; i < workerCount; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	slog.Info("started worker pool", "workers", workerCount)

	// Start batch flusher
	p.wg.Add(1)
	go p.flusher()
	slog.Info("started batch flusher", "interval", flushInterval)

	return p, nil
}

// Close shuts down the logging pipeline
func (p *Pipeline) Close() error {
	close(p.done)
	close(p.logChan)
	p.wg.Wait()

	// Flush remaining batch
	p.flush()

	return nil
}

// Log sends a log entry to the pipeline
func (p *Pipeline) Log(entry *models.LogEntry) {
	slog.Info("logging entry to pipeline", "trace_id", entry.TraceID, "model", entry.Request.Model)
	select {
	case p.logChan <- entry:
		slog.Debug("entry added to channel", "trace_id", entry.TraceID)
	default:
		slog.Warn("log channel full, dropping log entry", "trace_id", entry.TraceID)
	}
}

func (p *Pipeline) worker() {
	defer p.wg.Done()

	for {
		select {
		case entry, ok := <-p.logChan:
			if !ok {
				return
			}
			p.addToBatch(entry)
		case <-p.done:
			return
		}
	}
}

func (p *Pipeline) addToBatch(entry *models.LogEntry) {
	p.batchMu.Lock()
	p.batch = append(p.batch, entry)
	batchLen := len(p.batch)
	shouldFlush := batchLen >= batchSize
	p.batchMu.Unlock()

	slog.Info("added entry to batch", "trace_id", entry.TraceID, "batch_size", batchLen, "will_flush", shouldFlush)

	if shouldFlush {
		p.flush()
	}
}

func (p *Pipeline) flusher() {
	defer p.wg.Done()

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.flush()
		case <-p.done:
			return
		}
	}
}

func (p *Pipeline) flush() {
	p.batchMu.Lock()
	if len(p.batch) == 0 {
		p.batchMu.Unlock()
		return
	}

	batch := p.batch
	p.batch = make([]*models.LogEntry, 0, batchSize)
	p.batchMu.Unlock()

	slog.Info("flushing batch to OpenSearch", "count", len(batch), "url", p.opensearchURL)
	if err := p.bulkIndex(batch); err != nil {
		slog.Error("failed to bulk index logs", "error", err, "count", len(batch))
	} else {
		slog.Info("bulk indexed logs successfully", "count", len(batch))
	}
}

func (p *Pipeline) createIndex() error {
	mapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"trace_id":         map[string]string{"type": "keyword"},
				"timestamp":        map[string]string{"type": "date"},
				"virtual_key_name": map[string]string{"type": "keyword"},
				"virtual_key_id":   map[string]string{"type": "keyword"},
				"user_id":          map[string]string{"type": "keyword"},
				"request": map[string]interface{}{
					"properties": map[string]interface{}{
						"model":       map[string]string{"type": "keyword"},
						"messages":    map[string]string{"type": "keyword"},
						"temperature": map[string]string{"type": "float"},
						"max_tokens":  map[string]string{"type": "integer"},
					},
				},
				"response": map[string]interface{}{
					"properties": map[string]interface{}{
						"content":     map[string]string{"type": "text"},
						"status_code": map[string]string{"type": "integer"},
						"error":       map[string]string{"type": "text"},
						"usage": map[string]interface{}{
							"properties": map[string]interface{}{
								"prompt_tokens":     map[string]string{"type": "integer"},
								"completion_tokens": map[string]string{"type": "integer"},
								"total_tokens":      map[string]string{"type": "integer"},
							},
						},
					},
				},
				"metrics": map[string]interface{}{
					"properties": map[string]interface{}{
						"latency_ms": map[string]string{"type": "integer"},
						"cost_usd":   map[string]string{"type": "float"},
					},
				},
			},
		},
	}

	body, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	req, err := http.NewRequest("PUT", p.opensearchURL+"/"+indexName, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer resp.Body.Close()

	// 400 is ok - index already exists
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// toIndexableDoc converts a LogEntry to an indexable document,
// serializing complex fields like messages to JSON strings
func (p *Pipeline) toIndexableDoc(entry *models.LogEntry) map[string]interface{} {
	// Convert messages to JSON string if it's not already a string
	var messagesStr string
	if entry.Request.Messages != nil {
		if str, ok := entry.Request.Messages.(string); ok {
			messagesStr = str
		} else {
			msgBytes, _ := json.Marshal(entry.Request.Messages)
			messagesStr = string(msgBytes)
		}
	}

	return map[string]interface{}{
		"trace_id":         entry.TraceID,
		"timestamp":        entry.Timestamp,
		"virtual_key_name": entry.VirtualKeyName,
		"virtual_key_id":   entry.VirtualKeyID,
		"user_id":          entry.UserID,
		"request": map[string]interface{}{
			"model":       entry.Request.Model,
			"provider":    entry.Request.Provider,
			"messages":    messagesStr,
			"prompt":      entry.Request.Prompt,
			"temperature": entry.Request.Temperature,
			"max_tokens":  entry.Request.MaxTokens,
		},
		"response": map[string]interface{}{
			"content":     entry.Response.Content,
			"status_code": entry.Response.StatusCode,
			"error":       entry.Response.Error,
			"usage": map[string]interface{}{
				"prompt_tokens":     entry.Response.Usage.PromptTokens,
				"completion_tokens": entry.Response.Usage.CompletionTokens,
				"total_tokens":      entry.Response.Usage.TotalTokens,
			},
		},
		"metrics": map[string]interface{}{
			"latency_ms": entry.Metrics.LatencyMs,
			"cost_usd":   entry.Metrics.CostUSD,
		},
	}
}

func (p *Pipeline) bulkIndex(entries []*models.LogEntry) error {
	var buf bytes.Buffer

	for _, entry := range entries {
		// Action line
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": indexName,
				"_id":    entry.TraceID,
			},
		}
		actionBytes, _ := json.Marshal(action)
		buf.Write(actionBytes)
		buf.WriteByte('\n')

		// Convert messages to JSON string for OpenSearch text field
		doc := p.toIndexableDoc(entry)
		docBytes, _ := json.Marshal(doc)
		buf.Write(docBytes)
		buf.WriteByte('\n')
	}

	req, err := http.NewRequest("POST", p.opensearchURL+"/_bulk", &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to bulk index: %w", err)
	}
	defer resp.Body.Close()

	// Read and parse response body
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		slog.Error("OpenSearch bulk index failed", "status", resp.StatusCode, "response", string(respBody))
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse bulk response to check for individual document errors
	var bulkResp struct {
		Took   int  `json:"took"`
		Errors bool `json:"errors"`
		Items  []struct {
			Index struct {
				ID     string `json:"_id"`
				Status int    `json:"status"`
				Error  *struct {
					Type   string `json:"type"`
					Reason string `json:"reason"`
				} `json:"error,omitempty"`
			} `json:"index"`
		} `json:"items"`
	}

	if err := json.Unmarshal(respBody, &bulkResp); err != nil {
		slog.Warn("failed to parse bulk response", "error", err)
		return nil
	}

	if bulkResp.Errors {
		var failedCount int
		for _, item := range bulkResp.Items {
			if item.Index.Error != nil {
				failedCount++
				slog.Error("document index failed",
					"id", item.Index.ID,
					"status", item.Index.Status,
					"error_type", item.Index.Error.Type,
					"reason", item.Index.Error.Reason)
			}
		}
		return fmt.Errorf("bulk index had %d failed documents out of %d", failedCount, len(bulkResp.Items))
	}

	return nil
}

// Search searches logs in OpenSearch
func (p *Pipeline) Search(ctx context.Context, query string, model string, statusCode *int, startDate, endDate *time.Time, from, size int) ([]*models.LogEntry, int64, error) {
	must := make([]map[string]interface{}, 0)

	if query != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"request.messages", "response.content"},
			},
		})
	}

	if model != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]string{"request.model": model},
		})
	}

	if statusCode != nil {
		must = append(must, map[string]interface{}{
			"term": map[string]int{"response.status_code": *statusCode},
		})
	}

	if startDate != nil || endDate != nil {
		rangeQuery := map[string]interface{}{}
		if startDate != nil {
			rangeQuery["gte"] = startDate.Format(time.RFC3339)
		}
		if endDate != nil {
			rangeQuery["lte"] = endDate.Format(time.RFC3339)
		}
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{"timestamp": rangeQuery},
		})
	}

	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
		"sort": []map[string]interface{}{
			{"timestamp": map[string]string{"order": "desc"}},
		},
		"from": from,
		"size": size,
	}

	body, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.opensearchURL+"/"+indexName+"/_search", bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source *models.LogEntry `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	entries := make([]*models.LogEntry, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		entries = append(entries, hit.Source)
	}

	return entries, result.Hits.Total.Value, nil
}

// GetLog retrieves a single log entry by ID
func (p *Pipeline) GetLog(ctx context.Context, traceID string) (*models.LogEntry, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.opensearchURL+"/"+indexName+"/_doc/"+traceID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	var result struct {
		Source *models.LogEntry `json:"_source"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Source, nil
}

// GetStats retrieves aggregated statistics
func (p *Pipeline) GetStats(ctx context.Context, userID string, startDate, endDate time.Time) (*models.Overview, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"user_id": userID}},
					{"range": map[string]interface{}{
						"timestamp": map[string]interface{}{
							"gte": startDate.Format(time.RFC3339),
							"lte": endDate.Format(time.RFC3339),
						},
					}},
				},
			},
		},
		"aggs": map[string]interface{}{
			"total_cost": map[string]interface{}{
				"sum": map[string]string{"field": "metrics.cost_usd"},
			},
			"avg_latency": map[string]interface{}{
				"avg": map[string]string{"field": "metrics.latency_ms"},
			},
			"success_count": map[string]interface{}{
				"filter": map[string]interface{}{
					"range": map[string]interface{}{
						"response.status_code": map[string]int{"lt": 400},
					},
				},
			},
		},
		"size": 0,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.opensearchURL+"/"+indexName+"/_search", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
		} `json:"hits"`
		Aggregations struct {
			TotalCost struct {
				Value float64 `json:"value"`
			} `json:"total_cost"`
			AvgLatency struct {
				Value float64 `json:"value"`
			} `json:"avg_latency"`
			SuccessCount struct {
				DocCount int64 `json:"doc_count"`
			} `json:"success_count"`
		} `json:"aggregations"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	successRate := 0.0
	if result.Hits.Total.Value > 0 {
		successRate = float64(result.Aggregations.SuccessCount.DocCount) / float64(result.Hits.Total.Value) * 100
	}

	return &models.Overview{
		TotalSpend:    result.Aggregations.TotalCost.Value,
		TotalRequests: result.Hits.Total.Value,
		AvgLatency:    result.Aggregations.AvgLatency.Value,
		SuccessRate:   successRate,
	}, nil
}
