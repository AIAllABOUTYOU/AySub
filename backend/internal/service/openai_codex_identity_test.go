package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureCodexIdentityHeaders(t *testing.T) {
	h := make(http.Header)
	ensureCodexIdentityHeaders(h)
	enforceCodexIdentityHeaders(h)

	require.Equal(t, "codex_cli_rs", h.Get("originator"))
	require.Equal(t, codexCLIUserAgent, h.Get("user-agent"))
	require.Equal(t, codexCLIVersion, h.Get("version"))
	require.Equal(t, "responses=experimental", h.Get("OpenAI-Beta"))
}

func TestEnforceCodexIdentityHeaders(t *testing.T) {
	const tuiUA = "codex-tui/0.140.2 (Mac OS X 14.0; arm64) iTerm (codex-tui; 0.140.2)"
	tests := []struct {
		name, originator, userAgent, version string
		wantOriginator, wantUA, wantVersion  string
	}{
		{"repair mismatch", "codex_cli_rs", tuiUA, "", "codex-tui", tuiUA, ""},
		{"fallback third party", "opencode", "luna/1.0.0", "", "codex_cli_rs", codexCLIUserAgent, ""},
		{"recover trailer identity", "cccc", "cccc/0.142.0 (Ubuntu; x86_64) screen (codex-tui; 0.142.0)", "", "codex-tui", "codex-tui/0.142.0 (Ubuntu; x86_64) screen (codex-tui; 0.142.0)", ""},
		{"raise old version", "codex_cli_rs", "codex_cli_rs/0.125.0", "0.125.0", "codex_cli_rs", "codex_cli_rs/0.125.0", codexCLIVersion},
		{"keep current version", "codex_cli_rs", "codex_cli_rs/0.145.0", "0.145.0", "codex_cli_rs", "codex_cli_rs/0.145.0", "0.145.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := make(http.Header)
			h.Set("originator", tt.originator)
			h.Set("user-agent", tt.userAgent)
			if tt.version != "" {
				h.Set("version", tt.version)
			}
			enforceCodexIdentityHeaders(h)
			require.Equal(t, tt.wantOriginator, h.Get("originator"))
			require.Equal(t, tt.wantUA, h.Get("user-agent"))
			require.Equal(t, tt.wantVersion, h.Get("version"))
		})
	}
}

func TestEnforceCodexIdentityHeaders_NoOriginatorIsNoop(t *testing.T) {
	h := http.Header{"User-Agent": []string{"luna/1.0.0"}}
	enforceCodexIdentityHeaders(h)
	require.Empty(t, h.Get("originator"))
	require.Equal(t, "luna/1.0.0", h.Get("user-agent"))
}
