package handler

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *OpenAIGatewayHandler) handleOpenAIAccountFailover(
	c *gin.Context,
	reqLog *zap.Logger,
	account *service.Account,
	failoverErr *service.UpstreamFailoverError,
	failedAccountIDs map[int64]struct{},
	lastFailoverErr **service.UpstreamFailoverError,
	switchCount *int,
	maxAccountSwitches int,
) bool {
	if h == nil || account == nil || failoverErr == nil {
		return false
	}
	h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
	h.gatewayService.TempUnscheduleFailoverError(c.Request.Context(), account.ID, failoverErr)
	h.gatewayService.RecordOpenAIAccountSwitch()
	failedAccountIDs[account.ID] = struct{}{}
	if lastFailoverErr != nil {
		*lastFailoverErr = failoverErr
	}
	if *switchCount >= maxAccountSwitches {
		return false
	}
	*switchCount++
	if h.gatewayService.ShouldStopOpenAIOAuth429Failover(account, failoverErr.StatusCode, *switchCount) {
		return false
	}
	if reqLog != nil {
		reqLog.Warn("openai.upstream_failover_switching",
			zap.Int64("account_id", account.ID),
			zap.Int("upstream_status", failoverErr.StatusCode),
			zap.Int("switch_count", *switchCount),
			zap.Int("max_switches", maxAccountSwitches),
		)
	}
	return true
}

func sleepOpenAISameAccountRetry(ctx context.Context, delay time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(delay):
		return true
	}
}
