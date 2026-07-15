ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS long_context_billing_applied BOOLEAN NOT NULL DEFAULT FALSE;
