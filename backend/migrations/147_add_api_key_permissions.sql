-- Migration: Add API key-level model and endpoint permissions.
-- Empty allow-lists keep backward-compatible inherited group/channel behavior.

ALTER TABLE api_keys
	ADD COLUMN IF NOT EXISTS permission_mode VARCHAR(20) NOT NULL DEFAULT 'inherit';

ALTER TABLE api_keys
	ADD COLUMN IF NOT EXISTS allowed_models JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE api_keys
	ADD COLUMN IF NOT EXISTS allowed_endpoints JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE api_keys
	ADD COLUMN IF NOT EXISTS permission_updated_at TIMESTAMPTZ;

ALTER TABLE api_keys
	DROP CONSTRAINT IF EXISTS api_keys_permission_mode_check;

ALTER TABLE api_keys
	ADD CONSTRAINT api_keys_permission_mode_check
	CHECK (permission_mode IN ('inherit', 'restrict'));

CREATE INDEX IF NOT EXISTS idx_api_keys_permission_mode
	ON api_keys(permission_mode)
	WHERE deleted_at IS NULL;

COMMENT ON COLUMN api_keys.permission_mode IS 'API key permission mode: inherit or restrict';
COMMENT ON COLUMN api_keys.allowed_models IS 'API key model allow-list. Empty means inherit current group/channel models';
COMMENT ON COLUMN api_keys.allowed_endpoints IS 'API key endpoint allow-list. Empty means inherit current group/channel endpoints';
COMMENT ON COLUMN api_keys.permission_updated_at IS 'Last time API key permissions were explicitly updated';
