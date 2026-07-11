package apicompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponsesUsageUnmarshalCacheCreationAliases(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"canonical", `{"cache_creation_input_tokens":3}`, 3},
		{"cache write input", `{"cache_write_input_tokens":4}`, 4},
		{"top level cache creation", `{"cache_creation_tokens":5}`, 5},
		{"nested input cache write", `{"cache_creation_input_tokens":3,"input_tokens_details":{"cache_write_tokens":6}}`, 6},
		{"nested prompt cache creation", `{"prompt_tokens_details":{"cache_creation_tokens":7}}`, 7},
		{"explicit nested zero wins", `{"cache_write_input_tokens":8,"input_tokens_details":{"cache_write_tokens":0}}`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var usage ResponsesUsage
			require.NoError(t, json.Unmarshal([]byte(tt.body), &usage))
			require.Equal(t, tt.want, usage.CacheCreationInputTokens)
		})
	}
}
