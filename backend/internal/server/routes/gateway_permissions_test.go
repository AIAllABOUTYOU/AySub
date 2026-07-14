package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPermissionEndpointIDCodexResponsesFamily(t *testing.T) {
	for _, endpoint := range []string{
		handler.EndpointResponses,
		handler.EndpointResponsesCompact,
		handler.EndpointAlphaSearch,
	} {
		require.Equal(t, service.EndpointPermissionResponses, permissionEndpointID(endpoint))
	}
}
