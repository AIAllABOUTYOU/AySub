package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeModelAvailabilityDiagnoser struct {
	calls []fakeModelAvailabilityCall
	resp  service.ModelAvailabilityDiagnosis
}

type fakeModelAvailabilityCall struct {
	groupID  *int64
	model    string
	platform string
}

func (f *fakeModelAvailabilityDiagnoser) DiagnoseModelAvailabilityForPlatform(
	_ context.Context,
	groupID *int64,
	model string,
	platform string,
) service.ModelAvailabilityDiagnosis {
	f.calls = append(f.calls, fakeModelAvailabilityCall{groupID: groupID, model: model, platform: platform})
	return f.resp
}

func newNoAccountTestContext() *gin.Context {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
	return c
}

func TestClassifyNoAccountErrorModelNotSupportedReturns404(t *testing.T) {
	groupID := int64(42)
	diagnoser := &fakeModelAvailabilityDiagnoser{
		resp: service.ModelAvailabilityDiagnosis{HasAccountsInPool: true, HasModelSupport: false},
	}
	apiKey := &service.APIKey{GroupID: &groupID}

	classification := classifyNoAccountErrorFromGin(
		newNoAccountTestContext(), diagnoser, apiKey,
		"gpt-5.6", "gpt-5.6", service.PlatformOpenAI,
	)

	require.Equal(t, http.StatusNotFound, classification.Status)
	require.Equal(t, "model_not_found", classification.ErrType)
	require.True(t, classification.ModelNotFound)
	require.Contains(t, classification.Message, "gpt-5.6")
	require.Len(t, diagnoser.calls, 1)
	require.Equal(t, service.PlatformOpenAI, diagnoser.calls[0].platform)
}

func TestClassifyOpenAICompatibleNoAccountErrorUsesXAIPlatform(t *testing.T) {
	groupID := int64(43)
	diagnoser := &fakeModelAvailabilityDiagnoser{
		resp: service.ModelAvailabilityDiagnosis{HasAccountsInPool: true, HasModelSupport: false},
	}
	apiKey := &service.APIKey{
		GroupID: &groupID,
		Group:   &service.Group{ID: groupID, Platform: service.PlatformXAI},
	}

	classification := classifyOpenAICompatibleNoAccountErrorFromGin(
		newNoAccountTestContext(), diagnoser, apiKey, "grok-4.5", "grok-4.5",
	)

	require.True(t, classification.ModelNotFound)
	require.Len(t, diagnoser.calls, 1)
	require.Equal(t, service.PlatformXAI, diagnoser.calls[0].platform)
	require.EqualError(t,
		openAICompatibleSelectionErrorForLog(
			fmt.Errorf("no available OpenAI accounts supporting model: grok-4.5"),
			service.PlatformXAI,
		),
		"no available xAI accounts supporting model: grok-4.5",
	)
}

func TestClassifyNoAccountErrorTransientOrEmptyPoolStays503(t *testing.T) {
	groupID := int64(44)
	apiKey := &service.APIKey{GroupID: &groupID}

	for _, diagnosis := range []service.ModelAvailabilityDiagnosis{
		{HasAccountsInPool: true, HasModelSupport: true},
		{HasAccountsInPool: false, HasModelSupport: false},
	} {
		diagnoser := &fakeModelAvailabilityDiagnoser{resp: diagnosis}
		classification := classifyNoAccountErrorFromGin(
			newNoAccountTestContext(), diagnoser, apiKey,
			"gpt-5.6", "gpt-5.6", service.PlatformOpenAI,
		)
		require.Equal(t, http.StatusServiceUnavailable, classification.Status)
		require.False(t, classification.ModelNotFound)
	}
}

func TestClassifyNoAccountErrorInvalidInputsStay503(t *testing.T) {
	groupID := int64(45)
	classification := classifyNoAccountErrorFromGin(
		nil, nil, &service.APIKey{GroupID: &groupID},
		"gpt-5.6", "gpt-5.6", service.PlatformOpenAI,
	)
	require.Equal(t, http.StatusServiceUnavailable, classification.Status)
	require.False(t, classification.ModelNotFound)
}
