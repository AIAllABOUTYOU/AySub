//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterCodexInput_StripsFunctionCallItemID_WhenPreservingReferences(t *testing.T) {
	input := []any{
		map[string]any{
			"type":    "function_call",
			"id":      "item_A9v0SNfS3VaLrfX0j3y4xhyK",
			"call_id": "fc_abc123",
			"name":    "bash",
		},
		map[string]any{
			"type":    "function_call_output",
			"call_id": "fc_abc123",
			"output":  "done",
		},
	}

	filtered := filterCodexInputWithOptions(input, codexInputFilterOptions{PreserveReferences: true})
	require.Len(t, filtered, 2)

	fc, ok := filtered[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "function_call", fc["type"])
	require.NotContains(t, fc, "id")
	require.Equal(t, "fc_abc123", fc["call_id"])
}

func TestFilterCodexInput_KeepsFcID_WhenPreservingReferences(t *testing.T) {
	input := []any{
		map[string]any{
			"type":    "function_call",
			"id":      "fc_validID123",
			"call_id": "fc_validID123",
			"name":    "bash",
		},
	}

	filtered := filterCodexInputWithOptions(input, codexInputFilterOptions{PreserveReferences: true})
	require.Len(t, filtered, 1)
	fc, ok := filtered[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "fc_validID123", fc["id"])
}

func TestFilterCodexInput_StripsItemIDFromAllToolCallInputTypes(t *testing.T) {
	for _, typ := range []string{"function_call", "tool_call", "local_shell_call", "custom_tool_call", "mcp_tool_call"} {
		input := []any{
			map[string]any{
				"type":    typ,
				"id":      "item_xyz",
				"call_id": "fc_001",
				"name":    "tool",
			},
		}
		filtered := filterCodexInputWithOptions(input, codexInputFilterOptions{PreserveReferences: true})
		require.Len(t, filtered, 1)
		item, ok := filtered[0].(map[string]any)
		require.True(t, ok)
		require.NotContains(t, item, "id")
	}
}

func TestFilterCodexInput_OutputTypeKeepsItemID(t *testing.T) {
	input := []any{
		map[string]any{
			"type":    "function_call_output",
			"id":      "o1",
			"call_id": "fc_abc",
			"output":  "done",
		},
	}

	filtered := filterCodexInputWithOptions(input, codexInputFilterOptions{PreserveReferences: true})
	require.Len(t, filtered, 1)
	out, ok := filtered[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "o1", out["id"])
}

func TestFilterCodexInput_NonToolCallItemKeepsID(t *testing.T) {
	input := []any{
		map[string]any{
			"type": "message",
			"id":   "item_msg_001",
			"role": "user",
		},
	}

	filtered := filterCodexInputWithOptions(input, codexInputFilterOptions{PreserveReferences: true})
	require.Len(t, filtered, 1)
	msg, ok := filtered[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "item_msg_001", msg["id"])
}
