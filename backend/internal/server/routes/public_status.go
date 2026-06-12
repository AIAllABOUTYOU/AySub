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

	models := v1.Group("/models")
	{
		models.GET("/marketplace", h.AvailableChannel.PublicMarketplace)
	}

	// Public announcements (no auth required)
	announcements := v1.Group("/announcements")
	{
		announcements.GET("/public", h.Announcement.ListPublic)
	}
}
