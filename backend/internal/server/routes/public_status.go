package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicStatusRoutes registers anonymous public status endpoints.
func RegisterPublicStatusRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	status := v1.Group("/status")
	{
		status.GET("/public", h.PublicStatus.Get)
	}
}
