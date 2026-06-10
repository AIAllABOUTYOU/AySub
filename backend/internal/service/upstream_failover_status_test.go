package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayService_ShouldFailoverUpstreamError_IncludesTransientHTTPStatus(t *testing.T) {
	svc := &GatewayService{}

	require.True(t, svc.shouldFailoverUpstreamError(http.StatusRequestTimeout))
	require.True(t, svc.shouldFailoverUpstreamError(http.StatusTooManyRequests))
	require.True(t, svc.shouldFailoverUpstreamError(http.StatusInternalServerError))
	require.True(t, svc.shouldFailoverUpstreamError(529))
	require.False(t, svc.shouldFailoverUpstreamError(http.StatusBadRequest))
}

func TestOpenAIGatewayService_ShouldFailoverUpstreamError_IncludesTransientHTTPStatus(t *testing.T) {
	svc := &OpenAIGatewayService{}

	require.True(t, svc.shouldFailoverUpstreamError(http.StatusRequestTimeout))
	require.True(t, svc.shouldFailoverUpstreamError(http.StatusTooManyRequests))
	require.True(t, svc.shouldFailoverUpstreamError(http.StatusInternalServerError))
	require.True(t, svc.shouldFailoverUpstreamError(529))
	require.False(t, svc.shouldFailoverUpstreamError(http.StatusBadRequest))
}

func TestOpenAIPassthroughFailoverResponse_IncludesTransientHTTPStatus(t *testing.T) {
	require.True(t, shouldFailoverOpenAIPassthroughResponse(http.StatusRequestTimeout))
	require.True(t, shouldFailoverOpenAIPassthroughResponse(http.StatusTooManyRequests))
	require.True(t, shouldFailoverOpenAIPassthroughResponse(http.StatusBadGateway))
	require.True(t, shouldFailoverOpenAIPassthroughResponse(529))
	require.False(t, shouldFailoverOpenAIPassthroughResponse(http.StatusBadRequest))
}

func TestAntigravityGatewayService_ShouldFailoverUpstreamError_IncludesTransientHTTPStatus(t *testing.T) {
	svc := &AntigravityGatewayService{}

	require.True(t, svc.shouldFailoverUpstreamError(http.StatusRequestTimeout))
	require.True(t, svc.shouldFailoverUpstreamError(http.StatusTooManyRequests))
	require.True(t, svc.shouldFailoverUpstreamError(http.StatusInternalServerError))
	require.True(t, svc.shouldFailoverUpstreamError(529))
	require.False(t, svc.shouldFailoverUpstreamError(http.StatusBadRequest))
}
