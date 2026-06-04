CREATE TABLE IF NOT EXISTS user_totp_recovery_codes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_totp_recovery_codes_user_hash
    ON user_totp_recovery_codes(user_id, code_hash);

CREATE INDEX IF NOT EXISTS idx_user_totp_recovery_codes_user_available
    ON user_totp_recovery_codes(user_id, used_at)
    WHERE used_at IS NULL;
