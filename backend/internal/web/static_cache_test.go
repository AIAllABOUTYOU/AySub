//go:build unit

package web

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStaticAssetCacheHeaders(t *testing.T) {
	header := make(http.Header)
	applyStaticAssetCacheHeaders(header, "assets/index-abc.js")
	require.Equal(t, staticAssetsCacheControl, header.Get("Cache-Control"))
	require.False(t, isLongCacheStaticPath("index.html"))
	require.True(t, isLongCacheStaticPath("/favicon.ico"))
}
