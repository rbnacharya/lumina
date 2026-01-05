package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lumina/gateway/internal/cache"
	"github.com/lumina/gateway/internal/database"
	"github.com/lumina/gateway/internal/models"
)

const (
	virtualKeyPrefix = "lum_"
)

var (
	ErrInvalidKey       = errors.New("invalid virtual key")
	ErrKeyRevoked       = errors.New("virtual key has been revoked")
	ErrBudgetExceeded   = errors.New("budget limit exceeded")
	ErrModelNotAllowed  = errors.New("model not allowed for this key")
	ErrProviderNotFound = errors.New("provider not configured for this key")
)

// KeyService manages virtual keys
type KeyService struct {
	db            *database.DB
	cache         *cache.Cache
	encryptionKey []byte
}

// NewKeyService creates a new key service
func NewKeyService(db *database.DB, cache *cache.Cache, encryptionKey string) *KeyService {
	return &KeyService{
		db:            db,
		cache:         cache,
		encryptionKey: []byte(encryptionKey[:32]), // Use first 32 bytes
	}
}

// GenerateVirtualKey generates a new virtual key
func (s *KeyService) GenerateVirtualKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return virtualKeyPrefix + hex.EncodeToString(b)
}

// HashKey creates a SHA256 hash of a virtual key
func (s *KeyService) HashKey(virtualKey string) string {
	hash := sha256.Sum256([]byte(virtualKey))
	return hex.EncodeToString(hash[:])
}

// Encrypt encrypts the real API key
func (s *KeyService) Encrypt(plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, nil
}

// Decrypt decrypts the real API key
func (s *KeyService) Decrypt(ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// CreateKey creates a new virtual key (access control only, providers are at account level)
func (s *KeyService) CreateKey(ctx context.Context, userID string, req *models.CreateKeyRequest) (*models.CreateKeyResponse, error) {
	// Generate virtual key
	virtualKey := s.GenerateVirtualKey()
	keyHash := s.HashKey(virtualKey)

	// Create key in database
	key := &models.VirtualKey{
		ID:            uuid.New().String(),
		UserID:        userID,
		Name:          req.Name,
		KeyHash:       keyHash,
		AllowedModels: req.AllowedModels,
		BudgetLimit:   req.BudgetLimit,
		CurrentSpend:  0,
		CreatedAt:     time.Now(),
	}

	if err := s.db.CreateVirtualKey(ctx, key); err != nil {
		return nil, err
	}

	return &models.CreateKeyResponse{
		ID:            key.ID,
		Name:          key.Name,
		AllowedModels: key.AllowedModels,
		VirtualKey:    virtualKey, // Only returned once
		CreatedAt:     key.CreatedAt,
	}, nil
}

// ValidateKey validates a virtual key and returns the key configuration
func (s *KeyService) ValidateKey(ctx context.Context, virtualKey string) (*models.KeyConfig, error) {
	if !strings.HasPrefix(virtualKey, virtualKeyPrefix) {
		return nil, ErrInvalidKey
	}

	keyHash := s.HashKey(virtualKey)

	// Check cache first
	config, err := s.cache.GetKeyConfig(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("cache error: %w", err)
	}

	if config != nil {
		return config, nil
	}

	// Fallback to database
	key, err := s.db.GetVirtualKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if key == nil {
		return nil, ErrInvalidKey
	}

	if key.RevokedAt != nil {
		return nil, ErrKeyRevoked
	}

	// Fetch provider API keys from user's account (not the key)
	userProviders, err := s.db.GetUserProviders(ctx, key.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user providers: %w", err)
	}

	// Decrypt all provider API keys
	providers := make(map[string]string)
	for _, p := range userProviders {
		realAPIKey, err := s.Decrypt(p.APIKeyEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decryption error: %w", err)
		}
		providers[string(p.Provider)] = realAPIKey
	}

	config = &models.KeyConfig{
		KeyID:         key.ID,
		UserID:        key.UserID,
		Name:          key.Name,
		AllowedModels: key.AllowedModels,
		Providers:     providers,
		BudgetLimit:   key.BudgetLimit,
		CurrentSpend:  key.CurrentSpend,
	}

	// Cache the configuration
	if err := s.cache.SetKeyConfig(ctx, keyHash, config); err != nil {
		// Log but don't fail
		fmt.Printf("failed to cache key config: %v\n", err)
	}

	return config, nil
}

// GetProviderKey returns the API key for a specific provider
func (s *KeyService) GetProviderKey(config *models.KeyConfig, provider string) (string, error) {
	apiKey, ok := config.Providers[provider]
	if !ok {
		return "", ErrProviderNotFound
	}
	return apiKey, nil
}

// IsModelAllowed checks if a model is allowed for the key
// Model format: "provider/model" e.g., "openai/gpt-4o", "anthropic/claude-3-sonnet"
func (s *KeyService) IsModelAllowed(config *models.KeyConfig, model string) bool {
	// If no allowed models specified, allow all
	if len(config.AllowedModels) == 0 {
		return true
	}

	for _, pattern := range config.AllowedModels {
		if matchModelPattern(pattern, model) {
			return true
		}
	}
	return false
}

// matchModelPattern matches a model against a pattern
// Patterns can be:
// - exact: "openai/gpt-4o"
// - provider wildcard: "openai/*"
// - model wildcard: "*/gpt-4*"
// - full wildcard: "*"
func matchModelPattern(pattern, model string) bool {
	if pattern == "*" {
		return true
	}

	// Use filepath.Match for glob-style matching
	matched, err := filepath.Match(pattern, model)
	if err == nil && matched {
		return true
	}

	// Also support simple prefix matching for patterns ending with *
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}

	return false
}

// CheckBudget checks if the request would exceed the budget limit
func (s *KeyService) CheckBudget(config *models.KeyConfig, estimatedCost float64) error {
	if config.BudgetLimit == nil {
		return nil
	}

	if config.CurrentSpend+estimatedCost > *config.BudgetLimit {
		return ErrBudgetExceeded
	}

	return nil
}

// UpdateSpend updates the spend for a key
func (s *KeyService) UpdateSpend(ctx context.Context, keyID string, cost float64, tokens int) error {
	// Update database
	if err := s.db.UpdateKeySpend(ctx, keyID, cost); err != nil {
		return err
	}

	// Update daily stats
	if err := s.db.UpsertDailyStat(ctx, keyID, tokens, cost); err != nil {
		return err
	}

	return nil
}

// RevokeKey revokes a virtual key
func (s *KeyService) RevokeKey(ctx context.Context, keyID, userID string) error {
	// Get key to verify ownership
	key, err := s.db.GetVirtualKeyByID(ctx, keyID)
	if err != nil {
		return err
	}

	if key == nil {
		return errors.New("key not found")
	}

	if key.UserID != userID {
		return errors.New("unauthorized")
	}

	// Revoke in database
	if err := s.db.RevokeVirtualKey(ctx, keyID); err != nil {
		return err
	}

	// Remove from cache
	if err := s.cache.DeleteKeyConfig(ctx, key.KeyHash); err != nil {
		// Log but don't fail
		fmt.Printf("failed to delete key from cache: %v\n", err)
	}

	return nil
}

// UpdateKey updates a virtual key
func (s *KeyService) UpdateKey(ctx context.Context, keyID, userID string, req *models.UpdateKeyRequest) error {
	// Get key to verify ownership
	key, err := s.db.GetVirtualKeyByID(ctx, keyID)
	if err != nil {
		return err
	}

	if key == nil {
		return errors.New("key not found")
	}

	if key.UserID != userID {
		return errors.New("unauthorized")
	}

	// Update basic info (name, allowed_models, budget_limit)
	if err := s.db.UpdateVirtualKey(ctx, keyID, req.Name, req.AllowedModels, req.BudgetLimit); err != nil {
		return err
	}

	// Invalidate cache
	if err := s.cache.DeleteKeyConfig(ctx, key.KeyHash); err != nil {
		fmt.Printf("failed to delete key from cache: %v\n", err)
	}

	return nil
}

// invalidateUserKeyCache invalidates all cached key configs for a user
func (s *KeyService) invalidateUserKeyCache(ctx context.Context, userID string) error {
	keys, err := s.db.ListVirtualKeysByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to list user keys: %w", err)
	}

	fmt.Printf("invalidating cache for %d keys for user %s\n", len(keys), userID)
	for _, key := range keys {
		fmt.Printf("deleting cache for key %s (hash: %s)\n", key.ID, key.KeyHash)
		if err := s.cache.DeleteKeyConfig(ctx, key.KeyHash); err != nil {
			fmt.Printf("failed to delete key %s from cache: %v\n", key.ID, err)
		}
	}

	return nil
}

// SetUserProvider sets or updates an account-level provider API key
func (s *KeyService) SetUserProvider(ctx context.Context, userID string, provider models.ProviderType, apiKey string) error {
	encryptedKey, err := s.Encrypt(apiKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	if err := s.db.SetUserProvider(ctx, userID, provider, encryptedKey); err != nil {
		return err
	}

	// Invalidate all cached keys for this user since they contain provider keys
	if err := s.invalidateUserKeyCache(ctx, userID); err != nil {
		fmt.Printf("failed to invalidate user key cache: %v\n", err)
	}

	return nil
}

// GetUserProviders returns all configured providers for a user (without actual API keys)
func (s *KeyService) GetUserProviders(ctx context.Context, userID string) ([]models.ProviderInfo, error) {
	providers, err := s.db.GetUserProviders(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]models.ProviderInfo, len(providers))
	for i, p := range providers {
		result[i] = models.ProviderInfo{
			Provider:  p.Provider,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		}
	}

	return result, nil
}

// RemoveUserProvider removes an account-level provider API key
func (s *KeyService) RemoveUserProvider(ctx context.Context, userID string, provider models.ProviderType) error {
	if err := s.db.RemoveUserProvider(ctx, userID, provider); err != nil {
		return err
	}

	// Invalidate all cached keys for this user since they contain provider keys
	if err := s.invalidateUserKeyCache(ctx, userID); err != nil {
		fmt.Printf("failed to invalidate user key cache: %v\n", err)
	}

	return nil
}

// ListKeys lists all keys for a user
func (s *KeyService) ListKeys(ctx context.Context, userID string) ([]*models.VirtualKey, error) {
	return s.db.ListVirtualKeysByUser(ctx, userID)
}

// GetKey gets a key by ID
func (s *KeyService) GetKey(ctx context.Context, keyID, userID string) (*models.VirtualKey, error) {
	key, err := s.db.GetVirtualKeyByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	if key == nil {
		return nil, errors.New("key not found")
	}

	if key.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	return key, nil
}
