package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCodexClientEntriesJSON(t *testing.T) {
	require.NoError(t, ValidateCodexClientEntriesJSON(""))
	require.NoError(t, ValidateCodexClientEntriesJSON(`[]`))
	require.NoError(t, ValidateCodexClientEntriesJSON(`[{"originator":"evil"}]`))
	require.Error(t, ValidateCodexClientEntriesJSON("not-json"))
	require.Error(t, ValidateCodexClientEntriesJSON(`{"originator":"x"}`))
	require.Error(t, ValidateCodexClientEntriesJSON(`[1,2,3]`))
}

func TestValidateCodexWhitelistEntriesJSON(t *testing.T) {
	require.NoError(t, ValidateCodexWhitelistEntriesJSON(""))
	require.NoError(t, ValidateCodexWhitelistEntriesJSON(`[{"originator":"opencode","ua_contains":["opencode/"]}]`))
	require.Error(t, ValidateCodexWhitelistEntriesJSON(`[{"originator":"opencode"}]`))
	require.Error(t, ValidateCodexWhitelistEntriesJSON(`[{"originator":"x","ua_contains":[""]}]`))
}

func TestValidateEngineFingerprintSignalsJSON(t *testing.T) {
	require.NoError(t, ValidateEngineFingerprintSignalsJSON(""))
	require.NoError(t, ValidateEngineFingerprintSignalsJSON(`[{"type":"header_prefix","match":["x-codex-"],"required":true}]`))
	require.Error(t, ValidateEngineFingerprintSignalsJSON(`[{"type":"bogus","match":["x"]}]`))
}
