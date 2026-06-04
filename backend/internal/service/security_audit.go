package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/google/uuid"
)

const (
	SecurityAuditResultSuccess = "success"
	SecurityAuditResultDenied  = "denied"
	SecurityAuditResultFailure = "failure"

	SecurityAuditRiskLow      = "low"
	SecurityAuditRiskMedium   = "medium"
	SecurityAuditRiskHigh     = "high"
	SecurityAuditRiskCritical = "critical"

	SecurityPolicyActionObserve       = "observe"
	SecurityPolicyActionBlock         = "block"
	SecurityPolicyActionChallenge     = "challenge"
	SecurityPolicyActionDisableAPIKey = "disable_api_key"
	SecurityPolicyActionDisableUser   = "disable_user"
	SecurityPolicyActionTemporaryLock = "temporary_lock"
	SecurityPolicyActionNotifyAdmin   = "notify_admin"
	SecurityPolicyActionNotifyUser    = "notify_user"

	SecurityLockStatusActive   = "active"
	SecurityLockStatusUnlocked = "unlocked"

	SecurityAuditExportStatusPending   = "pending"
	SecurityAuditExportStatusCompleted = "completed"
	SecurityAuditExportStatusFailed    = "failed"

	defaultSecurityAuditExportLimit = 50000
)

var ErrSecuritySubjectLocked = infraerrors.Forbidden("SECURITY_SUBJECT_LOCKED", "subject is locked by security policy")

type SecurityAuditLog struct {
	ID           int64          `json:"id"`
	EventType    string         `json:"event_type"`
	ActorType    string         `json:"actor_type"`
	ActorID      *int64         `json:"actor_id,omitempty"`
	ActorLabel   string         `json:"actor_label,omitempty"`
	SubjectType  string         `json:"subject_type"`
	SubjectID    *int64         `json:"subject_id,omitempty"`
	SubjectLabel string         `json:"subject_label,omitempty"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	Action       string         `json:"action"`
	Result       string         `json:"result"`
	RiskLevel    string         `json:"risk_level"`
	RequestID    string         `json:"request_id"`
	IP           string         `json:"ip"`
	UserAgent    string         `json:"user_agent"`
	Endpoint     string         `json:"endpoint"`
	Reason       string         `json:"reason"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	DiffSummary  map[string]any `json:"diff_summary,omitempty"`
	PrevHash     string         `json:"prev_hash"`
	EntryHash    string         `json:"entry_hash"`
	CreatedAt    time.Time      `json:"created_at"`
}

type SecurityAuditCreateInput struct {
	ActorType    string
	ActorID      *int64
	ActorLabel   string
	SubjectType  string
	SubjectID    *int64
	SubjectLabel string
	ResourceType string
	ResourceID   string
	Action       string
	Result       string
	RiskLevel    string
	RequestID    string
	IP           string
	UserAgent    string
	Endpoint     string
	Reason       string
	Metadata     map[string]any
	DiffSummary  map[string]any
}

type SecurityAuditListFilter struct {
	Page        int
	PageSize    int
	ActorType   string
	ActorID     *int64
	SubjectType string
	SubjectID   *int64
	Action      string
	Result      string
	RiskLevel   string
	RequestID   string
	Query       string
	StartTime   *time.Time
	EndTime     *time.Time
}

type SecurityAuditIntegrityResult struct {
	Valid            bool   `json:"valid"`
	Checked          int64  `json:"checked"`
	BrokenAtID       *int64 `json:"broken_at_id,omitempty"`
	ExpectedPrevHash string `json:"expected_prev_hash,omitempty"`
	ActualPrevHash   string `json:"actual_prev_hash,omitempty"`
	ExpectedHash     string `json:"expected_hash,omitempty"`
	ActualHash       string `json:"actual_hash,omitempty"`
}

type SecurityAuditExport struct {
	ID                int64          `json:"id"`
	ExportKey         string         `json:"export_key"`
	RequestedByType   string         `json:"requested_by_type"`
	RequestedByID     *int64         `json:"requested_by_id,omitempty"`
	Status            string         `json:"status"`
	Filters           map[string]any `json:"filters,omitempty"`
	FilePath          string         `json:"-"`
	FileSHA256        string         `json:"file_sha256"`
	RowCount          int64          `json:"row_count"`
	ErrorMessage      string         `json:"error_message,omitempty"`
	RequestedAt       time.Time      `json:"requested_at"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
	ExpiresAt         *time.Time     `json:"expires_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DownloadAvailable bool           `json:"download_available"`
}

type SecurityAuditExportListFilter struct {
	Page            int
	PageSize        int
	Status          string
	RequestedByType string
	RequestedByID   *int64
	Query           string
}

type SecurityAuditExportCreateInput struct {
	Filter          SecurityAuditListFilter
	RequestedByType string
	RequestedByID   *int64
}

type SecurityAuditExportFile struct {
	Path        string
	Filename    string
	ContentType string
}

type SecurityIncident struct {
	ID              int64          `json:"id"`
	IncidentKey     string         `json:"incident_key"`
	Title           string         `json:"title"`
	Status          string         `json:"status"`
	Severity        string         `json:"severity"`
	SubjectType     string         `json:"subject_type"`
	SubjectID       *int64         `json:"subject_id,omitempty"`
	FirstAuditLogID *int64         `json:"first_audit_log_id,omitempty"`
	LastAuditLogID  *int64         `json:"last_audit_log_id,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	DetectedAt      time.Time      `json:"detected_at"`
	ResolvedAt      *time.Time     `json:"resolved_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type SecurityIncidentListFilter struct {
	Page        int
	PageSize    int
	Status      string
	Severity    string
	SubjectType string
	SubjectID   *int64
	Query       string
}

type SecurityPolicyRule struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	Code        string         `json:"code"`
	Description string         `json:"description"`
	Enabled     bool           `json:"enabled"`
	Severity    string         `json:"severity"`
	Action      string         `json:"action"`
	Conditions  map[string]any `json:"conditions,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type SecurityPolicyRuleInput struct {
	Name        string         `json:"name"`
	Code        string         `json:"code"`
	Description string         `json:"description"`
	Enabled     *bool          `json:"enabled"`
	Severity    string         `json:"severity"`
	Action      string         `json:"action"`
	Conditions  map[string]any `json:"conditions"`
	Metadata    map[string]any `json:"metadata"`
}

type SecurityPolicyRuleListFilter struct {
	Page     int
	PageSize int
	Enabled  *bool
	Severity string
	Action   string
	Query    string
}

type SecuritySubjectLock struct {
	ID           int64          `json:"id"`
	SubjectType  string         `json:"subject_type"`
	SubjectID    int64          `json:"subject_id"`
	Reason       string         `json:"reason"`
	Status       string         `json:"status"`
	LockedByType string         `json:"locked_by_type"`
	LockedByID   *int64         `json:"locked_by_id,omitempty"`
	AuditLogID   *int64         `json:"audit_log_id,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	LockedAt     time.Time      `json:"locked_at"`
	ExpiresAt    *time.Time     `json:"expires_at,omitempty"`
	UnlockedAt   *time.Time     `json:"unlocked_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type SecuritySubjectLockInput struct {
	SubjectType  string         `json:"subject_type"`
	SubjectID    int64          `json:"subject_id"`
	Reason       string         `json:"reason"`
	LockedByType string         `json:"locked_by_type"`
	LockedByID   *int64         `json:"locked_by_id"`
	AuditLogID   *int64         `json:"audit_log_id"`
	Metadata     map[string]any `json:"metadata"`
	ExpiresAt    *time.Time     `json:"expires_at"`
}

type SecuritySubjectLockListFilter struct {
	Page        int
	PageSize    int
	Status      string
	SubjectType string
	SubjectID   *int64
	Query       string
}

type SecurityPolicyEvaluationInput struct {
	UserID     int64
	APIKeyID   int64
	GroupID    *int64
	Endpoint   string
	Model      string
	IP         string
	UserAgent  string
	Metadata   map[string]any
	OccurredAt time.Time
}

type SecurityPolicyDecision struct {
	Matched     bool                 `json:"matched"`
	Blocked     bool                 `json:"blocked"`
	Action      string               `json:"action"`
	Reason      string               `json:"reason"`
	RuleID      *int64               `json:"rule_id,omitempty"`
	RuleCode    string               `json:"rule_code,omitempty"`
	RiskLevel   string               `json:"risk_level"`
	SubjectLock *SecuritySubjectLock `json:"subject_lock,omitempty"`
}

type SecurityAuditRepository interface {
	Append(ctx context.Context, log *SecurityAuditLog) (*SecurityAuditLog, error)
	List(ctx context.Context, filter SecurityAuditListFilter) ([]SecurityAuditLog, *pagination.PaginationResult, error)
	ListForExport(ctx context.Context, filter SecurityAuditListFilter, limit int) ([]SecurityAuditLog, error)
	ListForIntegrity(ctx context.Context) ([]SecurityAuditLog, error)
	ListIncidents(ctx context.Context, filter SecurityIncidentListFilter) ([]SecurityIncident, *pagination.PaginationResult, error)
	ListPolicyRules(ctx context.Context, filter SecurityPolicyRuleListFilter) ([]SecurityPolicyRule, *pagination.PaginationResult, error)
	ListEnabledPolicyRules(ctx context.Context) ([]SecurityPolicyRule, error)
	CreatePolicyRule(ctx context.Context, input SecurityPolicyRuleInput) (*SecurityPolicyRule, error)
	UpdatePolicyRule(ctx context.Context, id int64, input SecurityPolicyRuleInput) (*SecurityPolicyRule, error)
	DeletePolicyRule(ctx context.Context, id int64) error
	ListSubjectLocks(ctx context.Context, filter SecuritySubjectLockListFilter) ([]SecuritySubjectLock, *pagination.PaginationResult, error)
	GetActiveSubjectLock(ctx context.Context, subjectType string, subjectID int64, now time.Time) (*SecuritySubjectLock, error)
	CreateSubjectLock(ctx context.Context, input SecuritySubjectLockInput) (*SecuritySubjectLock, error)
	UnlockSubjectLock(ctx context.Context, id int64, actorType string, actorID *int64, reason string) (*SecuritySubjectLock, error)
	CreateAuditExport(ctx context.Context, item *SecurityAuditExport) (*SecurityAuditExport, error)
	UpdateAuditExportResult(ctx context.Context, id int64, status, filePath, fileSHA256 string, rowCount int64, errorMessage string, completedAt, expiresAt *time.Time) (*SecurityAuditExport, error)
	ListAuditExports(ctx context.Context, filter SecurityAuditExportListFilter) ([]SecurityAuditExport, *pagination.PaginationResult, error)
	GetAuditExport(ctx context.Context, id int64) (*SecurityAuditExport, error)
}

type SecurityAuditService struct {
	repo SecurityAuditRepository
}

func NewSecurityAuditService(repo SecurityAuditRepository) *SecurityAuditService {
	return &SecurityAuditService{repo: repo}
}

func (s *SecurityAuditService) Create(ctx context.Context, input SecurityAuditCreateInput) (*SecurityAuditLog, error) {
	return s.CreateAuditLog(ctx, input)
}

func (s *SecurityAuditService) CreateAuditLog(ctx context.Context, input SecurityAuditCreateInput) (*SecurityAuditLog, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("security audit service is not configured")
	}
	log := normalizeSecurityAuditInput(input)
	return s.repo.Append(ctx, &log)
}

func (s *SecurityAuditService) List(ctx context.Context, filter SecurityAuditListFilter) ([]SecurityAuditLog, *pagination.PaginationResult, error) {
	return s.ListAuditLogs(ctx, filter)
}

func (s *SecurityAuditService) ListAuditLogs(ctx context.Context, filter SecurityAuditListFilter) ([]SecurityAuditLog, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("security audit service is not configured")
	}
	filter.Page, filter.PageSize = normalizeSecurityAuditPagination(filter.Page, filter.PageSize)
	return s.repo.List(ctx, filter)
}

func (s *SecurityAuditService) ListUserEvents(ctx context.Context, userID int64, filter SecurityAuditListFilter) ([]SecurityAuditLog, *pagination.PaginationResult, error) {
	filter.SubjectType = "user"
	filter.SubjectID = &userID
	return s.List(ctx, filter)
}

func (s *SecurityAuditService) CheckIntegrity(ctx context.Context) (*SecurityAuditIntegrityResult, error) {
	return s.IntegrityCheck(ctx)
}

func (s *SecurityAuditService) IntegrityCheck(ctx context.Context) (*SecurityAuditIntegrityResult, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("security audit service is not configured")
	}
	logs, err := s.repo.ListForIntegrity(ctx)
	if err != nil {
		return nil, err
	}
	result := &SecurityAuditIntegrityResult{Valid: true, Checked: int64(len(logs))}
	prevHash := ""
	for i := range logs {
		log := logs[i]
		if log.PrevHash != prevHash {
			result.Valid = false
			result.BrokenAtID = &log.ID
			result.ExpectedPrevHash = prevHash
			result.ActualPrevHash = log.PrevHash
			return result, nil
		}
		expected := ComputeSecurityAuditEntryHash(&log)
		if log.EntryHash != expected {
			result.Valid = false
			result.BrokenAtID = &log.ID
			result.ExpectedHash = expected
			result.ActualHash = log.EntryHash
			return result, nil
		}
		prevHash = log.EntryHash
	}
	return result, nil
}

func (s *SecurityAuditService) CreateAuditExport(ctx context.Context, input SecurityAuditExportCreateInput) (*SecurityAuditExport, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("security audit service is not configured")
	}
	input.Filter.Page = 1
	input.Filter.PageSize = defaultSecurityAuditExportLimit
	export := &SecurityAuditExport{
		ExportKey:       uuid.NewString(),
		RequestedByType: strings.TrimSpace(input.RequestedByType),
		RequestedByID:   input.RequestedByID,
		Status:          SecurityAuditExportStatusPending,
		Filters:         securityAuditFilterSnapshot(input.Filter),
		RequestedAt:     time.Now().UTC(),
	}
	if export.RequestedByType == "" {
		export.RequestedByType = "admin"
	}
	created, err := s.repo.CreateAuditExport(ctx, export)
	if err != nil {
		return nil, err
	}

	logs, err := s.repo.ListForExport(ctx, input.Filter, defaultSecurityAuditExportLimit)
	if err != nil {
		return s.failAuditExport(ctx, created.ID, err)
	}
	filePath, sha, rowCount, err := writeSecurityAuditExportCSV(created.ExportKey, logs)
	if err != nil {
		return s.failAuditExport(ctx, created.ID, err)
	}
	completedAt := time.Now().UTC()
	expiresAt := completedAt.Add(7 * 24 * time.Hour)
	return s.repo.UpdateAuditExportResult(ctx, created.ID, SecurityAuditExportStatusCompleted, filePath, sha, rowCount, "", &completedAt, &expiresAt)
}

func (s *SecurityAuditService) ListAuditExports(ctx context.Context, filter SecurityAuditExportListFilter) ([]SecurityAuditExport, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("security audit service is not configured")
	}
	filter.Page, filter.PageSize = normalizeSecurityAuditPagination(filter.Page, filter.PageSize)
	return s.repo.ListAuditExports(ctx, filter)
}

func (s *SecurityAuditService) GetAuditExportFile(ctx context.Context, id int64) (*SecurityAuditExportFile, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("security audit service is not configured")
	}
	if id <= 0 {
		return nil, infraerrors.BadRequest("INVALID_SECURITY_EXPORT_ID", "invalid export id")
	}
	export, err := s.repo.GetAuditExport(ctx, id)
	if err != nil {
		return nil, err
	}
	if export.Status != SecurityAuditExportStatusCompleted || strings.TrimSpace(export.FilePath) == "" {
		return nil, infraerrors.BadRequest("SECURITY_EXPORT_NOT_READY", "security audit export is not ready")
	}
	if export.ExpiresAt != nil && time.Now().After(*export.ExpiresAt) {
		return nil, infraerrors.BadRequest("SECURITY_EXPORT_EXPIRED", "security audit export has expired")
	}
	if _, err := os.Stat(export.FilePath); err != nil {
		if os.IsNotExist(err) {
			return nil, infraerrors.NotFound("SECURITY_EXPORT_FILE_NOT_FOUND", "security audit export file not found")
		}
		return nil, fmt.Errorf("stat security audit export file: %w", err)
	}
	return &SecurityAuditExportFile{
		Path:        export.FilePath,
		Filename:    "security_audit_" + export.ExportKey + ".csv",
		ContentType: "text/csv; charset=utf-8",
	}, nil
}

func (s *SecurityAuditService) failAuditExport(ctx context.Context, id int64, cause error) (*SecurityAuditExport, error) {
	completedAt := time.Now().UTC()
	message := ""
	if cause != nil {
		message = cause.Error()
	}
	item, updateErr := s.repo.UpdateAuditExportResult(ctx, id, SecurityAuditExportStatusFailed, "", "", 0, message, &completedAt, nil)
	if updateErr != nil {
		return nil, updateErr
	}
	return item, cause
}

func (s *SecurityAuditService) ListIncidents(ctx context.Context, filter SecurityIncidentListFilter) ([]SecurityIncident, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("security audit service is not configured")
	}
	filter.Page, filter.PageSize = normalizeSecurityAuditPagination(filter.Page, filter.PageSize)
	return s.repo.ListIncidents(ctx, filter)
}

func (s *SecurityAuditService) ListPolicyRules(ctx context.Context, filter SecurityPolicyRuleListFilter) ([]SecurityPolicyRule, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("security audit service is not configured")
	}
	filter.Page, filter.PageSize = normalizeSecurityAuditPagination(filter.Page, filter.PageSize)
	return s.repo.ListPolicyRules(ctx, filter)
}

func (s *SecurityAuditService) CreatePolicyRule(ctx context.Context, input SecurityPolicyRuleInput) (*SecurityPolicyRule, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("security audit service is not configured")
	}
	input = normalizeSecurityPolicyRuleInput(input, true)
	if input.Name == "" || input.Code == "" {
		return nil, infraerrors.BadRequest("INVALID_SECURITY_POLICY", "policy name and code are required")
	}
	return s.repo.CreatePolicyRule(ctx, input)
}

func (s *SecurityAuditService) UpdatePolicyRule(ctx context.Context, id int64, input SecurityPolicyRuleInput) (*SecurityPolicyRule, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("security audit service is not configured")
	}
	if id <= 0 {
		return nil, infraerrors.BadRequest("INVALID_SECURITY_POLICY_ID", "invalid policy id")
	}
	input = normalizeSecurityPolicyRuleInput(input, false)
	if input.Name == "" || input.Code == "" {
		return nil, infraerrors.BadRequest("INVALID_SECURITY_POLICY", "policy name and code are required")
	}
	return s.repo.UpdatePolicyRule(ctx, id, input)
}

func (s *SecurityAuditService) DeletePolicyRule(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("security audit service is not configured")
	}
	if id <= 0 {
		return infraerrors.BadRequest("INVALID_SECURITY_POLICY_ID", "invalid policy id")
	}
	return s.repo.DeletePolicyRule(ctx, id)
}

func (s *SecurityAuditService) ListSubjectLocks(ctx context.Context, filter SecuritySubjectLockListFilter) ([]SecuritySubjectLock, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("security audit service is not configured")
	}
	filter.Page, filter.PageSize = normalizeSecurityAuditPagination(filter.Page, filter.PageSize)
	return s.repo.ListSubjectLocks(ctx, filter)
}

func (s *SecurityAuditService) LockSubject(ctx context.Context, input SecuritySubjectLockInput) (*SecuritySubjectLock, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("security audit service is not configured")
	}
	input.SubjectType = strings.TrimSpace(input.SubjectType)
	if input.SubjectType == "" || input.SubjectID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_SECURITY_LOCK_SUBJECT", "invalid lock subject")
	}
	input.Reason = redactSecurityAuditText(input.Reason)
	input.LockedByType = strings.TrimSpace(input.LockedByType)
	input.Metadata = RedactSecurityAuditMap(input.Metadata)
	return s.repo.CreateSubjectLock(ctx, input)
}

func (s *SecurityAuditService) UnlockSubject(ctx context.Context, id int64, actorType string, actorID *int64, reason string) (*SecuritySubjectLock, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("security audit service is not configured")
	}
	if id <= 0 {
		return nil, infraerrors.BadRequest("INVALID_SECURITY_LOCK_ID", "invalid lock id")
	}
	return s.repo.UnlockSubjectLock(ctx, id, strings.TrimSpace(actorType), actorID, redactSecurityAuditText(reason))
}

func (s *SecurityAuditService) EnforceGatewaySecurity(ctx context.Context, input SecurityPolicyEvaluationInput) (*SecurityPolicyDecision, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	now := input.OccurredAt
	if now.IsZero() {
		now = time.Now()
	}
	if input.APIKeyID > 0 {
		lock, err := s.repo.GetActiveSubjectLock(ctx, "api_key", input.APIKeyID, now)
		if err != nil {
			return nil, err
		}
		if lock != nil {
			return &SecurityPolicyDecision{
				Matched:     true,
				Blocked:     true,
				Action:      SecurityPolicyActionBlock,
				Reason:      lock.Reason,
				RiskLevel:   SecurityAuditRiskHigh,
				SubjectLock: lock,
			}, nil
		}
	}
	if input.UserID > 0 {
		lock, err := s.repo.GetActiveSubjectLock(ctx, "user", input.UserID, now)
		if err != nil {
			return nil, err
		}
		if lock != nil {
			return &SecurityPolicyDecision{
				Matched:     true,
				Blocked:     true,
				Action:      SecurityPolicyActionBlock,
				Reason:      lock.Reason,
				RiskLevel:   SecurityAuditRiskHigh,
				SubjectLock: lock,
			}, nil
		}
	}

	rules, err := s.repo.ListEnabledPolicyRules(ctx)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		rule := rules[i]
		if !securityPolicyMatches(rule.Conditions, input) {
			continue
		}
		decision := &SecurityPolicyDecision{
			Matched:   true,
			Blocked:   securityPolicyActionBlocks(rule.Action),
			Action:    normalizeSecurityPolicyAction(rule.Action),
			Reason:    rule.Description,
			RuleID:    &rule.ID,
			RuleCode:  rule.Code,
			RiskLevel: normalizeSecurityRisk(rule.Severity),
		}
		if decision.Reason == "" {
			decision.Reason = "blocked by security policy " + rule.Code
		}
		if decision.Blocked {
			if lockInput, ok := policyActionLockInput(rule, input, decision.Reason, now); ok {
				if lock, lockErr := s.repo.CreateSubjectLock(ctx, lockInput); lockErr == nil {
					decision.SubjectLock = lock
				}
			}
			return decision, ErrSecuritySubjectLocked
		}
		return decision, nil
	}
	return nil, nil
}

func normalizeSecurityAuditInput(input SecurityAuditCreateInput) SecurityAuditLog {
	action := strings.TrimSpace(input.Action)
	result := strings.TrimSpace(strings.ToLower(input.Result))
	if result == "" {
		result = SecurityAuditResultSuccess
	}
	riskLevel := strings.TrimSpace(strings.ToLower(input.RiskLevel))
	if riskLevel == "" {
		riskLevel = SecurityAuditRiskLow
	}
	return SecurityAuditLog{
		EventType:    action,
		ActorType:    strings.TrimSpace(input.ActorType),
		ActorID:      input.ActorID,
		ActorLabel:   redactSecurityAuditText(input.ActorLabel),
		SubjectType:  strings.TrimSpace(input.SubjectType),
		SubjectID:    input.SubjectID,
		SubjectLabel: redactSecurityAuditText(input.SubjectLabel),
		ResourceType: strings.TrimSpace(input.ResourceType),
		ResourceID:   redactSecurityAuditText(input.ResourceID),
		Action:       action,
		Result:       result,
		RiskLevel:    riskLevel,
		RequestID:    strings.TrimSpace(input.RequestID),
		IP:           strings.TrimSpace(input.IP),
		UserAgent:    strings.TrimSpace(input.UserAgent),
		Endpoint:     strings.TrimSpace(input.Endpoint),
		Reason:       redactSecurityAuditText(input.Reason),
		Metadata:     RedactSecurityAuditMap(input.Metadata),
		DiffSummary:  RedactSecurityAuditMap(input.DiffSummary),
	}
}

func normalizeSecurityAuditPagination(page, pageSize int) (int, int) {
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

// ComputeSecurityAuditEntryHash is exported so tests and integrity checks use
// the exact same canonical representation as inserts.
func ComputeSecurityAuditEntryHash(log *SecurityAuditLog) string {
	if log == nil {
		return ""
	}
	payload := map[string]any{
		"actor_id":      log.ActorID,
		"actor_label":   log.ActorLabel,
		"actor_type":    log.ActorType,
		"action":        log.Action,
		"diff_summary":  log.DiffSummary,
		"endpoint":      log.Endpoint,
		"ip":            log.IP,
		"metadata":      log.Metadata,
		"prev_hash":     log.PrevHash,
		"reason":        log.Reason,
		"request_id":    log.RequestID,
		"resource_id":   log.ResourceID,
		"resource_type": log.ResourceType,
		"result":        log.Result,
		"risk_level":    log.RiskLevel,
		"subject_id":    log.SubjectID,
		"subject_label": log.SubjectLabel,
		"subject_type":  log.SubjectType,
		"user_agent":    log.UserAgent,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func RedactSecurityAuditMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	return logredact.RedactMap(input,
		"api_key",
		"apikey",
		"bearer",
		"cookie",
		"private_key",
		"secret_key",
		"session",
		"signature",
	)
}

func securityAuditFilterSnapshot(filter SecurityAuditListFilter) map[string]any {
	out := map[string]any{}
	if filter.ActorType != "" {
		out["actor_type"] = strings.TrimSpace(filter.ActorType)
	}
	if filter.ActorID != nil {
		out["actor_id"] = *filter.ActorID
	}
	if filter.SubjectType != "" {
		out["subject_type"] = strings.TrimSpace(filter.SubjectType)
	}
	if filter.SubjectID != nil {
		out["subject_id"] = *filter.SubjectID
	}
	if filter.Action != "" {
		out["action"] = strings.TrimSpace(filter.Action)
	}
	if filter.Result != "" {
		out["result"] = strings.TrimSpace(filter.Result)
	}
	if filter.RiskLevel != "" {
		out["risk_level"] = strings.TrimSpace(filter.RiskLevel)
	}
	if filter.RequestID != "" {
		out["request_id"] = strings.TrimSpace(filter.RequestID)
	}
	if filter.Query != "" {
		out["q"] = strings.TrimSpace(filter.Query)
	}
	if filter.StartTime != nil {
		out["start_time"] = filter.StartTime.UTC().Format(time.RFC3339)
	}
	if filter.EndTime != nil {
		out["end_time"] = filter.EndTime.UTC().Format(time.RFC3339)
	}
	out["limit"] = defaultSecurityAuditExportLimit
	return out
}

func writeSecurityAuditExportCSV(exportKey string, logs []SecurityAuditLog) (string, string, int64, error) {
	dir := securityAuditExportDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", 0, fmt.Errorf("create security audit export dir: %w", err)
	}
	filename := "security_audit_" + exportKey + ".csv"
	path := filepath.Join(dir, filename)
	tmpPath := path + ".tmp"

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.Write([]string{
		"id", "created_at", "action", "result", "risk_level",
		"actor_type", "actor_id", "actor_label",
		"subject_type", "subject_id", "subject_label",
		"resource_type", "resource_id", "request_id", "ip", "user_agent", "endpoint", "reason",
		"metadata", "diff_summary", "prev_hash", "entry_hash",
	}); err != nil {
		return "", "", 0, fmt.Errorf("write security audit export header: %w", err)
	}
	for i := range logs {
		log := logs[i]
		if err := writer.Write([]string{
			strconv.FormatInt(log.ID, 10),
			log.CreatedAt.UTC().Format(time.RFC3339),
			log.Action,
			log.Result,
			log.RiskLevel,
			log.ActorType,
			formatOptionalInt64(log.ActorID),
			log.ActorLabel,
			log.SubjectType,
			formatOptionalInt64(log.SubjectID),
			log.SubjectLabel,
			log.ResourceType,
			log.ResourceID,
			log.RequestID,
			log.IP,
			log.UserAgent,
			log.Endpoint,
			log.Reason,
			mustSecurityAuditJSONString(log.Metadata),
			mustSecurityAuditJSONString(log.DiffSummary),
			log.PrevHash,
			log.EntryHash,
		}); err != nil {
			return "", "", 0, fmt.Errorf("write security audit export row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", "", 0, fmt.Errorf("flush security audit export: %w", err)
	}
	raw := buf.Bytes()
	sum := sha256.Sum256(raw)
	if err := os.WriteFile(tmpPath, raw, 0o600); err != nil {
		return "", "", 0, fmt.Errorf("write security audit export file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return "", "", 0, fmt.Errorf("finalize security audit export file: %w", err)
	}
	return path, hex.EncodeToString(sum[:]), int64(len(logs)), nil
}

func securityAuditExportDir() string {
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if dataDir == "" {
		dataDir = filepath.Join(".", "data")
	}
	return filepath.Join(dataDir, "security", "audit_exports")
}

func formatOptionalInt64(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func mustSecurityAuditJSONString(value map[string]any) string {
	raw, err := json.Marshal(nonNilSecurityAuditMap(value))
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func nonNilSecurityAuditMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	return input
}

func redactSecurityAuditText(input string) string {
	return logredact.RedactText(strings.TrimSpace(input),
		"api_key",
		"apikey",
		"bearer",
		"cookie",
		"private_key",
		"secret_key",
		"session",
		"signature",
	)
}

func normalizeSecurityPolicyRuleInput(input SecurityPolicyRuleInput, create bool) SecurityPolicyRuleInput {
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.TrimSpace(input.Code)
	input.Description = redactSecurityAuditText(input.Description)
	input.Severity = normalizeSecurityRisk(input.Severity)
	input.Action = normalizeSecurityPolicyAction(input.Action)
	if input.Enabled == nil && create {
		enabled := true
		input.Enabled = &enabled
	}
	input.Conditions = RedactSecurityAuditMap(input.Conditions)
	input.Metadata = RedactSecurityAuditMap(input.Metadata)
	return input
}

func normalizeSecurityRisk(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case SecurityAuditRiskCritical:
		return SecurityAuditRiskCritical
	case SecurityAuditRiskHigh:
		return SecurityAuditRiskHigh
	case SecurityAuditRiskLow:
		return SecurityAuditRiskLow
	default:
		return SecurityAuditRiskMedium
	}
}

func normalizeSecurityPolicyAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case SecurityPolicyActionBlock:
		return SecurityPolicyActionBlock
	case SecurityPolicyActionChallenge:
		return SecurityPolicyActionChallenge
	case SecurityPolicyActionDisableAPIKey:
		return SecurityPolicyActionDisableAPIKey
	case SecurityPolicyActionDisableUser:
		return SecurityPolicyActionDisableUser
	case SecurityPolicyActionTemporaryLock:
		return SecurityPolicyActionTemporaryLock
	case SecurityPolicyActionNotifyAdmin:
		return SecurityPolicyActionNotifyAdmin
	case SecurityPolicyActionNotifyUser:
		return SecurityPolicyActionNotifyUser
	default:
		return SecurityPolicyActionObserve
	}
}

func securityPolicyActionBlocks(action string) bool {
	switch normalizeSecurityPolicyAction(action) {
	case SecurityPolicyActionBlock, SecurityPolicyActionDisableAPIKey, SecurityPolicyActionDisableUser, SecurityPolicyActionTemporaryLock:
		return true
	default:
		return false
	}
}

func securityPolicyMatches(conditions map[string]any, input SecurityPolicyEvaluationInput) bool {
	if len(conditions) == 0 {
		return false
	}
	if !securityConditionIDMatches(conditions["user_id"], input.UserID) {
		return false
	}
	if !securityConditionIDMatches(conditions["api_key_id"], input.APIKeyID) {
		return false
	}
	if input.GroupID != nil && !securityConditionIDMatches(conditions["group_id"], *input.GroupID) {
		return false
	}
	if input.GroupID == nil && hasSecurityCondition(conditions, "group_id") {
		return false
	}
	if !securityConditionStringMatches(conditions["endpoint"], input.Endpoint) {
		return false
	}
	if !securityConditionStringMatches(conditions["model"], input.Model) {
		return false
	}
	if !securityConditionStringMatches(conditions["user_agent"], input.UserAgent) {
		return false
	}
	if !securityConditionIPMatches(conditions["ip"], input.IP) {
		return false
	}
	if !securityConditionIPMatches(conditions["cidr"], input.IP) {
		return false
	}
	return true
}

func hasSecurityCondition(conditions map[string]any, key string) bool {
	_, ok := conditions[key]
	return ok
}

func securityConditionIDMatches(condition any, actual int64) bool {
	if condition == nil {
		return true
	}
	values := securityConditionValues(condition)
	if len(values) == 0 {
		return true
	}
	actualString := strconv.FormatInt(actual, 10)
	for _, value := range values {
		if strings.TrimSpace(value) == actualString {
			return true
		}
	}
	return false
}

func securityConditionStringMatches(condition any, actual string) bool {
	if condition == nil {
		return true
	}
	values := securityConditionValues(condition)
	if len(values) == 0 {
		return true
	}
	actual = strings.TrimSpace(actual)
	if actual == "" {
		return false
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.HasSuffix(value, "*") {
			if strings.HasPrefix(actual, strings.TrimSuffix(value, "*")) {
				return true
			}
			continue
		}
		if strings.EqualFold(actual, value) {
			return true
		}
	}
	return false
}

func securityConditionIPMatches(condition any, actual string) bool {
	if condition == nil {
		return true
	}
	values := securityConditionValues(condition)
	if len(values) == 0 {
		return true
	}
	parsed := net.ParseIP(strings.TrimSpace(actual))
	if parsed == nil {
		return false
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, cidr, err := net.ParseCIDR(value); err == nil {
			if cidr.Contains(parsed) {
				return true
			}
			continue
		}
		if ip := net.ParseIP(value); ip != nil && ip.Equal(parsed) {
			return true
		}
	}
	return false
}

func securityConditionValues(condition any) []string {
	switch v := condition.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		return []string{v}
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, securityConditionValues(item)...)
		}
		return out
	case float64:
		return []string{strconv.FormatInt(int64(v), 10)}
	case int:
		return []string{strconv.Itoa(v)}
	case int64:
		return []string{strconv.FormatInt(v, 10)}
	default:
		return nil
	}
}

func policyActionLockInput(rule SecurityPolicyRule, input SecurityPolicyEvaluationInput, reason string, now time.Time) (SecuritySubjectLockInput, bool) {
	action := normalizeSecurityPolicyAction(rule.Action)
	lock := SecuritySubjectLockInput{
		Reason:       reason,
		LockedByType: "policy",
		LockedByID:   &rule.ID,
		Metadata: map[string]any{
			"policy_id":   rule.ID,
			"policy_code": rule.Code,
			"action":      action,
			"endpoint":    input.Endpoint,
			"model":       input.Model,
		},
	}
	if action == SecurityPolicyActionTemporaryLock {
		expiresAt := now.Add(securityPolicyTemporaryLockDuration(rule.Metadata))
		lock.ExpiresAt = &expiresAt
	}
	switch action {
	case SecurityPolicyActionDisableAPIKey:
		if input.APIKeyID <= 0 {
			return lock, false
		}
		lock.SubjectType = "api_key"
		lock.SubjectID = input.APIKeyID
		return lock, true
	case SecurityPolicyActionDisableUser:
		if input.UserID <= 0 {
			return lock, false
		}
		lock.SubjectType = "user"
		lock.SubjectID = input.UserID
		return lock, true
	case SecurityPolicyActionTemporaryLock:
		if input.APIKeyID > 0 {
			lock.SubjectType = "api_key"
			lock.SubjectID = input.APIKeyID
			return lock, true
		}
		if input.UserID > 0 {
			lock.SubjectType = "user"
			lock.SubjectID = input.UserID
			return lock, true
		}
		return lock, false
	case SecurityPolicyActionBlock:
		return lock, false
	default:
		return lock, false
	}
}

func securityPolicyTemporaryLockDuration(metadata map[string]any) time.Duration {
	if len(metadata) == 0 {
		return time.Hour
	}
	for _, key := range []string{"lock_minutes", "duration_minutes"} {
		values := securityConditionValues(metadata[key])
		if len(values) == 0 {
			continue
		}
		minutes, err := strconv.Atoi(values[0])
		if err == nil && minutes > 0 && minutes <= 43200 {
			return time.Duration(minutes) * time.Minute
		}
	}
	return time.Hour
}
