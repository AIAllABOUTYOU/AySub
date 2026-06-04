package admin

import (
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SecurityHandler struct {
	auditService *service.SecurityAuditService
}

func NewSecurityHandler(auditService *service.SecurityAuditService) *SecurityHandler {
	return &SecurityHandler{auditService: auditService}
}

// ListAuditLogs handles admin read-only audit log queries.
// GET /api/v1/admin/security/audit-logs
func (h *SecurityHandler) ListAuditLogs(c *gin.Context) {
	filter, ok := parseSecurityAuditListFilter(c)
	if !ok {
		return
	}
	logs, page, err := h.auditService.ListAuditLogs(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, logs, page.Total, page.Page, page.PageSize)
}

// IntegrityCheck verifies the hash chain from the first audit entry to the latest.
// GET /api/v1/admin/security/integrity/check
func (h *SecurityHandler) IntegrityCheck(c *gin.Context) {
	result, err := h.auditService.IntegrityCheck(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ListExports lists security audit export jobs.
// GET /api/v1/admin/security/exports
func (h *SecurityHandler) ListExports(c *gin.Context) {
	filter, ok := parseSecurityAuditExportListFilter(c)
	if !ok {
		return
	}
	items, page, err := h.auditService.ListAuditExports(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, page.Total, page.Page, page.PageSize)
}

// CreateExport creates a CSV export from the current audit-log filter.
// POST /api/v1/admin/security/exports
func (h *SecurityHandler) CreateExport(c *gin.Context) {
	filter, ok := parseSecurityAuditExportCreateFilter(c)
	if !ok {
		return
	}
	item, err := h.auditService.CreateAuditExport(c.Request.Context(), service.SecurityAuditExportCreateInput{
		Filter:          filter,
		RequestedByType: "admin",
		RequestedByID:   currentAdminActorID(c),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

// DownloadExport downloads a completed audit CSV export.
// GET /api/v1/admin/security/exports/:id/download
func (h *SecurityHandler) DownloadExport(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	file, err := h.auditService.GetAuditExportFile(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Content-Type", file.ContentType)
	c.Header("Content-Disposition", `attachment; filename="`+file.Filename+`"`)
	c.File(file.Path)
}

// ListIncidents handles admin incident queries.
// GET /api/v1/admin/security/incidents
func (h *SecurityHandler) ListIncidents(c *gin.Context) {
	filter, ok := parseSecurityIncidentListFilter(c)
	if !ok {
		return
	}
	items, page, err := h.auditService.ListIncidents(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, page.Total, page.Page, page.PageSize)
}

// ListPolicies handles admin security policy queries.
// GET /api/v1/admin/security/policies
func (h *SecurityHandler) ListPolicies(c *gin.Context) {
	filter := parseSecurityPolicyListFilter(c)
	items, page, err := h.auditService.ListPolicyRules(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, page.Total, page.Page, page.PageSize)
}

// CreatePolicy creates a security policy rule.
// POST /api/v1/admin/security/policies
func (h *SecurityHandler) CreatePolicy(c *gin.Context) {
	var req service.SecurityPolicyRuleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.auditService.CreatePolicyRule(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

// UpdatePolicy updates a security policy rule.
// PUT /api/v1/admin/security/policies/:id
func (h *SecurityHandler) UpdatePolicy(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	var req service.SecurityPolicyRuleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.auditService.UpdatePolicyRule(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

// DeletePolicy deletes a security policy rule.
// DELETE /api/v1/admin/security/policies/:id
func (h *SecurityHandler) DeletePolicy(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.auditService.DeletePolicyRule(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"success": true})
}

// ListLocks handles admin lock queries.
// GET /api/v1/admin/security/locks
func (h *SecurityHandler) ListLocks(c *gin.Context) {
	filter, ok := parseSecurityLockListFilter(c)
	if !ok {
		return
	}
	items, page, err := h.auditService.ListSubjectLocks(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, page.Total, page.Page, page.PageSize)
}

// CreateLock manually locks a user or API key.
// POST /api/v1/admin/security/locks
func (h *SecurityHandler) CreateLock(c *gin.Context) {
	var req service.SecuritySubjectLockInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actorID := currentAdminActorID(c)
	req.LockedByType = "admin"
	req.LockedByID = actorID
	item, err := h.auditService.LockSubject(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

type unlockSecurityLockRequest struct {
	Reason string `json:"reason"`
}

// UnlockLock unlocks a security subject lock.
// POST /api/v1/admin/security/locks/:id/unlock
func (h *SecurityHandler) UnlockLock(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}
	var req unlockSecurityLockRequest
	_ = c.ShouldBindJSON(&req)
	item, err := h.auditService.UnlockSubject(c.Request.Context(), id, "admin", currentAdminActorID(c), req.Reason)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func parseSecurityAuditListFilter(c *gin.Context) (service.SecurityAuditListFilter, bool) {
	page, pageSize := response.ParsePagination(c)
	filter := service.SecurityAuditListFilter{
		Page:      page,
		PageSize:  pageSize,
		ActorType: strings.TrimSpace(c.Query("actor_type")),
		Action:    strings.TrimSpace(c.Query("action")),
		Result:    strings.TrimSpace(c.Query("result")),
		RiskLevel: strings.TrimSpace(c.Query("risk_level")),
		RequestID: strings.TrimSpace(c.Query("request_id")),
		Query:     strings.TrimSpace(c.Query("q")),
	}

	if v := strings.TrimSpace(c.Query("actor_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid actor_id")
			return filter, false
		}
		filter.ActorID = &id
	}
	if v := strings.TrimSpace(c.Query("subject_type")); v != "" {
		filter.SubjectType = v
	}
	if v := strings.TrimSpace(c.Query("subject_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid subject_id")
			return filter, false
		}
		filter.SubjectID = &id
	}

	start, ok := parseSecurityAuditTime(c, "start_time")
	if !ok {
		return filter, false
	}
	end, ok := parseSecurityAuditTime(c, "end_time")
	if !ok {
		return filter, false
	}
	filter.StartTime = start
	filter.EndTime = end
	return filter, true
}

type securityAuditExportCreateRequest struct {
	ActorType   string `json:"actor_type"`
	ActorID     *int64 `json:"actor_id"`
	SubjectType string `json:"subject_type"`
	SubjectID   *int64 `json:"subject_id"`
	Action      string `json:"action"`
	Result      string `json:"result"`
	RiskLevel   string `json:"risk_level"`
	RequestID   string `json:"request_id"`
	Query       string `json:"q"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
}

func parseSecurityAuditExportCreateFilter(c *gin.Context) (service.SecurityAuditListFilter, bool) {
	var req securityAuditExportCreateRequest
	if c.Request != nil && c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return service.SecurityAuditListFilter{}, false
		}
	}
	filter := service.SecurityAuditListFilter{
		ActorType:   strings.TrimSpace(req.ActorType),
		ActorID:     req.ActorID,
		SubjectType: strings.TrimSpace(req.SubjectType),
		SubjectID:   req.SubjectID,
		Action:      strings.TrimSpace(req.Action),
		Result:      strings.TrimSpace(req.Result),
		RiskLevel:   strings.TrimSpace(req.RiskLevel),
		RequestID:   strings.TrimSpace(req.RequestID),
		Query:       strings.TrimSpace(req.Query),
	}
	if strings.TrimSpace(req.StartTime) != "" {
		t, ok := parseSecurityAuditTimeValue(c, "start_time", req.StartTime)
		if !ok {
			return filter, false
		}
		filter.StartTime = t
	}
	if strings.TrimSpace(req.EndTime) != "" {
		t, ok := parseSecurityAuditTimeValue(c, "end_time", req.EndTime)
		if !ok {
			return filter, false
		}
		filter.EndTime = t
	}
	return filter, true
}

func parseSecurityAuditExportListFilter(c *gin.Context) (service.SecurityAuditExportListFilter, bool) {
	page, pageSize := response.ParsePagination(c)
	filter := service.SecurityAuditExportListFilter{
		Page:            page,
		PageSize:        pageSize,
		Status:          strings.TrimSpace(c.Query("status")),
		RequestedByType: strings.TrimSpace(c.Query("requested_by_type")),
		Query:           strings.TrimSpace(c.Query("q")),
	}
	if v := strings.TrimSpace(c.Query("requested_by_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid requested_by_id")
			return filter, false
		}
		filter.RequestedByID = &id
	}
	return filter, true
}

func parseSecurityIncidentListFilter(c *gin.Context) (service.SecurityIncidentListFilter, bool) {
	page, pageSize := response.ParsePagination(c)
	filter := service.SecurityIncidentListFilter{
		Page:        page,
		PageSize:    pageSize,
		Status:      strings.TrimSpace(c.Query("status")),
		Severity:    strings.TrimSpace(c.Query("severity")),
		SubjectType: strings.TrimSpace(c.Query("subject_type")),
		Query:       strings.TrimSpace(c.Query("q")),
	}
	if v := strings.TrimSpace(c.Query("subject_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid subject_id")
			return filter, false
		}
		filter.SubjectID = &id
	}
	return filter, true
}

func parseSecurityPolicyListFilter(c *gin.Context) service.SecurityPolicyRuleListFilter {
	page, pageSize := response.ParsePagination(c)
	filter := service.SecurityPolicyRuleListFilter{
		Page:     page,
		PageSize: pageSize,
		Severity: strings.TrimSpace(c.Query("severity")),
		Action:   strings.TrimSpace(c.Query("action")),
		Query:    strings.TrimSpace(c.Query("q")),
	}
	if v := strings.TrimSpace(c.Query("enabled")); v != "" {
		enabled := strings.EqualFold(v, "true") || v == "1"
		filter.Enabled = &enabled
	}
	return filter
}

func parseSecurityLockListFilter(c *gin.Context) (service.SecuritySubjectLockListFilter, bool) {
	page, pageSize := response.ParsePagination(c)
	filter := service.SecuritySubjectLockListFilter{
		Page:        page,
		PageSize:    pageSize,
		Status:      strings.TrimSpace(c.Query("status")),
		SubjectType: strings.TrimSpace(c.Query("subject_type")),
		Query:       strings.TrimSpace(c.Query("q")),
	}
	if v := strings.TrimSpace(c.Query("subject_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid subject_id")
			return filter, false
		}
		filter.SubjectID = &id
	}
	return filter, true
}

func parsePositiveIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid "+name)
		return 0, false
	}
	return id, true
}

func currentAdminActorID(c *gin.Context) *int64 {
	if c == nil {
		return nil
	}
	for _, key := range []string{"user_id", "admin_id"} {
		if raw, ok := c.Get(key); ok {
			switch v := raw.(type) {
			case int64:
				if v > 0 {
					return &v
				}
			case int:
				if v > 0 {
					id := int64(v)
					return &id
				}
			}
		}
	}
	return nil
}

func parseSecurityAuditTime(c *gin.Context, name string) (*time.Time, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, true
	}
	return parseSecurityAuditTimeValue(c, name, raw)
}

func parseSecurityAuditTimeValue(c *gin.Context, name string, raw string) (*time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return &t, true
		}
	}
	response.BadRequest(c, "Invalid "+name)
	return nil, false
}
