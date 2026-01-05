-- Migration: Account-level provider keys
-- Provider API keys are now stored at the user/account level, not per virtual key
-- Virtual keys only control: allowed models + budget limits

-- Create user_providers table for account-level API keys
CREATE TABLE IF NOT EXISTS user_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    provider provider_type NOT NULL,
    api_key_encrypted BYTEA NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, provider)
);

-- Migrate existing data from key_providers to user_providers
-- Take the most recent key for each user/provider combination
INSERT INTO user_providers (user_id, provider, api_key_encrypted)
SELECT DISTINCT ON (vk.user_id, kp.provider)
    vk.user_id,
    kp.provider,
    kp.real_key_encrypted
FROM key_providers kp
JOIN virtual_keys vk ON kp.key_id = vk.id
ORDER BY vk.user_id, kp.provider, kp.created_at DESC
ON CONFLICT (user_id, provider) DO NOTHING;

-- Drop the key_providers table (no longer needed)
DROP TABLE IF EXISTS key_providers;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_user_providers_user_id ON user_providers(user_id);
