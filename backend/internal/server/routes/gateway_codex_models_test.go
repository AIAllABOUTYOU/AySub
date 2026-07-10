package routes

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayRoutesCodexModelsManifestPathIsRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()
	registered := make(map[string]bool)
	for _, route := range router.Routes() {
		if route.Method == http.MethodGet {
			registered[route.Path] = true
		}
	}
	require.True(t, registered["/backend-api/codex/models"])
	require.True(t, registered["/v1/models"])
}
