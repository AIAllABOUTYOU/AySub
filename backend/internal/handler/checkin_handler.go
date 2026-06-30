package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type CheckinHandler struct {
	checkinService       *service.CheckinService
	securityAuditService *service.SecurityAuditService
}

func NewCheckinHandler(checkinService *service.CheckinService, securityAuditServices ...*service.SecurityAuditService) *CheckinHandler {
	var securityAuditService *service.SecurityAuditService
	if len(securityAuditServices) > 0 {
		securityAuditService = securityAuditServices[0]
	}
	return &CheckinHandler{checkinService: checkinService, securityAuditService: securityAuditService}
}

// Status returns today's check-in state for the current user.
// GET /api/v1/user/checkin/status
func (h *CheckinHandler) Status(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	status, err := h.checkinService.Status(c.Request.Context(), subject.UserID, c.Query("timezone"), c.Query("month"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

// Claim grants today's check-in reward for the current user.
// POST /api/v1/user/checkin
func (h *CheckinHandler) Claim(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	status, err := h.checkinService.Claim(c.Request.Context(), subject.UserID, c.Query("timezone"))
	if err != nil {
		securityAuditForUser(c, h.securityAuditService, subject.UserID, "", "checkin.claim", service.SecurityAuditResultDenied, service.SecurityAuditRiskMedium, err.Error(), map[string]any{
			"timezone": c.Query("timezone"),
		})
		response.ErrorFrom(c, err)
		return
	}
	metadata := map[string]any{
		"timezone":     c.Query("timezone"),
		"checkin_date": status.CheckinDate,
		"streak_days":  status.StreakDays,
	}
	if status.AwardedAmount != nil {
		metadata["awarded_amount"] = *status.AwardedAmount
	}
	if status.NewBalance != nil {
		metadata["new_balance"] = *status.NewBalance
	}
	securityAuditForUser(c, h.securityAuditService, subject.UserID, "", "checkin.claim", service.SecurityAuditResultSuccess, service.SecurityAuditRiskLow, "每日签到完成", metadata)
	response.Success(c, status)
}
