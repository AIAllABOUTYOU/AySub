package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type adminMediaCacheCleanupRequest struct {
	Type       string `json:"type"`
	BeforeUnix int64  `json:"before_unix"`
	OlderThan  string `json:"older_than"`
	Limit      int    `json:"limit"`
}

func (h *OpenAIGatewayHandler) AdminListMediaCache(c *gin.Context) {
	if h == nil || h.gatewayService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Service temporarily unavailable")
		return
	}
	page, pageSize := response.ParsePagination(c)
	before, err := parseAdminMediaCacheBefore(c.Query("before_unix"), c.Query("older_than"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.gatewayService.ListLocalMediaCache(service.LocalMediaCacheListFilter{
		Type:     c.Query("type"),
		Search:   c.Query("search"),
		Before:   before,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *OpenAIGatewayHandler) AdminDeleteMediaCacheItem(c *gin.Context) {
	if h == nil || h.gatewayService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Service temporarily unavailable")
		return
	}
	if err := h.gatewayService.DeleteLocalMediaCacheItem(c.Param("type"), c.Param("id")); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": 1})
}

func (h *OpenAIGatewayHandler) AdminCleanupMediaCache(c *gin.Context) {
	if h == nil || h.gatewayService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Service temporarily unavailable")
		return
	}
	var req adminMediaCacheCleanupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	before, err := parseAdminMediaCacheBefore(strconv.FormatInt(req.BeforeUnix, 10), req.OlderThan)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.gatewayService.CleanupLocalMediaCache(service.LocalMediaCacheCleanupInput{
		Type:   req.Type,
		Before: before,
		Limit:  req.Limit,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *OpenAIGatewayHandler) AdminCleanupMediaOrphans(c *gin.Context) {
	if h == nil || h.gatewayService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Service temporarily unavailable")
		return
	}
	result, err := h.gatewayService.CleanupLocalMediaOrphans()
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, result)
}

func parseAdminMediaCacheBefore(beforeUnix, olderThan string) (time.Time, error) {
	beforeUnix = strings.TrimSpace(beforeUnix)
	if beforeUnix != "" && beforeUnix != "0" {
		value, err := strconv.ParseInt(beforeUnix, 10, 64)
		if err != nil || value < 0 {
			return time.Time{}, errInvalidMediaCacheBefore()
		}
		return time.Unix(value, 0), nil
	}
	olderThan = strings.TrimSpace(olderThan)
	if olderThan == "" {
		return time.Time{}, nil
	}
	duration, err := time.ParseDuration(olderThan)
	if err != nil || duration <= 0 {
		return time.Time{}, errInvalidMediaCacheBefore()
	}
	return time.Now().Add(-duration), nil
}

func errInvalidMediaCacheBefore() error {
	return errString("before_unix must be a unix timestamp or older_than must be a positive duration")
}

type errString string

func (e errString) Error() string {
	return string(e)
}
