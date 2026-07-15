package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestPatchGrokResponsesBodyRemovesNullReasoningContent(t *testing.T) {
	body := []byte(`{"model":"grok-4","input":[{"type":"reasoning","content":null,"summary":[]},{"role":"user","content":"hello"}]}`)

	patched, err := patchGrokResponsesBody(body, "grok-4")
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(patched, "input.0.content").Exists())
	require.Equal(t, "reasoning", gjson.GetBytes(patched, "input.0.type").String())
	require.Equal(t, "hello", gjson.GetBytes(patched, "input.1.content").String())
}

func TestPatchGrokResponsesBodyPreservesNonNullReasoningContent(t *testing.T) {
	body := []byte(`{"model":"grok-4","input":[{"type":"reasoning","content":[]},{"type":"message","content":null}]}`)

	patched, err := patchGrokResponsesBody(body, "grok-4")
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(patched, "input.0.content").Exists())
	require.True(t, gjson.GetBytes(patched, "input.1.content").Exists())
}
