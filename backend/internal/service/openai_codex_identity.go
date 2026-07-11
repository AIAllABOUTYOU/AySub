package service

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

const codexUpstreamMinVersion = "0.144.0"

func enforceCodexIdentityHeaders(h http.Header) {
	if h == nil || h.Get("originator") == "" {
		return
	}
	originator, pairedUA, ok := openai.PairCodexClientIdentity(h.Get("user-agent"))
	if !ok {
		originator, pairedUA = "codex_cli_rs", codexCLIUserAgent
	}
	h.Set("user-agent", pairedUA)
	h.Set("originator", originator)
	if version := strings.TrimSpace(h.Get("version")); version != "" && CompareVersions(version, codexUpstreamMinVersion) < 0 {
		h.Set("version", codexCLIVersion)
	}
}
