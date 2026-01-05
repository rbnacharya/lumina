package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/lumina/gateway/internal/auth"
	"github.com/lumina/gateway/internal/database"
	"github.com/lumina/gateway/internal/logging"
	"github.com/lumina/gateway/internal/models"
)

// Handler handles dashboard API requests
type Handler struct {
	db          *database.DB
	keyService  *auth.KeyService
	jwtManager  *auth.JWTManager
	logPipeline *logging.Pipeline
}

// NewHandler creates a new API handler
func NewHandler(db *database.DB, keyService *auth.KeyService, jwtManager *auth.JWTManager) *Handler {
	return &Handler{
		db:         db,
		keyService: keyService,
		jwtManager: jwtManager,
	}
}

// SetLogPipeline sets the log pipeline (called after initialization)
func (h *Handler) SetLogPipeline(pipeline *logging.Pipeline) {
	h.logPipeline = pipeline
}

// Auth handlers

// Register handles user registration
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password required"})
		return
	}

	// Check if user exists
	existing, err := h.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if existing != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Create user
	user, err := h.db.CreateUser(r.Context(), req.Email, string(hash))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
		return
	}

	// Generate token
	token, err := h.jwtManager.GenerateToken(user.ID, user.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours
	})

	writeJSON(w, http.StatusCreated, models.AuthResponse{User: user, Token: token})
}

// Login handles user login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Get user
	user, err := h.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	// Generate token
	token, err := h.jwtManager.GenerateToken(user.ID, user.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours
	})

	writeJSON(w, http.StatusOK, models.AuthResponse{User: user, Token: token})
}

// Logout handles user logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// Me returns the current user
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	user, err := h.db.GetUserByID(r.Context(), userID)
	if err != nil || user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// Key management handlers

// ListKeys lists all virtual keys for the user
func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	keys, err := h.keyService.ListKeys(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list keys"})
		return
	}

	// Mask sensitive data
	for _, key := range keys {
		key.KeyHash = ""
		// Providers are included but real_key_encrypted is already excluded in JSON
	}

	writeJSON(w, http.StatusOK, keys)
}

// CreateKey creates a new virtual key (access control only)
func (h *Handler) CreateKey(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var req models.CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	resp, err := h.keyService.CreateKey(r.Context(), userID, &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create key"})
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// GetKey gets a single key by ID
func (h *Handler) GetKey(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	keyID := chi.URLParam(r, "id")

	key, err := h.keyService.GetKey(r.Context(), keyID, userID)
	if err != nil {
		if err.Error() == "key not found" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
			return
		}
		if err.Error() == "unauthorized" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get key"})
		return
	}

	// Mask sensitive data
	key.KeyHash = ""

	writeJSON(w, http.StatusOK, key)
}

// RevokeKey revokes a virtual key
func (h *Handler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	keyID := chi.URLParam(r, "id")

	if err := h.keyService.RevokeKey(r.Context(), keyID, userID); err != nil {
		if err.Error() == "key not found" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
			return
		}
		if err.Error() == "unauthorized" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to revoke key"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "key revoked"})
}

// UpdateKey updates a virtual key
func (h *Handler) UpdateKey(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	keyID := chi.URLParam(r, "id")

	var req models.UpdateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.keyService.UpdateKey(r.Context(), keyID, userID, &req); err != nil {
		if err.Error() == "key not found" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
			return
		}
		if err.Error() == "unauthorized" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update key"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "key updated"})
}

// User Provider handlers (account-level API keys)

// ListProviders lists all configured providers for the user
func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	providers, err := h.keyService.GetUserProviders(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list providers"})
		return
	}

	writeJSON(w, http.StatusOK, providers)
}

// SetProvider sets or updates an account-level provider API key
func (h *Handler) SetProvider(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var req models.SetProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Provider != models.ProviderOpenAI && req.Provider != models.ProviderAnthropic {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider must be 'openai' or 'anthropic'"})
		return
	}

	if req.APIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "api_key is required"})
		return
	}

	if err := h.keyService.SetUserProvider(r.Context(), userID, req.Provider, req.APIKey); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to set provider"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "provider configured"})
}

// RemoveProvider removes an account-level provider API key
func (h *Handler) RemoveProvider(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	provider := chi.URLParam(r, "provider")

	var providerType models.ProviderType
	switch provider {
	case "openai":
		providerType = models.ProviderOpenAI
	case "anthropic":
		providerType = models.ProviderAnthropic
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid provider"})
		return
	}

	if err := h.keyService.RemoveUserProvider(r.Context(), userID, providerType); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to remove provider"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "provider removed"})
}

// Stats handlers

// GetOverview returns overview statistics
func (h *Handler) GetOverview(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	// Get from database for now (can enhance with OpenSearch later)
	overview, err := h.db.GetUserOverview(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get overview"})
		return
	}

	// If log pipeline is available, get additional stats
	if h.logPipeline != nil {
		endDate := time.Now()
		startDate := endDate.AddDate(0, 0, -30) // Last 30 days

		stats, err := h.logPipeline.GetStats(r.Context(), userID, startDate, endDate)
		if err == nil {
			overview.TotalRequests = stats.TotalRequests
			overview.AvgLatency = stats.AvgLatency
			overview.SuccessRate = stats.SuccessRate
		}
	}

	writeJSON(w, http.StatusOK, overview)
}

// GetDailyStats returns daily statistics
func (h *Handler) GetDailyStats(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	// Parse date range
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7) // Default to last 7 days

	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			startDate = t
		}
	}

	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			endDate = t
		}
	}

	stats, err := h.db.GetDailyStats(r.Context(), userID, startDate, endDate)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get daily stats"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// Log handlers

// SearchLogs searches through logs
func (h *Handler) SearchLogs(w http.ResponseWriter, r *http.Request) {
	if h.logPipeline == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "logging not available"})
		return
	}

	query := r.URL.Query().Get("q")
	model := r.URL.Query().Get("model")

	var statusCode *int
	if sc := r.URL.Query().Get("status"); sc != "" {
		if code, err := strconv.Atoi(sc); err == nil {
			statusCode = &code
		}
	}

	var startDate, endDate *time.Time
	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			startDate = &t
		}
	}
	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			endDate = &t
		}
	}

	page := 0
	if p := r.URL.Query().Get("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil {
			page = pageNum
		}
	}

	size := 20
	if s := r.URL.Query().Get("size"); s != "" {
		if sizeNum, err := strconv.Atoi(s); err == nil && sizeNum <= 100 {
			size = sizeNum
		}
	}

	entries, total, err := h.logPipeline.Search(r.Context(), query, model, statusCode, startDate, endDate, page*size, size)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"total":   total,
		"page":    page,
		"size":    size,
	})
}

// GetLog retrieves a single log entry
func (h *Handler) GetLog(w http.ResponseWriter, r *http.Request) {
	if h.logPipeline == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "logging not available"})
		return
	}

	traceID := chi.URLParam(r, "id")

	entry, err := h.logPipeline.GetLog(r.Context(), traceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get log"})
		return
	}
	if entry == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "log not found"})
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
