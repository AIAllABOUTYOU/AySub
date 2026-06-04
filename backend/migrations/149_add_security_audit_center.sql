CREATE TABLE IF NOT EXISTS security_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    actor_type TEXT NOT NULL DEFAULT '',
    actor_id BIGINT,
    actor_label TEXT NOT NULL DEFAULT '',
    subject_type TEXT NOT NULL DEFAULT '',
    subject_id BIGINT,
    subject_label TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL DEFAULT '',
    resource_id TEXT NOT NULL DEFAULT '',
    result TEXT NOT NULL,
    risk_level TEXT NOT NULL,
    request_id TEXT NOT NULL DEFAULT '',
    ip_address INET,
    user_agent TEXT NOT NULL DEFAULT '',
    endpoint TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    diff_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    prev_hash TEXT NOT NULL DEFAULT '',
    entry_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_security_audit_logs_entry_hash
    ON security_audit_logs (entry_hash);

CREATE INDEX IF NOT EXISTS idx_security_audit_logs_created_at
    ON security_audit_logs (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_security_audit_logs_subject
    ON security_audit_logs (subject_type, subject_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_security_audit_logs_actor
    ON security_audit_logs (actor_type, actor_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_security_audit_logs_action_result
    ON security_audit_logs (action, result, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_security_audit_logs_request_id
    ON security_audit_logs (request_id)
    WHERE request_id <> '';

CREATE TABLE IF NOT EXISTS security_incidents (
    id BIGSERIAL PRIMARY KEY,
    incident_key TEXT NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    severity TEXT NOT NULL DEFAULT 'medium',
    subject_type TEXT NOT NULL DEFAULT '',
    subject_id BIGINT,
    first_audit_log_id BIGINT REFERENCES security_audit_logs(id) ON DELETE SET NULL,
    last_audit_log_id BIGINT REFERENCES security_audit_logs(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_security_incidents_incident_key
    ON security_incidents (incident_key);

CREATE INDEX IF NOT EXISTS idx_security_incidents_status_severity
    ON security_incidents (status, severity, detected_at DESC);

CREATE INDEX IF NOT EXISTS idx_security_incidents_subject
    ON security_incidents (subject_type, subject_id, detected_at DESC);

CREATE TABLE IF NOT EXISTS security_policy_rules (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    severity TEXT NOT NULL DEFAULT 'medium',
    action TEXT NOT NULL DEFAULT 'audit',
    conditions JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_security_policy_rules_code
    ON security_policy_rules (code);

CREATE INDEX IF NOT EXISTS idx_security_policy_rules_enabled
    ON security_policy_rules (enabled, severity);

CREATE TABLE IF NOT EXISTS security_subject_locks (
    id BIGSERIAL PRIMARY KEY,
    subject_type TEXT NOT NULL,
    subject_id BIGINT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    locked_by_type TEXT NOT NULL DEFAULT '',
    locked_by_id BIGINT,
    audit_log_id BIGINT REFERENCES security_audit_logs(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    locked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    unlocked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_security_subject_locks_active_subject
    ON security_subject_locks (subject_type, subject_id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_security_subject_locks_status_expiry
    ON security_subject_locks (status, expires_at);

CREATE TABLE IF NOT EXISTS security_audit_exports (
    id BIGSERIAL PRIMARY KEY,
    export_key TEXT NOT NULL,
    requested_by_type TEXT NOT NULL DEFAULT '',
    requested_by_id BIGINT,
    status TEXT NOT NULL DEFAULT 'pending',
    filters JSONB NOT NULL DEFAULT '{}'::jsonb,
    file_path TEXT NOT NULL DEFAULT '',
    file_sha256 TEXT NOT NULL DEFAULT '',
    row_count BIGINT NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_security_audit_exports_export_key
    ON security_audit_exports (export_key);

CREATE INDEX IF NOT EXISTS idx_security_audit_exports_requested_by
    ON security_audit_exports (requested_by_type, requested_by_id, requested_at DESC);

CREATE INDEX IF NOT EXISTS idx_security_audit_exports_status
    ON security_audit_exports (status, requested_at DESC);
