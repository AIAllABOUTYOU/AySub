package middleware

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// WriteAPIKeySecurityAuditAsync records API-key boundary decisions without
// blocking the client path. It intentionally never stores the raw key value.
func WriteAPIKeySecurityAuditAsync(c *gin.Context, svc *service.SecurityAuditService, apiKey *service.APIKey, action, result, risk, reason string, metadata map[string]any) {
	if svc == nil {
		return
	}

	input := service.SecurityAuditCreateInput{
		ActorType:    "anonymous",
		SubjectType:  "api_key",
		ResourceType: "api_key",
		Action:       action,
		Result:       result,
		RiskLevel:    risk,
		Reason:       reason,
		Metadata:     cloneSecurityAuditMetadata(metadata),
	}

	if apiKey != nil {
		input.ResourceID = strconv.FormatInt(apiKey.ID, 10)
		input.Metadata["api_key_id"] = apiKey.ID
		input.Metadata["api_key_name"] = apiKey.Name
		input.Metadata["api_key_status"] = apiKey.Status
		input.Metadata["user_id"] = apiKey.UserID
		input.Metadata["permission_mode"] = apiKey.PermissionMode
		if apiKey.GroupID != nil {
			input.Metadata["group_id"] = *apiKey.GroupID
		}
		input.DiffSummary = map[string]any{
			"status":            apiKey.Status,
			"group_id":          apiKey.GroupID,
			"permission_mode":   apiKey.PermissionMode,
			"allowed_models":    apiKey.AllowedModels,
			"allowed_endpoints": apiKey.AllowedEndpoints,
		}
		if apiKey.UserID > 0 {
			input.ActorType = "user"
			input.ActorID = auditInt64Ptr(apiKey.UserID)
			input.SubjectType = "user"
			input.SubjectID = auditInt64Ptr(apiKey.UserID)
			if apiKey.User != nil {
				input.ActorLabel = strings.TrimSpace(apiKey.User.Email)
				input.SubjectLabel = strings.TrimSpace(apiKey.User.Email)
			}
		}
	}

	if c != nil && c.Request != nil {
		input.IP = ip.GetClientIP(c)
		input.UserAgent = c.Request.UserAgent()
		if c.Request.URL != nil {
			input.Endpoint = c.Request.Method + " " + c.Request.URL.Path
		}
		input.RequestID = auditRequestID(c.Request.Context())
	}

	ctx := context.Background()
	if input.RequestID != "" {
		ctx = context.WithValue(ctx, ctxkey.RequestID, input.RequestID)
	}
	go func() {
		auditCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if _, err := svc.CreateAuditLog(auditCtx, input); err != nil {
			slog.Warn("security audit write failed", "action", input.Action, "error", err)
		}
	}()
}

func auditRequestID(ctx context.Context) string {
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

func auditInt64Ptr(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}

func cloneSecurityAuditMetadata(metadata map[string]any) map[string]any {
	out := make(map[string]any, len(metadata)+8)
	for k, v := range metadata {
		out[k] = v
	}
	return out
}
