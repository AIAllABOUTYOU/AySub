package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PublicStatusHandler exposes the anonymous, redacted public status endpoint.
type PublicStatusHandler struct {
	publicStatusService *service.PublicStatusService
}

func NewPublicStatusHandler(publicStatusService *service.PublicStatusService) *PublicStatusHandler {
	return &PublicStatusHandler{publicStatusService: publicStatusService}
}

// Get returns the public status view.
// GET /api/v1/status/public
func (h *PublicStatusHandler) Get(c *gin.Context) {
	status, err := h.publicStatusService.GetPublicStatus(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}
