package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type UserSecurityHandler struct {
	auditService  *service.SecurityAuditService
	apiKeyService *service.APIKeyService
	authService   *service.AuthService
	userService   *service.UserService
	totpService   *service.TotpService
}

func NewUserSecurityHandler(auditService *service.SecurityAuditService, apiKeyServices ...*service.APIKeyService) *UserSecurityHandler {
	var apiKeyService *service.APIKeyService
	if len(apiKeyServices) > 0 {
		apiKeyService = apiKeyServices[0]
	}
	return &UserSecurityHandler{auditService: auditService, apiKeyService: apiKeyService}
}

func (h *UserSecurityHandler) SetSensitiveOperationServices(authService *service.AuthService, userService *service.UserService, totpService *service.TotpService) {
	h.authService = authService
	h.userService = userService
	h.totpService = totpService
}

// ListEvents returns the authenticated user's own security audit events.
// GET /api/v1/user/security/events
func (h *UserSecurityHandler) ListEvents(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	filter, ok := parseUserSecurityAuditFilter(c)
	if !ok {
		return
	}
	logs, page, err := h.auditService.ListUserEvents(c.Request.Context(), subject.UserID, filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, logs, page.Total, page.Page, page.PageSize)
}

// RevokeAPIKey lets users revoke their own API key from the account security page.
// POST /api/v1/user/security/api-keys/:id/revoke
func (h *UserSecurityHandler) RevokeAPIKey(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.apiKeyService == nil {
		response.InternalError(c, "API key service is not configured")
		return
	}

	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || keyID <= 0 {
		response.BadRequest(c, "Invalid key ID")
		return
	}
	if err := h.requireSensitiveOperationToken(c, subject.UserID, "api_key.revoke"); err != nil {
		writeSecurityAuditAsync(c, h.auditService, service.SecurityAuditCreateInput{
			SubjectType:  "user",
			SubjectID:    auditUserSubject(subject.UserID),
			ResourceType: "api_key",
			ResourceID:   strconv.FormatInt(keyID, 10),
			Action:       "api_key.revoke",
			Result:       service.SecurityAuditResultDenied,
			RiskLevel:    service.SecurityAuditRiskHigh,
			Reason:       err.Error(),
		})
		response.ErrorFrom(c, err)
		return
	}

	key, getErr := h.apiKeyService.GetByID(c.Request.Context(), keyID)
	err = h.apiKeyService.Delete(c.Request.Context(), keyID, subject.UserID)
	if err != nil {
		writeSecurityAuditAsync(c, h.auditService, service.SecurityAuditCreateInput{
			SubjectType:  "user",
			SubjectID:    auditUserSubject(subject.UserID),
			ResourceType: "api_key",
			ResourceID:   strconv.FormatInt(keyID, 10),
			Action:       "api_key.revoke",
			Result:       service.SecurityAuditResultFailure,
			RiskLevel:    service.SecurityAuditRiskHigh,
			Reason:       err.Error(),
		})
		response.ErrorFrom(c, err)
		return
	}

	if getErr == nil && key != nil {
		securityAuditForAPIKey(c, h.auditService, key, "api_key.revoke", service.SecurityAuditResultSuccess, service.SecurityAuditRiskHigh, "API key revoked by user", map[string]any{
			"revoked_key_id": key.ID,
			"name":           key.Name,
			"group_id":       key.GroupID,
		})
	} else {
		writeSecurityAuditAsync(c, h.auditService, service.SecurityAuditCreateInput{
			SubjectType:  "user",
			SubjectID:    auditUserSubject(subject.UserID),
			ResourceType: "api_key",
			ResourceID:   strconv.FormatInt(keyID, 10),
			Action:       "api_key.revoke",
			Result:       service.SecurityAuditResultSuccess,
			RiskLevel:    service.SecurityAuditRiskHigh,
			Reason:       "API key revoked by user",
		})
	}
	response.Success(c, gin.H{"success": true})
}

type SensitiveOperationVerifyRequest struct {
	Action       string `json:"action" binding:"required"`
	Password     string `json:"password"`
	TotpCode     string `json:"totp_code"`
	RecoveryCode string `json:"recovery_code"`
}

func (h *UserSecurityHandler) VerifySensitiveOperation(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.authService == nil || h.userService == nil {
		response.InternalError(c, "auth service is not configured")
		return
	}
	var req SensitiveOperationVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	action := strings.TrimSpace(req.Action)
	if action == "" {
		response.BadRequest(c, "Action is required")
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	method := ""
	if strings.TrimSpace(req.Password) != "" {
		method = "password"
		if !user.CheckPassword(req.Password) {
			err = service.ErrPasswordIncorrect
		}
	} else if strings.TrimSpace(req.TotpCode) != "" {
		method = "totp"
		if h.totpService == nil {
			err = service.ErrTotpNotSetup
		} else {
			err = h.totpService.VerifyCode(c.Request.Context(), subject.UserID, req.TotpCode)
		}
	} else if strings.TrimSpace(req.RecoveryCode) != "" {
		method = "recovery_code"
		if h.totpService == nil {
			err = service.ErrTotpRecoveryInvalid
		} else {
			err = h.totpService.VerifyRecoveryCode(c.Request.Context(), subject.UserID, req.RecoveryCode)
		}
	} else {
		err = service.ErrSensitiveVerificationRequired
	}
	if err != nil {
		securityAuditForUser(c, h.auditService, subject.UserID, user.Email, "security.sensitive_verify", service.SecurityAuditResultDenied, service.SecurityAuditRiskHigh, err.Error(), map[string]any{
			"action": action,
			"method": method,
		})
		response.ErrorFrom(c, err)
		return
	}

	token, expiresIn, err := h.authService.GenerateSensitiveOperationToken(c.Request.Context(), subject.UserID, action)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	securityAuditForUser(c, h.auditService, subject.UserID, user.Email, "security.sensitive_verify", service.SecurityAuditResultSuccess, service.SecurityAuditRiskMedium, "sensitive operation verified", map[string]any{
		"action": action,
		"method": method,
	})
	response.Success(c, gin.H{
		"token":      token,
		"expires_in": expiresIn,
		"action":     action,
	})
}

func (h *UserSecurityHandler) requireSensitiveOperationToken(c *gin.Context, userID int64, action string) error {
	if h.authService == nil {
		return service.ErrSensitiveVerificationRequired
	}
	token := strings.TrimSpace(c.GetHeader("X-Sensitive-Operation-Token"))
	if token == "" {
		token = strings.TrimSpace(c.GetHeader("X-Step-Up-Token"))
	}
	if token == "" {
		token = strings.TrimSpace(c.Query("sensitive_token"))
	}
	return h.authService.ValidateSensitiveOperationToken(c.Request.Context(), token, userID, action)
}

func parseUserSecurityAuditFilter(c *gin.Context) (service.SecurityAuditListFilter, bool) {
	page, pageSize := response.ParsePagination(c)
	filter := service.SecurityAuditListFilter{
		Page:      page,
		PageSize:  pageSize,
		Action:    strings.TrimSpace(c.Query("action")),
		Result:    strings.TrimSpace(c.Query("result")),
		RiskLevel: strings.TrimSpace(c.Query("risk_level")),
		RequestID: strings.TrimSpace(c.Query("request_id")),
		Query:     strings.TrimSpace(c.Query("q")),
	}

	start, ok := parseUserSecurityAuditTime(c, "start_time")
	if !ok {
		return filter, false
	}
	end, ok := parseUserSecurityAuditTime(c, "end_time")
	if !ok {
		return filter, false
	}
	filter.StartTime = start
	filter.EndTime = end

	if v := strings.TrimSpace(c.Query("actor_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid actor_id")
			return filter, false
		}
		filter.ActorID = &id
	}
	if v := strings.TrimSpace(c.Query("actor_type")); v != "" {
		filter.ActorType = v
	}
	return filter, true
}

func parseUserSecurityAuditTime(c *gin.Context, name string) (*time.Time, bool) {
	raw := strings.TrimSpace(c.Query(name))
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
