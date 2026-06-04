package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const securityAuditAdvisoryLockKey int64 = 0x534143415544 // "SACAUD" prefix, scoped to this database.

type securityAuditRepository struct {
	db *sql.DB
}

func NewSecurityAuditRepository(db *sql.DB) service.SecurityAuditRepository {
	return &securityAuditRepository{db: db}
}

func (r *securityAuditRepository) Append(ctx context.Context, log *service.SecurityAuditLog) (*service.SecurityAuditLog, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	if log == nil {
		return nil, fmt.Errorf("security audit log is nil")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin security audit tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, securityAuditAdvisoryLockKey); err != nil {
		return nil, fmt.Errorf("lock security audit chain: %w", err)
	}

	var prevHash string
	err = tx.QueryRowContext(ctx, `SELECT entry_hash FROM security_audit_logs ORDER BY id DESC LIMIT 1`).Scan(&prevHash)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("load previous security audit hash: %w", err)
	}
	if err == sql.ErrNoRows {
		prevHash = ""
	}
	log.PrevHash = prevHash

	metadata, err := json.Marshal(nonNilMap(log.Metadata))
	if err != nil {
		return nil, fmt.Errorf("marshal security audit metadata: %w", err)
	}
	diffSummary, err := json.Marshal(nonNilMap(log.DiffSummary))
	if err != nil {
		return nil, fmt.Errorf("marshal security audit diff summary: %w", err)
	}
	log.Metadata = decodeJSONMap(metadata)
	log.DiffSummary = decodeJSONMap(diffSummary)

	var ipValue any
	if parsed := net.ParseIP(strings.TrimSpace(log.IP)); parsed != nil {
		ipValue = parsed.String()
		log.IP = parsed.String()
	} else {
		log.IP = ""
	}
	log.EntryHash = service.ComputeSecurityAuditEntryHash(log)

	row := tx.QueryRowContext(ctx, `
		INSERT INTO security_audit_logs (
			actor_type, actor_id, actor_label,
			subject_type, subject_id, subject_label,
			action, resource_type, resource_id, result, risk_level,
			request_id, ip_address, user_agent, endpoint, reason,
			metadata, diff_summary, prev_hash, entry_hash
		) VALUES (
			$1, $2, $3,
			$4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17::jsonb, $18::jsonb, $19, $20
		)
		RETURNING id, COALESCE(ip_address::text, ''), created_at`,
		log.ActorType, log.ActorID, log.ActorLabel,
		log.SubjectType, log.SubjectID, log.SubjectLabel,
		log.Action, log.ResourceType, log.ResourceID, log.Result, log.RiskLevel,
		log.RequestID, ipValue, log.UserAgent, log.Endpoint, log.Reason,
		string(metadata), string(diffSummary), log.PrevHash, log.EntryHash,
	)
	if err := row.Scan(&log.ID, &log.IP, &log.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert security audit log: %w", err)
	}
	log.EventType = log.Action

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit security audit tx: %w", err)
	}
	return log, nil
}

func (r *securityAuditRepository) List(ctx context.Context, filter service.SecurityAuditListFilter) ([]service.SecurityAuditLog, *pagination.PaginationResult, error) {
	if r == nil || r.db == nil {
		return nil, nil, fmt.Errorf("security audit repository is not configured")
	}

	page, pageSize := normalizeSecurityAuditPage(filter.Page, filter.PageSize)
	where, args := buildSecurityAuditWhere(filter)
	whereClause := strings.Join(where, " AND ")

	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM security_audit_logs WHERE %s`, whereClause)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count security audit logs: %w", err)
	}

	offset := (page - 1) * pageSize
	dataArgs := append(append([]any{}, args...), pageSize, offset)
	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	dataQuery := fmt.Sprintf(`
		SELECT
			id, actor_type, actor_id, actor_label,
			subject_type, subject_id, subject_label,
			action, resource_type, resource_id, result, risk_level,
			request_id, COALESCE(ip_address::text, ''), user_agent, endpoint, reason,
			metadata, diff_summary, prev_hash, entry_hash, created_at
		FROM security_audit_logs
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d OFFSET $%d`, whereClause, limitIdx, offsetIdx)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("list security audit logs: %w", err)
	}
	defer rows.Close()

	logs, err := scanSecurityAuditRows(rows)
	if err != nil {
		return nil, nil, err
	}

	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	if pages < 1 {
		pages = 1
	}
	return logs, &pagination.PaginationResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	}, nil
}

func (r *securityAuditRepository) ListForExport(ctx context.Context, filter service.SecurityAuditListFilter, limit int) ([]service.SecurityAuditLog, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	if limit <= 0 {
		limit = 50000
	}
	if limit > 50000 {
		limit = 50000
	}
	where, args := buildSecurityAuditWhere(filter)
	whereClause := strings.Join(where, " AND ")
	dataArgs := append(append([]any{}, args...), limit)
	limitIdx := len(args) + 1
	query := fmt.Sprintf(`
		SELECT
			id, actor_type, actor_id, actor_label,
			subject_type, subject_id, subject_label,
			action, resource_type, resource_id, result, risk_level,
			request_id, COALESCE(ip_address::text, ''), user_agent, endpoint, reason,
			metadata, diff_summary, prev_hash, entry_hash, created_at
		FROM security_audit_logs
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d`, whereClause, limitIdx)
	rows, err := r.db.QueryContext(ctx, query, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("list security audit export logs: %w", err)
	}
	defer rows.Close()
	return scanSecurityAuditRows(rows)
}

func (r *securityAuditRepository) ListForIntegrity(ctx context.Context) ([]service.SecurityAuditLog, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, actor_type, actor_id, actor_label,
			subject_type, subject_id, subject_label,
			action, resource_type, resource_id, result, risk_level,
			request_id, COALESCE(ip_address::text, ''), user_agent, endpoint, reason,
			metadata, diff_summary, prev_hash, entry_hash, created_at
		FROM security_audit_logs
		ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("load security audit integrity logs: %w", err)
	}
	defer rows.Close()
	return scanSecurityAuditRows(rows)
}

func (r *securityAuditRepository) ListIncidents(ctx context.Context, filter service.SecurityIncidentListFilter) ([]service.SecurityIncident, *pagination.PaginationResult, error) {
	if r == nil || r.db == nil {
		return nil, nil, fmt.Errorf("security audit repository is not configured")
	}
	page, pageSize := normalizeSecurityAuditPage(filter.Page, filter.PageSize)
	where, args := buildSecurityIncidentWhere(filter)
	whereClause := strings.Join(where, " AND ")
	total, err := countSecurityRows(ctx, r.db, "security_incidents", whereClause, args)
	if err != nil {
		return nil, nil, fmt.Errorf("count security incidents: %w", err)
	}
	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	dataArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	query := fmt.Sprintf(`
		SELECT id, incident_key, title, status, severity, subject_type, subject_id,
		       first_audit_log_id, last_audit_log_id, metadata, detected_at, resolved_at, created_at, updated_at
		FROM security_incidents
		WHERE %s
		ORDER BY detected_at DESC, id DESC
		LIMIT $%d OFFSET $%d`, whereClause, limitIdx, offsetIdx)
	rows, err := r.db.QueryContext(ctx, query, dataArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("list security incidents: %w", err)
	}
	defer rows.Close()
	items, err := scanSecurityIncidentRows(rows)
	if err != nil {
		return nil, nil, err
	}
	return items, securityPagination(total, page, pageSize), nil
}

func (r *securityAuditRepository) ListPolicyRules(ctx context.Context, filter service.SecurityPolicyRuleListFilter) ([]service.SecurityPolicyRule, *pagination.PaginationResult, error) {
	if r == nil || r.db == nil {
		return nil, nil, fmt.Errorf("security audit repository is not configured")
	}
	page, pageSize := normalizeSecurityAuditPage(filter.Page, filter.PageSize)
	where, args := buildSecurityPolicyWhere(filter)
	whereClause := strings.Join(where, " AND ")
	total, err := countSecurityRows(ctx, r.db, "security_policy_rules", whereClause, args)
	if err != nil {
		return nil, nil, fmt.Errorf("count security policy rules: %w", err)
	}
	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	dataArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	query := fmt.Sprintf(`
		SELECT id, name, code, description, enabled, severity, action, conditions, metadata, created_at, updated_at
		FROM security_policy_rules
		WHERE %s
		ORDER BY updated_at DESC, id DESC
		LIMIT $%d OFFSET $%d`, whereClause, limitIdx, offsetIdx)
	rows, err := r.db.QueryContext(ctx, query, dataArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("list security policy rules: %w", err)
	}
	defer rows.Close()
	items, err := scanSecurityPolicyRows(rows)
	if err != nil {
		return nil, nil, err
	}
	return items, securityPagination(total, page, pageSize), nil
}

func (r *securityAuditRepository) ListEnabledPolicyRules(ctx context.Context) ([]service.SecurityPolicyRule, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, code, description, enabled, severity, action, conditions, metadata, created_at, updated_at
		FROM security_policy_rules
		WHERE enabled = TRUE
		ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list enabled security policy rules: %w", err)
	}
	defer rows.Close()
	return scanSecurityPolicyRows(rows)
}

func (r *securityAuditRepository) CreatePolicyRule(ctx context.Context, input service.SecurityPolicyRuleInput) (*service.SecurityPolicyRule, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	conditions, err := json.Marshal(nonNilMap(input.Conditions))
	if err != nil {
		return nil, fmt.Errorf("marshal security policy conditions: %w", err)
	}
	metadata, err := json.Marshal(nonNilMap(input.Metadata))
	if err != nil {
		return nil, fmt.Errorf("marshal security policy metadata: %w", err)
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO security_policy_rules (name, code, description, enabled, severity, action, conditions, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb)
		RETURNING id, name, code, description, enabled, severity, action, conditions, metadata, created_at, updated_at`,
		input.Name, input.Code, input.Description, enabled, input.Severity, input.Action, string(conditions), string(metadata))
	return scanSecurityPolicyRow(row)
}

func (r *securityAuditRepository) UpdatePolicyRule(ctx context.Context, id int64, input service.SecurityPolicyRuleInput) (*service.SecurityPolicyRule, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	conditions, err := json.Marshal(nonNilMap(input.Conditions))
	if err != nil {
		return nil, fmt.Errorf("marshal security policy conditions: %w", err)
	}
	metadata, err := json.Marshal(nonNilMap(input.Metadata))
	if err != nil {
		return nil, fmt.Errorf("marshal security policy metadata: %w", err)
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE security_policy_rules
		SET name = $2, code = $3, description = $4, enabled = $5, severity = $6,
		    action = $7, conditions = $8::jsonb, metadata = $9::jsonb, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, code, description, enabled, severity, action, conditions, metadata, created_at, updated_at`,
		id, input.Name, input.Code, input.Description, enabled, input.Severity, input.Action, string(conditions), string(metadata))
	return scanSecurityPolicyRow(row)
}

func (r *securityAuditRepository) DeletePolicyRule(ctx context.Context, id int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("security audit repository is not configured")
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM security_policy_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete security policy rule: %w", err)
	}
	return nil
}

func (r *securityAuditRepository) ListSubjectLocks(ctx context.Context, filter service.SecuritySubjectLockListFilter) ([]service.SecuritySubjectLock, *pagination.PaginationResult, error) {
	if r == nil || r.db == nil {
		return nil, nil, fmt.Errorf("security audit repository is not configured")
	}
	page, pageSize := normalizeSecurityAuditPage(filter.Page, filter.PageSize)
	where, args := buildSecurityLockWhere(filter)
	whereClause := strings.Join(where, " AND ")
	total, err := countSecurityRows(ctx, r.db, "security_subject_locks", whereClause, args)
	if err != nil {
		return nil, nil, fmt.Errorf("count security locks: %w", err)
	}
	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	dataArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	query := fmt.Sprintf(`
		SELECT id, subject_type, subject_id, reason, status, locked_by_type, locked_by_id,
		       audit_log_id, metadata, locked_at, expires_at, unlocked_at, created_at, updated_at
		FROM security_subject_locks
		WHERE %s
		ORDER BY locked_at DESC, id DESC
		LIMIT $%d OFFSET $%d`, whereClause, limitIdx, offsetIdx)
	rows, err := r.db.QueryContext(ctx, query, dataArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("list security locks: %w", err)
	}
	defer rows.Close()
	items, err := scanSecurityLockRows(rows)
	if err != nil {
		return nil, nil, err
	}
	return items, securityPagination(total, page, pageSize), nil
}

func (r *securityAuditRepository) GetActiveSubjectLock(ctx context.Context, subjectType string, subjectID int64, now time.Time) (*service.SecuritySubjectLock, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, subject_type, subject_id, reason, status, locked_by_type, locked_by_id,
		       audit_log_id, metadata, locked_at, expires_at, unlocked_at, created_at, updated_at
		FROM security_subject_locks
		WHERE subject_type = $1
		  AND subject_id = $2
		  AND status = 'active'
		  AND (expires_at IS NULL OR expires_at > $3)
		ORDER BY locked_at DESC, id DESC
		LIMIT 1`, subjectType, subjectID, now)
	lock, err := scanSecurityLockRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active security lock: %w", err)
	}
	return lock, nil
}

func (r *securityAuditRepository) CreateSubjectLock(ctx context.Context, input service.SecuritySubjectLockInput) (*service.SecuritySubjectLock, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	metadata, err := json.Marshal(nonNilMap(input.Metadata))
	if err != nil {
		return nil, fmt.Errorf("marshal security lock metadata: %w", err)
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO security_subject_locks (
			subject_type, subject_id, reason, status, locked_by_type, locked_by_id,
			audit_log_id, metadata, expires_at
		) VALUES (
			$1, $2, $3, 'active', $4, $5, $6, $7::jsonb, $8
		)
		ON CONFLICT (subject_type, subject_id) WHERE status = 'active'
		DO UPDATE SET
			reason = EXCLUDED.reason,
			locked_by_type = EXCLUDED.locked_by_type,
			locked_by_id = EXCLUDED.locked_by_id,
			audit_log_id = EXCLUDED.audit_log_id,
			metadata = EXCLUDED.metadata,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
		RETURNING id, subject_type, subject_id, reason, status, locked_by_type, locked_by_id,
		          audit_log_id, metadata, locked_at, expires_at, unlocked_at, created_at, updated_at`,
		input.SubjectType, input.SubjectID, input.Reason, input.LockedByType, input.LockedByID, input.AuditLogID, string(metadata), input.ExpiresAt)
	return scanSecurityLockRow(row)
}

func (r *securityAuditRepository) UnlockSubjectLock(ctx context.Context, id int64, actorType string, actorID *int64, reason string) (*service.SecuritySubjectLock, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	metadata, err := json.Marshal(map[string]any{
		"unlock_actor_type": actorType,
		"unlock_actor_id":   actorID,
		"unlock_reason":     reason,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal security unlock metadata: %w", err)
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE security_subject_locks
		SET status = 'unlocked',
		    unlocked_at = NOW(),
		    metadata = COALESCE(metadata, '{}'::jsonb) || $2::jsonb,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, subject_type, subject_id, reason, status, locked_by_type, locked_by_id,
		          audit_log_id, metadata, locked_at, expires_at, unlocked_at, created_at, updated_at`,
		id, string(metadata))
	return scanSecurityLockRow(row)
}

func (r *securityAuditRepository) CreateAuditExport(ctx context.Context, item *service.SecurityAuditExport) (*service.SecurityAuditExport, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	if item == nil {
		return nil, fmt.Errorf("security audit export is nil")
	}
	filters, err := json.Marshal(nonNilMap(item.Filters))
	if err != nil {
		return nil, fmt.Errorf("marshal security audit export filters: %w", err)
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO security_audit_exports (
			export_key, requested_by_type, requested_by_id, status, filters,
			file_path, file_sha256, row_count, error_message, requested_at, completed_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5::jsonb,
			$6, $7, $8, $9, $10, $11, $12
		)
		RETURNING id, export_key, requested_by_type, requested_by_id, status, filters,
		          file_path, file_sha256, row_count, error_message,
		          requested_at, completed_at, expires_at, created_at, updated_at`,
		item.ExportKey, item.RequestedByType, item.RequestedByID, item.Status, string(filters),
		item.FilePath, item.FileSHA256, item.RowCount, item.ErrorMessage, item.RequestedAt, item.CompletedAt, item.ExpiresAt,
	)
	return scanSecurityAuditExportRow(row)
}

func (r *securityAuditRepository) UpdateAuditExportResult(ctx context.Context, id int64, status, filePath, fileSHA256 string, rowCount int64, errorMessage string, completedAt, expiresAt *time.Time) (*service.SecurityAuditExport, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE security_audit_exports
		SET status = $2,
		    file_path = $3,
		    file_sha256 = $4,
		    row_count = $5,
		    error_message = $6,
		    completed_at = $7,
		    expires_at = $8,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, export_key, requested_by_type, requested_by_id, status, filters,
		          file_path, file_sha256, row_count, error_message,
		          requested_at, completed_at, expires_at, created_at, updated_at`,
		id, status, filePath, fileSHA256, rowCount, errorMessage, completedAt, expiresAt,
	)
	item, err := scanSecurityAuditExportRow(row)
	if err == sql.ErrNoRows {
		return nil, infraerrors.NotFound("SECURITY_EXPORT_NOT_FOUND", "security audit export not found")
	}
	if err != nil {
		return nil, fmt.Errorf("update security audit export: %w", err)
	}
	return item, nil
}

func (r *securityAuditRepository) ListAuditExports(ctx context.Context, filter service.SecurityAuditExportListFilter) ([]service.SecurityAuditExport, *pagination.PaginationResult, error) {
	if r == nil || r.db == nil {
		return nil, nil, fmt.Errorf("security audit repository is not configured")
	}
	page, pageSize := normalizeSecurityAuditPage(filter.Page, filter.PageSize)
	where, args := buildSecurityAuditExportWhere(filter)
	whereClause := strings.Join(where, " AND ")
	total, err := countSecurityRows(ctx, r.db, "security_audit_exports", whereClause, args)
	if err != nil {
		return nil, nil, fmt.Errorf("count security audit exports: %w", err)
	}
	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	dataArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	query := fmt.Sprintf(`
		SELECT id, export_key, requested_by_type, requested_by_id, status, filters,
		       file_path, file_sha256, row_count, error_message,
		       requested_at, completed_at, expires_at, created_at, updated_at
		FROM security_audit_exports
		WHERE %s
		ORDER BY requested_at DESC, id DESC
		LIMIT $%d OFFSET $%d`, whereClause, limitIdx, offsetIdx)
	rows, err := r.db.QueryContext(ctx, query, dataArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("list security audit exports: %w", err)
	}
	defer rows.Close()
	items, err := scanSecurityAuditExportRows(rows)
	if err != nil {
		return nil, nil, err
	}
	return items, securityPagination(total, page, pageSize), nil
}

func (r *securityAuditRepository) GetAuditExport(ctx context.Context, id int64) (*service.SecurityAuditExport, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("security audit repository is not configured")
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, export_key, requested_by_type, requested_by_id, status, filters,
		       file_path, file_sha256, row_count, error_message,
		       requested_at, completed_at, expires_at, created_at, updated_at
		FROM security_audit_exports
		WHERE id = $1`, id)
	item, err := scanSecurityAuditExportRow(row)
	if err == sql.ErrNoRows {
		return nil, infraerrors.NotFound("SECURITY_EXPORT_NOT_FOUND", "security audit export not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get security audit export: %w", err)
	}
	return item, nil
}

func buildSecurityAuditWhere(filter service.SecurityAuditListFilter) ([]string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 12)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.ActorType = strings.TrimSpace(filter.ActorType); filter.ActorType != "" {
		where = append(where, "actor_type = "+nextArg(filter.ActorType))
	}
	if filter.ActorID != nil {
		where = append(where, "actor_id = "+nextArg(*filter.ActorID))
	}
	if filter.SubjectType = strings.TrimSpace(filter.SubjectType); filter.SubjectType != "" {
		where = append(where, "subject_type = "+nextArg(filter.SubjectType))
	}
	if filter.SubjectID != nil {
		where = append(where, "subject_id = "+nextArg(*filter.SubjectID))
	}
	if filter.Action = strings.TrimSpace(filter.Action); filter.Action != "" {
		where = append(where, "action = "+nextArg(filter.Action))
	}
	if filter.Result = strings.TrimSpace(filter.Result); filter.Result != "" {
		where = append(where, "result = "+nextArg(strings.ToLower(filter.Result)))
	}
	if filter.RiskLevel = strings.TrimSpace(filter.RiskLevel); filter.RiskLevel != "" {
		where = append(where, "risk_level = "+nextArg(strings.ToLower(filter.RiskLevel)))
	}
	if filter.RequestID = strings.TrimSpace(filter.RequestID); filter.RequestID != "" {
		where = append(where, "request_id = "+nextArg(filter.RequestID))
	}
	if filter.StartTime != nil {
		where = append(where, "created_at >= "+nextArg(*filter.StartTime))
	}
	if filter.EndTime != nil {
		where = append(where, "created_at < "+nextArg(*filter.EndTime))
	}
	if filter.Query = strings.TrimSpace(filter.Query); filter.Query != "" {
		pattern := "%" + escapeLike(filter.Query) + "%"
		placeholder := nextArg(pattern)
		where = append(where, "(actor_label ILIKE "+placeholder+" OR subject_label ILIKE "+placeholder+" OR resource_id ILIKE "+placeholder+" OR reason ILIKE "+placeholder+")")
	}
	return where, args
}

func buildSecurityAuditExportWhere(filter service.SecurityAuditExportListFilter) ([]string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 8)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Status = strings.TrimSpace(filter.Status); filter.Status != "" {
		where = append(where, "status = "+nextArg(filter.Status))
	}
	if filter.RequestedByType = strings.TrimSpace(filter.RequestedByType); filter.RequestedByType != "" {
		where = append(where, "requested_by_type = "+nextArg(filter.RequestedByType))
	}
	if filter.RequestedByID != nil {
		where = append(where, "requested_by_id = "+nextArg(*filter.RequestedByID))
	}
	if filter.Query = strings.TrimSpace(filter.Query); filter.Query != "" {
		pattern := "%" + escapeLike(filter.Query) + "%"
		placeholder := nextArg(pattern)
		where = append(where, "(export_key ILIKE "+placeholder+" OR status ILIKE "+placeholder+" OR error_message ILIKE "+placeholder+")")
	}
	return where, args
}

func buildSecurityIncidentWhere(filter service.SecurityIncidentListFilter) ([]string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 8)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Status = strings.TrimSpace(filter.Status); filter.Status != "" {
		where = append(where, "status = "+nextArg(filter.Status))
	}
	if filter.Severity = strings.TrimSpace(filter.Severity); filter.Severity != "" {
		where = append(where, "severity = "+nextArg(strings.ToLower(filter.Severity)))
	}
	if filter.SubjectType = strings.TrimSpace(filter.SubjectType); filter.SubjectType != "" {
		where = append(where, "subject_type = "+nextArg(filter.SubjectType))
	}
	if filter.SubjectID != nil {
		where = append(where, "subject_id = "+nextArg(*filter.SubjectID))
	}
	if filter.Query = strings.TrimSpace(filter.Query); filter.Query != "" {
		pattern := "%" + escapeLike(filter.Query) + "%"
		placeholder := nextArg(pattern)
		where = append(where, "(incident_key ILIKE "+placeholder+" OR title ILIKE "+placeholder+")")
	}
	return where, args
}

func buildSecurityPolicyWhere(filter service.SecurityPolicyRuleListFilter) ([]string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 8)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Enabled != nil {
		where = append(where, "enabled = "+nextArg(*filter.Enabled))
	}
	if filter.Severity = strings.TrimSpace(filter.Severity); filter.Severity != "" {
		where = append(where, "severity = "+nextArg(strings.ToLower(filter.Severity)))
	}
	if filter.Action = strings.TrimSpace(filter.Action); filter.Action != "" {
		where = append(where, "action = "+nextArg(strings.ToLower(filter.Action)))
	}
	if filter.Query = strings.TrimSpace(filter.Query); filter.Query != "" {
		pattern := "%" + escapeLike(filter.Query) + "%"
		placeholder := nextArg(pattern)
		where = append(where, "(name ILIKE "+placeholder+" OR code ILIKE "+placeholder+" OR description ILIKE "+placeholder+")")
	}
	return where, args
}

func buildSecurityLockWhere(filter service.SecuritySubjectLockListFilter) ([]string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 8)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Status = strings.TrimSpace(filter.Status); filter.Status != "" {
		where = append(where, "status = "+nextArg(filter.Status))
	}
	if filter.SubjectType = strings.TrimSpace(filter.SubjectType); filter.SubjectType != "" {
		where = append(where, "subject_type = "+nextArg(filter.SubjectType))
	}
	if filter.SubjectID != nil {
		where = append(where, "subject_id = "+nextArg(*filter.SubjectID))
	}
	if filter.Query = strings.TrimSpace(filter.Query); filter.Query != "" {
		pattern := "%" + escapeLike(filter.Query) + "%"
		placeholder := nextArg(pattern)
		where = append(where, "(subject_type ILIKE "+placeholder+" OR reason ILIKE "+placeholder+" OR locked_by_type ILIKE "+placeholder+")")
	}
	return where, args
}

func scanSecurityAuditRows(rows *sql.Rows) ([]service.SecurityAuditLog, error) {
	logs := []service.SecurityAuditLog{}
	for rows.Next() {
		var log service.SecurityAuditLog
		var metadataRaw, diffRaw []byte
		if err := rows.Scan(
			&log.ID, &log.ActorType, &log.ActorID, &log.ActorLabel,
			&log.SubjectType, &log.SubjectID, &log.SubjectLabel,
			&log.Action, &log.ResourceType, &log.ResourceID, &log.Result, &log.RiskLevel,
			&log.RequestID, &log.IP, &log.UserAgent, &log.Endpoint, &log.Reason,
			&metadataRaw, &diffRaw, &log.PrevHash, &log.EntryHash, &log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan security audit log: %w", err)
		}
		log.EventType = log.Action
		log.Metadata = decodeJSONMap(metadataRaw)
		log.DiffSummary = decodeJSONMap(diffRaw)
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate security audit logs: %w", err)
	}
	return logs, nil
}

func scanSecurityIncidentRows(rows *sql.Rows) ([]service.SecurityIncident, error) {
	items := []service.SecurityIncident{}
	for rows.Next() {
		var item service.SecurityIncident
		var metadataRaw []byte
		if err := rows.Scan(
			&item.ID, &item.IncidentKey, &item.Title, &item.Status, &item.Severity,
			&item.SubjectType, &item.SubjectID, &item.FirstAuditLogID, &item.LastAuditLogID,
			&metadataRaw, &item.DetectedAt, &item.ResolvedAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan security incident: %w", err)
		}
		item.Metadata = decodeJSONMap(metadataRaw)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate security incidents: %w", err)
	}
	return items, nil
}

type sqlScanner interface {
	Scan(dest ...any) error
}

func scanSecurityPolicyRow(row sqlScanner) (*service.SecurityPolicyRule, error) {
	var item service.SecurityPolicyRule
	var conditionsRaw, metadataRaw []byte
	if err := row.Scan(
		&item.ID, &item.Name, &item.Code, &item.Description, &item.Enabled,
		&item.Severity, &item.Action, &conditionsRaw, &metadataRaw, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Conditions = decodeJSONMap(conditionsRaw)
	item.Metadata = decodeJSONMap(metadataRaw)
	return &item, nil
}

func scanSecurityPolicyRows(rows *sql.Rows) ([]service.SecurityPolicyRule, error) {
	items := []service.SecurityPolicyRule{}
	for rows.Next() {
		item, err := scanSecurityPolicyRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan security policy rule: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate security policy rules: %w", err)
	}
	return items, nil
}

func scanSecurityLockRow(row sqlScanner) (*service.SecuritySubjectLock, error) {
	var item service.SecuritySubjectLock
	var metadataRaw []byte
	if err := row.Scan(
		&item.ID, &item.SubjectType, &item.SubjectID, &item.Reason, &item.Status,
		&item.LockedByType, &item.LockedByID, &item.AuditLogID, &metadataRaw,
		&item.LockedAt, &item.ExpiresAt, &item.UnlockedAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = decodeJSONMap(metadataRaw)
	return &item, nil
}

func scanSecurityLockRows(rows *sql.Rows) ([]service.SecuritySubjectLock, error) {
	items := []service.SecuritySubjectLock{}
	for rows.Next() {
		item, err := scanSecurityLockRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan security lock: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate security locks: %w", err)
	}
	return items, nil
}

func scanSecurityAuditExportRow(row sqlScanner) (*service.SecurityAuditExport, error) {
	var item service.SecurityAuditExport
	var filtersRaw []byte
	if err := row.Scan(
		&item.ID, &item.ExportKey, &item.RequestedByType, &item.RequestedByID, &item.Status, &filtersRaw,
		&item.FilePath, &item.FileSHA256, &item.RowCount, &item.ErrorMessage,
		&item.RequestedAt, &item.CompletedAt, &item.ExpiresAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Filters = decodeJSONMap(filtersRaw)
	item.DownloadAvailable = securityAuditExportDownloadAvailable(item)
	return &item, nil
}

func scanSecurityAuditExportRows(rows *sql.Rows) ([]service.SecurityAuditExport, error) {
	items := []service.SecurityAuditExport{}
	for rows.Next() {
		item, err := scanSecurityAuditExportRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan security audit export: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate security audit exports: %w", err)
	}
	return items, nil
}

func securityAuditExportDownloadAvailable(item service.SecurityAuditExport) bool {
	if item.Status != service.SecurityAuditExportStatusCompleted || strings.TrimSpace(item.FilePath) == "" {
		return false
	}
	if item.ExpiresAt != nil && time.Now().After(*item.ExpiresAt) {
		return false
	}
	return true
}

func countSecurityRows(ctx context.Context, db *sql.DB, table, whereClause string, args []any) (int64, error) {
	var total int64
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, table, whereClause)
	if err := db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func securityPagination(total int64, page, pageSize int) *pagination.PaginationResult {
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	if pages < 1 {
		pages = 1
	}
	return &pagination.PaginationResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	}
}

func normalizeSecurityAuditPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func nonNilMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	return input
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}
