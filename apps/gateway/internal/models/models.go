package models

import (
	"time"
)

// ProviderType represents the LLM provider
type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
)

// User represents a dashboard user
type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// VirtualKey represents a virtual API key (access control only, no provider keys)
type VirtualKey struct {
	ID            string     `json:"id" db:"id"`
	UserID        string     `json:"user_id" db:"user_id"`
	Name          string     `json:"name" db:"name"`
	KeyHash       string     `json:"-" db:"key_hash"`
	AllowedModels []string   `json:"allowed_models" db:"allowed_models"`
	BudgetLimit   *float64   `json:"budget_limit" db:"budget_limit"`
	CurrentSpend  float64    `json:"current_spend" db:"current_spend"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// UserProvider represents an account-level provider API key
type UserProvider struct {
	ID              string       `json:"id" db:"id"`
	UserID          string       `json:"user_id" db:"user_id"`
	Provider        ProviderType `json:"provider" db:"provider"`
	APIKeyEncrypted []byte       `json:"-" db:"api_key_encrypted"`
	CreatedAt       time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at" db:"updated_at"`
}

// DailyStat represents daily usage statistics
type DailyStat struct {
	ID          string    `json:"id" db:"id"`
	KeyID       string    `json:"key_id" db:"key_id"`
	Date        time.Time `json:"date" db:"date"`
	TotalTokens int       `json:"total_tokens" db:"total_tokens"`
	TotalCost   float64   `json:"total_cost" db:"total_cost"`
}

// KeyConfig is cached in Redis for fast lookups
type KeyConfig struct {
	KeyID         string            `json:"key_id"`
	UserID        string            `json:"user_id"`
	Name          string            `json:"name"`
	AllowedModels []string          `json:"allowed_models"`
	Providers     map[string]string `json:"providers"` // provider -> real_api_key (from user account)
	BudgetLimit   *float64          `json:"budget_limit"`
	CurrentSpend  float64           `json:"current_spend"`
}

// LogEntry represents a logged request/response
type LogEntry struct {
	TraceID        string      `json:"trace_id"`
	Timestamp      time.Time   `json:"timestamp"`
	VirtualKeyName string      `json:"virtual_key_name"`
	VirtualKeyID   string      `json:"virtual_key_id"`
	UserID         string      `json:"user_id"`
	Request        RequestLog  `json:"request"`
	Response       ResponseLog `json:"response"`
	Metrics        MetricsLog  `json:"metrics"`
}

// RequestLog contains the request details
type RequestLog struct {
	Model       string      `json:"model"`
	Provider    string      `json:"provider"`
	Messages    interface{} `json:"messages,omitempty"`
	Prompt      string      `json:"prompt,omitempty"`
	Temperature *float64    `json:"temperature,omitempty"`
	MaxTokens   *int        `json:"max_tokens,omitempty"`
}

// ResponseLog contains the response details
type ResponseLog struct {
	Content    string   `json:"content,omitempty"`
	Usage      UsageLog `json:"usage"`
	StatusCode int      `json:"status_code"`
	Error      string   `json:"error,omitempty"`
}

// UsageLog contains token usage
type UsageLog struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// MetricsLog contains performance metrics
type MetricsLog struct {
	LatencyMs int     `json:"latency_ms"`
	CostUSD   float64 `json:"cost_usd"`
}

// Overview represents dashboard overview stats
type Overview struct {
	TotalSpend    float64 `json:"total_spend"`
	TotalRequests int64   `json:"total_requests"`
	AvgLatency    float64 `json:"avg_latency"`
	SuccessRate   float64 `json:"success_rate"`
}

// CreateKeyRequest is the request to create a new virtual key
type CreateKeyRequest struct {
	Name          string   `json:"name"`
	AllowedModels []string `json:"allowed_models"` // e.g., ["openai/*", "anthropic/claude-3-*"]
	BudgetLimit   *float64 `json:"budget_limit"`
}

// UpdateKeyRequest is the request to update a virtual key
type UpdateKeyRequest struct {
	Name          *string  `json:"name,omitempty"`
	AllowedModels []string `json:"allowed_models,omitempty"` // Replace allowed models
	BudgetLimit   *float64 `json:"budget_limit,omitempty"`
}

// SetProviderRequest is the request to set an account-level provider API key
type SetProviderRequest struct {
	Provider ProviderType `json:"provider"`
	APIKey   string       `json:"api_key"`
}

// ProviderInfo represents provider info returned to the frontend (without the actual key)
type ProviderInfo struct {
	Provider  ProviderType `json:"provider"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// CreateKeyResponse is the response after creating a key
type CreateKeyResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	AllowedModels []string `json:"allowed_models"`
	VirtualKey    string   `json:"virtual_key"` // Only shown once
	CreatedAt     time.Time `json:"created_at"`
}

// LoginRequest is the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest is the registration request body
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is the response for auth operations
type AuthResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token,omitempty"`
}
