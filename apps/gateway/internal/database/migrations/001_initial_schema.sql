-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Provider enum
DO $$ BEGIN
    CREATE TYPE provider_type AS ENUM ('openai', 'anthropic');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Virtual Keys table
CREATE TABLE IF NOT EXISTS virtual_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    provider provider_type NOT NULL,
    real_key_encrypted BYTEA NOT NULL,
    budget_limit DECIMAL(10,2),
    current_spend DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP
);

-- Daily Stats table
CREATE TABLE IF NOT EXISTS daily_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id UUID REFERENCES virtual_keys(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    total_tokens INTEGER DEFAULT 0,
    total_cost DECIMAL(10,4) DEFAULT 0,
    UNIQUE(key_id, date)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_virtual_keys_user_id ON virtual_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_virtual_keys_key_hash ON virtual_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_daily_stats_key_id ON daily_stats(key_id);
CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_stats(date);
