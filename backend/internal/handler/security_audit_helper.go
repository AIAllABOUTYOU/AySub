package handler

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type securityAuditContext struct {
	ActorType    string
	ActorID      *int64
	ActorLabel   string
	SubjectType  string
	SubjectID    *int64
	SubjectLabel string
	RequestID    string
	IP           string
	UserAgent    string
	Endpoint     string
}

func auditUserSubject(userID int64) *int64 {
	if userID <= 0 {
		return nil
	}
	return &userID
}

func securityAuditContextFromRequest(c *gin.Context, fallbackSubjectType string, fallbackUserID int64, fallbackLabel string) securityAuditContext {
	ctx := securityAuditContext{
		ActorType:   "system",
		SubjectType: strings.TrimSpace(fallbackSubjectType),
		SubjectID:   auditUserSubject(fallbackUserID),
		SubjectLabel: strings.TrimSpace(fallbackLabel),
	}
	if ctx.SubjectType == "" && fallbackUserID > 0 {
		ctx.SubjectType = "user"
	}
	if c == nil {
		return ctx
	}
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		ctx.ActorType = "user"
		ctx.ActorID = auditUserSubject(subject.UserID)
		if ctx.SubjectID == nil {
			ctx.SubjectType = "user"
			ctx.SubjectID = auditUserSubject(subject.UserID)
		}
	}
	if c.Request != nil {
		ctx.IP = ip.GetClientIP(c)
		ctx.UserAgent = c.Request.UserAgent()
		if c.Request.URL != nil {
			ctx.Endpoint = c.Request.Method + " " + c.Request.URL.Path
		}
		ctx.RequestID = securityAuditRequestID(c.Request.Context())
	}
	return ctx
}

func securityAuditRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(ctxkey.RequestID).(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if v, ok := ctx.Value(ctxkey.ClientRequestID).(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func writeSecurityAuditAsync(c *gin.Context, svc *service.SecurityAuditService, input service.SecurityAuditCreateInput) {
	if svc == nil {
		return
	}
	base := securityAuditContextFromRequest(c, input.SubjectType, derefInt64(input.SubjectID), input.SubjectLabel)
	if input.ActorType == "" {
		input.ActorType = base.ActorType
	}
	if input.ActorID == nil {
		input.ActorID = base.ActorID
	}
	if input.ActorLabel == "" {
		input.ActorLabel = base.ActorLabel
	}
	if input.SubjectType == "" {
		input.SubjectType = base.SubjectType
	}
	if input.SubjectID == nil {
		input.SubjectID = base.SubjectID
	}
	if input.SubjectLabel == "" {
		input.SubjectLabel = base.SubjectLabel
	}
	if input.RequestID == "" {
		input.RequestID = base.RequestID
	}
	if input.IP == "" {
		input.IP = base.IP
	}
	if input.UserAgent == "" {
		input.UserAgent = base.UserAgent
	}
	if input.Endpoint == "" {
		input.Endpoint = base.Endpoint
	}

	ctx := context.Background()
	if c != nil && c.Request != nil {
		if requestID := securityAuditRequestID(c.Request.Context()); requestID != "" {
			ctx = context.WithValue(ctx, ctxkey.RequestID, requestID)
		}
	}
	go func() {
		auditCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if _, err := svc.CreateAuditLog(auditCtx, input); err != nil {
			slog.Warn("security audit write failed", "action", input.Action, "error", err)
		}
	}()
}

func securityAuditForUser(c *gin.Context, svc *service.SecurityAuditService, userID int64, userLabel, action, result, risk, reason string, metadata map[string]any) {
	writeSecurityAuditAsync(c, svc, service.SecurityAuditCreateInput{
		SubjectType:  "user",
		SubjectID:    auditUserSubject(userID),
		SubjectLabel: strings.TrimSpace(userLabel),
		ResourceType: "user",
		ResourceID:   strconv.FormatInt(userID, 10),
		Action:       action,
		Result:       result,
		RiskLevel:    risk,
		Reason:       reason,
		Metadata:     metadata,
	})
}

func securityAuditForAPIKey(c *gin.Context, svc *service.SecurityAuditService, key *service.APIKey, action, result, risk, reason string, metadata map[string]any) {
	if key == nil {
		writeSecurityAuditAsync(c, svc, service.SecurityAuditCreateInput{
			SubjectType:  "api_key",
			ResourceType: "api_key",
			Action:       action,
			Result:       result,
			RiskLevel:    risk,
			Reason:       reason,
			Metadata:     metadata,
		})
		return
	}
	resourceID := strconv.FormatInt(key.ID, 10)
	subjectID := auditUserSubject(key.UserID)
	writeSecurityAuditAsync(c, svc, service.SecurityAuditCreateInput{
		SubjectType:  "user",
		SubjectID:    subjectID,
		ResourceType: "api_key",
		ResourceID:   resourceID,
		Action:       action,
		Result:       result,
		RiskLevel:    risk,
		Reason:       reason,
		Metadata:     metadata,
		DiffSummary: map[string]any{
			"name":              key.Name,
			"status":            key.Status,
			"group_id":          key.GroupID,
			"permission_mode":   key.PermissionMode,
			"allowed_models":    key.AllowedModels,
			"allowed_endpoints": key.AllowedEndpoints,
		},
	})
}

func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
