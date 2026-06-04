package routes

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type gatewayPermissionProtocol string

const (
	gatewayPermissionProtocolAuto   gatewayPermissionProtocol = "auto"
	gatewayPermissionProtocolGoogle gatewayPermissionProtocol = "google"
)

func requireAPIKeyGatewayPermission(protocol gatewayPermissionProtocol) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey, ok := middleware.GetAPIKeyFromContext(c)
		if !ok || apiKey == nil {
			c.Next()
			return
		}

		endpointID := permissionEndpointID(handler.GetInboundEndpoint(c))
		if endpointID == "" {
			c.Next()
			return
		}
		if !apiKey.AllowsEndpoint(endpointID) {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalPolicyDenied)
			writeAPIKeyPermissionDenied(c, protocol, endpointID, "", fmt.Sprintf("API key is not permitted to use endpoint %s", endpointID))
			return
		}

		model := permissionRequestedModel(c)
		if !apiKey.AllowsModel(model) {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalPolicyDenied)
			writeAPIKeyPermissionDenied(c, protocol, endpointID, model, fmt.Sprintf("API key is not permitted to use model %s", model))
			return
		}

		c.Next()
	}
}

func permissionEndpointID(inbound string) string {
	switch inbound {
	case handler.EndpointMessages:
		return service.EndpointPermissionMessages
	case handler.EndpointChatCompletions:
		return service.EndpointPermissionChatCompletions
	case handler.EndpointResponses:
		return service.EndpointPermissionResponses
	case handler.EndpointEmbeddings:
		return service.EndpointPermissionEmbeddings
	case handler.EndpointImagesGenerations, handler.EndpointImagesEdits:
		return service.EndpointPermissionImages
	case handler.EndpointVideos:
		return service.EndpointPermissionVideos
	case handler.EndpointAudioSpeech:
		return service.EndpointPermissionAudioSpeech
	case handler.EndpointAudioTranscriptions:
		return service.EndpointPermissionAudioTranscriptions
	case handler.EndpointAudioTranslations:
		return service.EndpointPermissionAudioTranslations
	case handler.EndpointLiveKitTokens, handler.EndpointLiveKitRTC:
		return service.EndpointPermissionLiveKit
	case handler.EndpointGeminiModels:
		return service.EndpointPermissionGeminiNative
	default:
		return ""
	}
}

func permissionRequestedModel(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if model := permissionGeminiModelFromPath(c); model != "" {
		return model
	}
	if c.Request.Body == nil || c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
		return ""
	}

	body, err := httputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))

	if model := permissionMultipartModel(c.GetHeader("Content-Type"), body); model != "" {
		return model
	}
	return strings.TrimSpace(gjson.GetBytes(body, "model").String())
}

func permissionGeminiModelFromPath(c *gin.Context) string {
	if handler.GetInboundEndpoint(c) != handler.EndpointGeminiModels {
		return ""
	}
	if model := strings.TrimSpace(c.Param("model")); model != "" {
		return strings.TrimPrefix(model, "models/")
	}
	action := strings.TrimSpace(c.Param("modelAction"))
	action = strings.TrimPrefix(action, "/")
	action = strings.TrimPrefix(action, "models/")
	if action == "" {
		return ""
	}
	if idx := strings.IndexByte(action, ':'); idx >= 0 {
		action = action[:idx]
	}
	return strings.TrimSpace(action)
}

func permissionMultipartModel(contentType string, body []byte) string {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
		return ""
	}
	boundary := params["boundary"]
	if boundary == "" {
		return ""
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if err != nil {
			return ""
		}
		if part.FormName() != "model" {
			_ = part.Close()
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(part, 4096))
		_ = part.Close()
		if readErr != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
}

func writeAPIKeyPermissionDenied(c *gin.Context, protocol gatewayPermissionProtocol, endpointID, model, message string) {
	if protocol == gatewayPermissionProtocolGoogle {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    http.StatusForbidden,
				"message": message,
				"status":  googleapi.HTTPStatusToGoogleStatus(http.StatusForbidden),
			},
		})
		c.Abort()
		return
	}

	inbound := handler.GetInboundEndpoint(c)
	if inbound == handler.EndpointMessages {
		c.JSON(http.StatusForbidden, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "permission_error",
				"message": message,
			},
		})
		c.Abort()
		return
	}

	c.JSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"type":    "permission_error",
			"code":    "api_key_permission_denied",
			"message": message,
			"param":   permissionDeniedParam(endpointID, model),
		},
	})
	c.Abort()
}

func permissionDeniedParam(endpointID, model string) string {
	if strings.TrimSpace(model) != "" {
		return "model"
	}
	if strings.TrimSpace(endpointID) != "" {
		return "endpoint"
	}
	return ""
}
