package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterCodexInputContinuationStripsInvalidMessageID(t *testing.T) {
	input := []any{
		map[string]any{"type": "message", "id": "item_123", "role": "user", "content": "hello"},
		map[string]any{"type": "message", "id": "msg_123", "role": "assistant", "content": "world"},
	}
	filtered := filterCodexInputWithOptions(input, codexInputFilterOptions{PreserveReferences: true})
	require.Len(t, filtered, 2)
	_, hasInvalidID := filtered[0].(map[string]any)["id"]
	require.False(t, hasInvalidID)
	require.Equal(t, "msg_123", filtered[1].(map[string]any)["id"])
}
