package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type noAccountErrorClassification struct {
	Status        int
	ErrType       string
	Message       string
	ModelNotFound bool
}

func classifyNoAccountError(
	ctx context.Context,
	diagnoser service.ModelAvailabilityDiagnoser,
	apiKey *service.APIKey,
	routingModel string,
	displayModel string,
	platform string,
) noAccountErrorClassification {
	fallback := noAccountErrorClassification{
		Status:  http.StatusServiceUnavailable,
		ErrType: "api_error",
		Message: "Service temporarily unavailable",
	}
	routingModel = strings.TrimSpace(routingModel)
	displayModel = strings.TrimSpace(displayModel)
	if displayModel == "" {
		displayModel = routingModel
	}
	if diagnoser == nil || apiKey == nil || apiKey.GroupID == nil || routingModel == "" {
		return fallback
	}

	diagnosis := diagnoser.DiagnoseModelAvailabilityForPlatform(ctx, apiKey.GroupID, routingModel, platform)
	if diagnosis.HasAccountsInPool && !diagnosis.HasModelSupport {
		return noAccountErrorClassification{
			Status:        http.StatusNotFound,
			ErrType:       "model_not_found",
			Message:       fmt.Sprintf("Model %q is not supported by any configured account in this group", displayModel),
			ModelNotFound: true,
		}
	}
	return fallback
}

func classifyNoAccountErrorFromGin(
	c *gin.Context,
	diagnoser service.ModelAvailabilityDiagnoser,
	apiKey *service.APIKey,
	routingModel string,
	displayModel string,
	platform string,
) noAccountErrorClassification {
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	return classifyNoAccountError(ctx, diagnoser, apiKey, routingModel, displayModel, platform)
}

func openAICompatibleRequestPlatform(apiKey *service.APIKey) string {
	if apiKey != nil && apiKey.Group != nil && apiKey.Group.Platform == service.PlatformXAI {
		return service.PlatformXAI
	}
	return service.PlatformOpenAI
}

func classifyOpenAICompatibleNoAccountErrorFromGin(
	c *gin.Context,
	diagnoser service.ModelAvailabilityDiagnoser,
	apiKey *service.APIKey,
	routingModel string,
	displayModel string,
) noAccountErrorClassification {
	return classifyNoAccountErrorFromGin(
		c,
		diagnoser,
		apiKey,
		routingModel,
		displayModel,
		openAICompatibleRequestPlatform(apiKey),
	)
}

func openAICompatibleSelectionErrorForLog(err error, platform string) error {
	if err == nil || platform != service.PlatformXAI {
		return err
	}
	message := strings.ReplaceAll(err.Error(), "OpenAI accounts", "xAI accounts")
	if message == err.Error() {
		return err
	}
	return fmt.Errorf("%s", message)
}
