//go:build embed || unit

package web

import (
	"net/http"
	"strings"
)

const staticAssetsCacheControl = "public, max-age=31536000, immutable"

func isLongCacheStaticPath(cleanPath string) bool {
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	return strings.HasPrefix(cleanPath, "assets/") || cleanPath == "logo.png" || cleanPath == "favicon.ico"
}

func applyStaticAssetCacheHeaders(header http.Header, cleanPath string) {
	if header != nil && isLongCacheStaticPath(cleanPath) {
		header.Set("Cache-Control", staticAssetsCacheControl)
	}
}
