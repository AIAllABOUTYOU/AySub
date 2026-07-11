package apicompat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicUsageFromResponsesUsageCacheCreation(t *testing.T) {
	usage := &ResponsesUsage{
		InputTokens:              20,
		OutputTokens:             5,
		CacheCreationInputTokens: 6,
		InputTokensDetails:       &ResponsesInputTokensDetails{CachedTokens: 4},
	}

	got := anthropicUsageFromResponsesUsage(usage)
	assert.Equal(t, 10, got.InputTokens)
	assert.Equal(t, 5, got.OutputTokens)
	assert.Equal(t, 4, got.CacheReadInputTokens)
	assert.Equal(t, 6, got.CacheCreationInputTokens)
}

func TestAnthropicToResponsesResponseCacheCreation(t *testing.T) {
	out := AnthropicToResponsesResponse(&AnthropicResponse{
		ID: "msg_test", Type: "message", Role: "assistant", Model: "claude-opus-4-6",
		Usage:      AnthropicUsage{InputTokens: 10, OutputTokens: 5, CacheReadInputTokens: 4, CacheCreationInputTokens: 6},
		StopReason: "end_turn",
	})

	require.NotNil(t, out.Usage)
	assert.Equal(t, 20, out.Usage.InputTokens)
	assert.Equal(t, 6, out.Usage.CacheCreationInputTokens)
}
