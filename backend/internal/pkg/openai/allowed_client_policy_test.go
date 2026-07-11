package openai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchClientEntries_WhitelistAND(t *testing.T) {
	entries := []AllowedClientEntry{{Originator: "opencode", UAContains: []string{"opencode/"}}}
	require.True(t, MatchClientEntries("opencode/1.2", "opencode", entries))
	require.False(t, MatchClientEntries("opencode/1.2", "other", entries))
	require.False(t, MatchClientEntries("curl/8", "opencode", entries))
}

func TestMatchDenyEntries_BlacklistOR(t *testing.T) {
	entries := []AllowedClientEntry{
		{Originator: "evilbot"},
		{UAContains: []string{"badscan/"}},
	}
	require.True(t, MatchDenyEntries("anything/1", "evilbot", entries))
	require.True(t, MatchDenyEntries("badscan/9", "other", entries))
	require.False(t, MatchDenyEntries("codex_cli_rs/0.141.0", "codex_cli_rs", entries))
	require.False(t, MatchDenyEntries("x", "y", []AllowedClientEntry{{}}))
}

func TestMatchClientEntry_ReturnsFingerprintPolicy(t *testing.T) {
	entries := []AllowedClientEntry{
		{Originator: "opencode", UAContains: []string{"opencode/"}, SkipEngineFingerprint: true},
		{Originator: "Claude Code", UAContains: []string{"Claude Code/"}},
	}
	entry, ok := MatchClientEntry("opencode/1.0", "opencode", entries)
	require.True(t, ok)
	require.True(t, entry.SkipEngineFingerprint)
	entry, ok = MatchClientEntry("Claude Code/1.0", "Claude Code", entries)
	require.True(t, ok)
	require.False(t, entry.SkipEngineFingerprint)
}

func TestAllowedClientEntry_IsWhitelistable(t *testing.T) {
	tests := []struct {
		name  string
		entry AllowedClientEntry
		want  bool
	}{
		{name: "complete", entry: AllowedClientEntry{Originator: "opencode", UAContains: []string{"opencode/"}}, want: true},
		{name: "missing originator", entry: AllowedClientEntry{UAContains: []string{"opencode/"}}, want: false},
		{name: "missing user agent marker", entry: AllowedClientEntry{Originator: "opencode"}, want: false},
		{name: "blank marker", entry: AllowedClientEntry{Originator: "opencode", UAContains: []string{"opencode/", ""}}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, test.entry.IsWhitelistable())
		})
	}
}
