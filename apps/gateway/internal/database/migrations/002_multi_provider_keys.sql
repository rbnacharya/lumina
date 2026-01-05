-- Migration: Multi-provider virtual keys
-- A single virtual key can now have multiple provider credentials and model access control

-- Add allowed_models column to virtual_keys (array of model patterns)
ALTER TABLE virtual_keys ADD COLUMN IF NOT EXISTS allowed_models TEXT[] DEFAULT '{}';

-- Create key_providers table for storing multiple provider credentials per key
CREATE TABLE IF NOT EXISTS key_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id UUID REFERENCES virtual_keys(id) ON DELETE CASCADE,
    provider provider_type NOT NULL,
    real_key_encrypted BYTEA NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(key_id, provider)
);

-- Migrate existing data: move real_key_encrypted to key_providers table
INSERT INTO key_providers (key_id, provider, real_key_encrypted)
SELECT id, provider, real_key_encrypted
FROM virtual_keys
WHERE real_key_encrypted IS NOT NULL
ON CONFLICT (key_id, provider) DO NOTHING;

-- Drop old columns from virtual_keys (no longer needed)
ALTER TABLE virtual_keys DROP COLUMN IF EXISTS provider;
ALTER TABLE virtual_keys DROP COLUMN IF EXISTS real_key_encrypted;

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_key_providers_key_id ON key_providers(key_id);
