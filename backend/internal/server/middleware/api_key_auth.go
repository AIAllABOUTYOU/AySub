package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// NewAPIKeyAuthMiddleware 创建 API Key 认证中间件
func NewAPIKeyAuthMiddleware(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config, auditServices ...*service.SecurityAuditService) APIKeyAuthMiddleware {
	return APIKeyAuthMiddleware(apiKeyAuthWithSubscription(apiKeyService, subscriptionService, cfg, firstSecurityAuditService(auditServices)))
}

// apiKeyAuthWithSubscription API Key认证中间件（支持订阅验证）
//
// 中间件职责分为两层：
//   - 鉴权（Authentication）：验证 Key 有效性、用户状态、IP 限制 —— 始终执行
//   - 计费执行（Billing Enforcement）：过期/配额/订阅/余额检查 —— skipBilling 时整块跳过
//
// /v1/usage 端点只需鉴权，不需要计费执行（允许过期/配额耗尽的 Key 查询自身用量）。
func apiKeyAuthWithSubscription(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config, auditService *service.SecurityAuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ── 1. 提取 API Key ──────────────────────────────────────────

		queryKey := strings.TrimSpace(c.Query("key"))
		queryApiKey := strings.TrimSpace(c.Query("api_key"))
		if queryKey != "" || queryApiKey != "" {
			auditAPIKeyAuthDenied(c, auditService, nil, "api_key_in_query_deprecated", "API key in query parameter is deprecated", 400)
			AbortWithError(c, 400, "api_key_in_query_deprecated", "API key in query parameter is deprecated. Please use Authorization header instead.")
			return
		}

		// 尝试从Authorization header中提取API key (Bearer scheme)
		authHeader := c.GetHeader("Authorization")
		var apiKeyString string

		if authHeader != "" {
			// 验证Bearer scheme
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				apiKeyString = strings.TrimSpace(parts[1])
			}
		}

		// 如果Authorization header中没有，尝试从x-api-key header中提取
		if apiKeyString == "" {
			apiKeyString = c.GetHeader("x-api-key")
		}

		// 如果x-api-key header中没有，尝试从x-goog-api-key header中提取（Gemini CLI兼容）
		if apiKeyString == "" {
			apiKeyString = c.GetHeader("x-goog-api-key")
		}

		// 如果所有header都没有API key
		if apiKeyString == "" {
			auditAPIKeyAuthDenied(c, auditService, nil, "API_KEY_REQUIRED", "API key is required", 401)
			AbortWithError(c, 401, "API_KEY_REQUIRED", "API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header")
			return
		}

		// ── 2. 验证 Key 存在 ─────────────────────────────────────────

		apiKey, err := apiKeyService.GetByKey(c.Request.Context(), apiKeyString)
		if err != nil {
			if errors.Is(err, service.ErrAPIKeyNotFound) {
				auditAPIKeyAuthDenied(c, auditService, nil, "INVALID_API_KEY", "Invalid API key", 401)
				AbortWithError(c, 401, "INVALID_API_KEY", "Invalid API key")
				return
			}
			auditAPIKeyAuthFailure(c, auditService, nil, "INTERNAL_ERROR", "Failed to validate API key", 500)
			AbortWithError(c, 500, "INTERNAL_ERROR", "Failed to validate API key")
			return
		}

		// ── 3. 基础鉴权（始终执行） ─────────────────────────────────

		// disabled / 未知状态 → 无条件拦截（expired 和 quota_exhausted 留给计费阶段）
		if !apiKey.IsActive() &&
			apiKey.Status != service.StatusAPIKeyExpired &&
			apiKey.Status != service.StatusAPIKeyQuotaExhausted {
			auditAPIKeyAuthDenied(c, auditService, apiKey, "API_KEY_DISABLED", "API key is disabled", 401)
			AbortWithError(c, 401, "API_KEY_DISABLED", "API key is disabled")
			return
		}

		// 检查 IP 限制（白名单/黑名单）
		// 注意：错误信息故意模糊，避免暴露具体的 IP 限制机制
		if len(apiKey.IPWhitelist) > 0 || len(apiKey.IPBlacklist) > 0 {
			clientIP := ip.GetTrustedClientIP(c)
			if cfg.TrustForwardedIPForAPIKeyACL() {
				clientIP = ip.GetClientIP(c)
			}
			allowed, _ := ip.CheckIPRestrictionWithCompiledRules(clientIP, apiKey.CompiledIPWhitelist, apiKey.CompiledIPBlacklist)
			if !allowed {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonIPRestriction)
				auditAPIKeyAuthDenied(c, auditService, apiKey, "ACCESS_DENIED", "API key IP ACL denied request", 403, map[string]any{
					"client_ip":     clientIP,
					"has_whitelist": len(apiKey.IPWhitelist) > 0,
					"has_blacklist": len(apiKey.IPBlacklist) > 0,
				})
				AbortWithError(c, 403, "ACCESS_DENIED", "Access denied")
				return
			}
		}

		// 检查关联的用户
		if apiKey.User == nil {
			auditAPIKeyAuthDenied(c, auditService, apiKey, "USER_NOT_FOUND", "User associated with API key not found", 401)
			AbortWithError(c, 401, "USER_NOT_FOUND", "User associated with API key not found")
			return
		}

		// 检查用户状态
		if !apiKey.User.IsActive() {
			auditAPIKeyAuthDenied(c, auditService, apiKey, "USER_INACTIVE", "User account is not active", 401)
			AbortWithError(c, 401, "USER_INACTIVE", "User account is not active")
			return
		}
		if denyBySecurityPolicy(c, auditService, apiKey) {
			return
		}
		if abortIfAPIKeyGroupUnavailable(c, apiKey, auditService) {
			return
		}

		// ── 4. SimpleMode → early return ─────────────────────────────

		if cfg.RunMode == config.RunModeSimple {
			c.Set(string(ContextKeyAPIKey), apiKey)
			c.Set(string(ContextKeyUser), AuthSubject{
				UserID:      apiKey.User.ID,
				Concurrency: apiKey.User.Concurrency,
			})
			c.Set(string(ContextKeyUserRole), apiKey.User.Role)
			setGroupContext(c, apiKey.Group)
			_ = apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)
			c.Next()
			return
		}

		// ── 5. 加载订阅（订阅模式时始终加载） ───────────────────────

		// skipBilling: /v1/usage 只需鉴权，跳过所有计费执行
		skipBilling := c.Request.URL.Path == "/v1/usage"

		var subscription *service.UserSubscription
		isSubscriptionType := apiKey.Group != nil && apiKey.Group.IsSubscriptionType()

		if isSubscriptionType && subscriptionService != nil {
			sub, subErr := subscriptionService.GetActiveSubscription(
				c.Request.Context(),
				apiKey.User.ID,
				apiKey.Group.ID,
			)
			if subErr != nil {
				if !skipBilling {
					auditAPIKeyAuthDenied(c, auditService, apiKey, "SUBSCRIPTION_NOT_FOUND", "No active subscription found for this group", 403)
					AbortWithError(c, 403, "SUBSCRIPTION_NOT_FOUND", "No active subscription found for this group")
					return
				}
				// skipBilling: 订阅不存在也放行，handler 会返回可用的数据
			} else {
				subscription = sub
			}
		}

		// ── 6. 计费执行（skipBilling 时整块跳过） ────────────────────

		if !skipBilling {
			// Key 状态检查
			switch apiKey.Status {
			case service.StatusAPIKeyQuotaExhausted:
				auditAPIKeyAuthDenied(c, auditService, apiKey, "API_KEY_QUOTA_EXHAUSTED", "API key quota exhausted", 429)
				AbortWithError(c, 429, "API_KEY_QUOTA_EXHAUSTED", "API key 额度已用完")
				return
			case service.StatusAPIKeyExpired:
				auditAPIKeyAuthDenied(c, auditService, apiKey, "API_KEY_EXPIRED", "API key expired", 403)
				AbortWithError(c, 403, "API_KEY_EXPIRED", "API key 已过期")
				return
			}

			// 运行时过期/配额检查（即使状态是 active，也要检查时间和用量）
			if apiKey.IsExpired() {
				auditAPIKeyAuthDenied(c, auditService, apiKey, "API_KEY_EXPIRED", "API key expired", 403)
				AbortWithError(c, 403, "API_KEY_EXPIRED", "API key 已过期")
				return
			}
			if apiKey.IsQuotaExhausted() {
				auditAPIKeyAuthDenied(c, auditService, apiKey, "API_KEY_QUOTA_EXHAUSTED", "API key quota exhausted", 429)
				AbortWithError(c, 429, "API_KEY_QUOTA_EXHAUSTED", "API key 额度已用完")
				return
			}

			// 订阅模式：验证订阅限额
			if subscription != nil {
				needsMaintenance, validateErr := subscriptionService.ValidateAndCheckLimits(subscription, apiKey.Group)
				if validateErr != nil {
					code := "SUBSCRIPTION_INVALID"
					status := 403
					if errors.Is(validateErr, service.ErrDailyLimitExceeded) ||
						errors.Is(validateErr, service.ErrWeeklyLimitExceeded) ||
						errors.Is(validateErr, service.ErrMonthlyLimitExceeded) {
						code = "USAGE_LIMIT_EXCEEDED"
						status = 429
					}
					auditAPIKeyAuthDenied(c, auditService, apiKey, code, validateErr.Error(), status)
					AbortWithError(c, status, code, validateErr.Error())
					return
				}

				// 窗口维护异步化（不阻塞请求）
				if needsMaintenance {
					maintenanceCopy := *subscription
					subscriptionService.DoWindowMaintenance(&maintenanceCopy)
				}
			} else {
				// 非订阅模式 或 订阅模式但 subscriptionService 未注入：回退到余额检查
				if apiKey.User.Balance <= 0 {
					auditAPIKeyAuthDenied(c, auditService, apiKey, "INSUFFICIENT_BALANCE", "Insufficient account balance", 403)
					AbortWithError(c, 403, "INSUFFICIENT_BALANCE", "Insufficient account balance")
					return
				}
			}
		}

		// ── 7. 设置上下文 → Next ─────────────────────────────────────

		if subscription != nil {
			c.Set(string(ContextKeySubscription), subscription)
		}
		c.Set(string(ContextKeyAPIKey), apiKey)
		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      apiKey.User.ID,
			Concurrency: apiKey.User.Concurrency,
		})
		c.Set(string(ContextKeyUserRole), apiKey.User.Role)
		setGroupContext(c, apiKey.Group)
		_ = apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)

		c.Next()
	}
}

// GetAPIKeyFromContext 从上下文中获取API key
func GetAPIKeyFromContext(c *gin.Context) (*service.APIKey, bool) {
	value, exists := c.Get(string(ContextKeyAPIKey))
	if !exists {
		return nil, false
	}
	apiKey, ok := value.(*service.APIKey)
	return apiKey, ok
}

// GetSubscriptionFromContext 从上下文中获取订阅信息
func GetSubscriptionFromContext(c *gin.Context) (*service.UserSubscription, bool) {
	value, exists := c.Get(string(ContextKeySubscription))
	if !exists {
		return nil, false
	}
	subscription, ok := value.(*service.UserSubscription)
	return subscription, ok
}

func setGroupContext(c *gin.Context, group *service.Group) {
	if !service.IsGroupContextValid(group) {
		return
	}
	if existing, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group); ok && existing != nil && existing.ID == group.ID && service.IsGroupContextValid(existing) {
		return
	}
	ctx := context.WithValue(c.Request.Context(), ctxkey.Group, group)
	c.Request = c.Request.WithContext(ctx)
}

func abortIfAPIKeyGroupUnavailable(c *gin.Context, apiKey *service.APIKey, auditServices ...*service.SecurityAuditService) bool {
	code, message, ok := validateAPIKeyGroupAvailable(apiKey)
	if ok {
		return false
	}
	service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonAPIKeyGroupUnavailable)
	auditAPIKeyAuthDenied(c, firstSecurityAuditService(auditServices), apiKey, code, message, 403)
	AbortWithError(c, 403, code, message)
	return true
}

func validateAPIKeyGroupAvailable(apiKey *service.APIKey) (string, string, bool) {
	if apiKey == nil || apiKey.GroupID == nil {
		return "", "", true
	}
	group := apiKey.Group
	if group == nil || strings.EqualFold(group.Status, "deleted") {
		return "GROUP_DELETED", "API Key 所属分组已删除", false
	}
	if !group.IsActive() {
		return "GROUP_DISABLED", "API Key 所属分组已停用", false
	}
	return "", "", true
}

func firstSecurityAuditService(auditServices []*service.SecurityAuditService) *service.SecurityAuditService {
	if len(auditServices) == 0 {
		return nil
	}
	return auditServices[0]
}

func auditAPIKeyAuthDenied(c *gin.Context, auditService *service.SecurityAuditService, apiKey *service.APIKey, code, reason string, statusCode int, extra ...map[string]any) {
	auditAPIKeyAuth(c, auditService, apiKey, service.SecurityAuditResultDenied, service.SecurityAuditRiskHigh, code, reason, statusCode, extra...)
}

func auditAPIKeyAuthFailure(c *gin.Context, auditService *service.SecurityAuditService, apiKey *service.APIKey, code, reason string, statusCode int, extra ...map[string]any) {
	auditAPIKeyAuth(c, auditService, apiKey, service.SecurityAuditResultFailure, service.SecurityAuditRiskHigh, code, reason, statusCode, extra...)
}

func auditAPIKeyAuth(c *gin.Context, auditService *service.SecurityAuditService, apiKey *service.APIKey, result, risk, code, reason string, statusCode int, extra ...map[string]any) {
	metadata := map[string]any{
		"error_code":  code,
		"status_code": statusCode,
	}
	for _, item := range extra {
		for k, v := range item {
			metadata[k] = v
		}
	}
	WriteAPIKeySecurityAuditAsync(c, auditService, apiKey, "api_key.auth", result, risk, reason, metadata)
}

func denyBySecurityPolicy(c *gin.Context, auditService *service.SecurityAuditService, apiKey *service.APIKey) bool {
	if auditService == nil || apiKey == nil {
		return false
	}
	input := securityPolicyEvaluationInput(c, apiKey)
	decision, err := auditService.EnforceGatewaySecurity(c.Request.Context(), input)
	if err != nil && !errors.Is(err, service.ErrSecuritySubjectLocked) {
		auditAPIKeyAuthFailure(c, auditService, apiKey, "SECURITY_POLICY_ERROR", err.Error(), http.StatusInternalServerError)
		AbortWithError(c, http.StatusInternalServerError, "SECURITY_POLICY_ERROR", "Failed to evaluate security policy")
		return true
	}
	if decision == nil || !decision.Blocked {
		return false
	}
	reason := decision.Reason
	if strings.TrimSpace(reason) == "" {
		reason = "Blocked by security policy"
	}
	auditAPIKeyAuthDenied(c, auditService, apiKey, "SECURITY_POLICY_DENIED", reason, http.StatusForbidden, map[string]any{
		"policy_action": decision.Action,
		"policy_rule_id": func() any {
			if decision.RuleID == nil {
				return nil
			}
			return *decision.RuleID
		}(),
		"policy_rule_code": decision.RuleCode,
		"subject_lock_id": func() any {
			if decision.SubjectLock == nil {
				return nil
			}
			return decision.SubjectLock.ID
		}(),
	})
	service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalPolicyDenied)
	AbortWithError(c, http.StatusForbidden, "SECURITY_POLICY_DENIED", reason)
	return true
}

func securityPolicyEvaluationInput(c *gin.Context, apiKey *service.APIKey) service.SecurityPolicyEvaluationInput {
	input := service.SecurityPolicyEvaluationInput{
		OccurredAt: time.Now(),
	}
	if apiKey != nil {
		input.UserID = apiKey.UserID
		input.APIKeyID = apiKey.ID
		input.GroupID = apiKey.GroupID
	}
	if c != nil && c.Request != nil {
		input.IP = ip.GetClientIP(c)
		input.UserAgent = c.Request.UserAgent()
		if c.Request.URL != nil {
			input.Endpoint = c.Request.Method + " " + c.Request.URL.Path
		}
	}
	return input
}
