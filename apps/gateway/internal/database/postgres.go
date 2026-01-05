package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	_ "github.com/lib/pq"

	"github.com/lumina/gateway/internal/models"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection
func New(databaseURL string) (*DB, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	// Create migrations table if not exists
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read and execute migrations
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if migration was already applied
		var exists bool
		err := db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", entry.Name()).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if exists {
			continue
		}

		// Read and execute migration
		content, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		_, err = db.conn.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", entry.Name(), err)
		}

		// Record migration
		_, err = db.conn.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", entry.Name())
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// User operations

// CreateUser creates a new user
func (db *DB) CreateUser(ctx context.Context, email, passwordHash string) (*models.User, error) {
	user := &models.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}

	_, err := db.conn.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, created_at) VALUES ($1, $2, $3, $4)`,
		user.ID, user.Email, user.PasswordHash, user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// Virtual Key operations

// CreateVirtualKey creates a new virtual key (access control only, providers are at account level)
func (db *DB) CreateVirtualKey(ctx context.Context, key *models.VirtualKey) error {
	_, err := db.conn.ExecContext(ctx,
		`INSERT INTO virtual_keys (id, user_id, name, key_hash, allowed_models, budget_limit, current_spend, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		key.ID, key.UserID, key.Name, key.KeyHash, pq.Array(key.AllowedModels), key.BudgetLimit, key.CurrentSpend, key.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create virtual key: %w", err)
	}
	return nil
}

// User Provider operations (account-level API keys)

// SetUserProvider sets or updates a provider API key for a user's account
func (db *DB) SetUserProvider(ctx context.Context, userID string, provider models.ProviderType, encryptedKey []byte) error {
	_, err := db.conn.ExecContext(ctx,
		`INSERT INTO user_providers (id, user_id, provider, api_key_encrypted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (user_id, provider) DO UPDATE SET api_key_encrypted = EXCLUDED.api_key_encrypted, updated_at = NOW()`,
		uuid.New().String(), userID, provider, encryptedKey,
	)
	if err != nil {
		return fmt.Errorf("failed to set user provider: %w", err)
	}
	return nil
}

// GetUserProviders retrieves all provider API keys for a user's account
func (db *DB) GetUserProviders(ctx context.Context, userID string) ([]models.UserProvider, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT id, user_id, provider, api_key_encrypted, created_at, updated_at
		FROM user_providers WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user providers: %w", err)
	}
	defer rows.Close()

	var providers []models.UserProvider
	for rows.Next() {
		var p models.UserProvider
		err := rows.Scan(&p.ID, &p.UserID, &p.Provider, &p.APIKeyEncrypted, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user provider: %w", err)
		}
		providers = append(providers, p)
	}

	return providers, nil
}

// GetUserProvider retrieves a specific provider API key for a user
func (db *DB) GetUserProvider(ctx context.Context, userID string, provider models.ProviderType) (*models.UserProvider, error) {
	p := &models.UserProvider{}
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, user_id, provider, api_key_encrypted, created_at, updated_at
		FROM user_providers WHERE user_id = $1 AND provider = $2`,
		userID, provider,
	).Scan(&p.ID, &p.UserID, &p.Provider, &p.APIKeyEncrypted, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user provider: %w", err)
	}
	return p, nil
}

// RemoveUserProvider removes a provider API key from a user's account
func (db *DB) RemoveUserProvider(ctx context.Context, userID string, provider models.ProviderType) error {
	_, err := db.conn.ExecContext(ctx,
		`DELETE FROM user_providers WHERE user_id = $1 AND provider = $2`,
		userID, provider,
	)
	if err != nil {
		return fmt.Errorf("failed to remove user provider: %w", err)
	}
	return nil
}

// GetVirtualKeyByHash retrieves a virtual key by its hash
func (db *DB) GetVirtualKeyByHash(ctx context.Context, keyHash string) (*models.VirtualKey, error) {
	key := &models.VirtualKey{}
	var allowedModels pq.StringArray
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, user_id, name, key_hash, allowed_models, budget_limit, current_spend, created_at, revoked_at
		FROM virtual_keys WHERE key_hash = $1 AND revoked_at IS NULL`,
		keyHash,
	).Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &allowedModels, &key.BudgetLimit, &key.CurrentSpend, &key.CreatedAt, &key.RevokedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual key: %w", err)
	}
	key.AllowedModels = allowedModels

	return key, nil
}

// ListVirtualKeysByUser lists all virtual keys for a user
func (db *DB) ListVirtualKeysByUser(ctx context.Context, userID string) ([]*models.VirtualKey, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT id, user_id, name, key_hash, allowed_models, budget_limit, current_spend, created_at, revoked_at
		FROM virtual_keys WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.VirtualKey
	for rows.Next() {
		key := &models.VirtualKey{}
		var allowedModels pq.StringArray
		err := rows.Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &allowedModels, &key.BudgetLimit, &key.CurrentSpend, &key.CreatedAt, &key.RevokedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan virtual key: %w", err)
		}
		key.AllowedModels = allowedModels
		keys = append(keys, key)
	}

	return keys, nil
}

// GetVirtualKeyByID retrieves a virtual key by ID
func (db *DB) GetVirtualKeyByID(ctx context.Context, id string) (*models.VirtualKey, error) {
	key := &models.VirtualKey{}
	var allowedModels pq.StringArray
	err := db.conn.QueryRowContext(ctx,
		`SELECT id, user_id, name, key_hash, allowed_models, budget_limit, current_spend, created_at, revoked_at
		FROM virtual_keys WHERE id = $1`,
		id,
	).Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &allowedModels, &key.BudgetLimit, &key.CurrentSpend, &key.CreatedAt, &key.RevokedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual key: %w", err)
	}
	key.AllowedModels = allowedModels

	return key, nil
}

// RevokeVirtualKey revokes a virtual key
func (db *DB) RevokeVirtualKey(ctx context.Context, id string) error {
	_, err := db.conn.ExecContext(ctx,
		`UPDATE virtual_keys SET revoked_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to revoke virtual key: %w", err)
	}
	return nil
}

// UpdateVirtualKey updates a virtual key's basic info
func (db *DB) UpdateVirtualKey(ctx context.Context, id string, name *string, allowedModels []string, budgetLimit *float64) error {
	query := `UPDATE virtual_keys SET `
	args := []interface{}{}
	argCount := 1
	updates := []string{}

	if name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argCount))
		args = append(args, *name)
		argCount++
	}

	if allowedModels != nil {
		updates = append(updates, fmt.Sprintf("allowed_models = $%d", argCount))
		args = append(args, pq.Array(allowedModels))
		argCount++
	}

	if budgetLimit != nil {
		updates = append(updates, fmt.Sprintf("budget_limit = $%d", argCount))
		args = append(args, *budgetLimit)
		argCount++
	}

	if len(updates) == 0 {
		return nil
	}

	query += strings.Join(updates, ", ")
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	_, err := db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update virtual key: %w", err)
	}
	return nil
}

// UpdateKeySpend updates the current spend for a key
func (db *DB) UpdateKeySpend(ctx context.Context, keyID string, amount float64) error {
	_, err := db.conn.ExecContext(ctx,
		`UPDATE virtual_keys SET current_spend = current_spend + $1 WHERE id = $2`,
		amount, keyID,
	)
	if err != nil {
		return fmt.Errorf("failed to update key spend: %w", err)
	}
	return nil
}

// Daily Stats operations

// UpsertDailyStat upserts daily statistics
func (db *DB) UpsertDailyStat(ctx context.Context, keyID string, tokens int, cost float64) error {
	_, err := db.conn.ExecContext(ctx,
		`INSERT INTO daily_stats (id, key_id, date, total_tokens, total_cost)
		VALUES ($1, $2, CURRENT_DATE, $3, $4)
		ON CONFLICT (key_id, date) DO UPDATE SET
			total_tokens = daily_stats.total_tokens + EXCLUDED.total_tokens,
			total_cost = daily_stats.total_cost + EXCLUDED.total_cost`,
		uuid.New().String(), keyID, tokens, cost,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert daily stat: %w", err)
	}
	return nil
}

// GetDailyStats retrieves daily stats for a user within a date range
func (db *DB) GetDailyStats(ctx context.Context, userID string, startDate, endDate time.Time) ([]*models.DailyStat, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT ds.id, ds.key_id, ds.date, ds.total_tokens, ds.total_cost
		FROM daily_stats ds
		JOIN virtual_keys vk ON ds.key_id = vk.id
		WHERE vk.user_id = $1 AND ds.date >= $2 AND ds.date <= $3
		ORDER BY ds.date DESC`,
		userID, startDate, endDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}
	defer rows.Close()

	var stats []*models.DailyStat
	for rows.Next() {
		stat := &models.DailyStat{}
		err := rows.Scan(&stat.ID, &stat.KeyID, &stat.Date, &stat.TotalTokens, &stat.TotalCost)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily stat: %w", err)
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetUserOverview gets overview statistics for a user
func (db *DB) GetUserOverview(ctx context.Context, userID string) (*models.Overview, error) {
	overview := &models.Overview{}

	// Get total spend from virtual keys
	err := db.conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(current_spend), 0) FROM virtual_keys WHERE user_id = $1`,
		userID,
	).Scan(&overview.TotalSpend)
	if err != nil {
		return nil, fmt.Errorf("failed to get total spend: %w", err)
	}

	return overview, nil
}
