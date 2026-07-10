package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractOpenAIReasoningEffortFromBodyModelCandidates(t *testing.T) {
	tests := []struct {
		name       string
		body       []byte
		candidates []string
		want       string
	}{
		{
			name:       "suffix fallback",
			body:       []byte(`{"model":"whatever","input":"hello"}`),
			candidates: []string{"gpt-5.4", "gpt-5.4", "gpt-5.4-xhigh"},
			want:       "xhigh",
		},
		{
			name:       "gpt56 suffix max",
			body:       []byte(`{"model":"whatever","input":"hello"}`),
			candidates: []string{"gpt-5.6-sol", "gpt-5.6-sol-max"},
			want:       "max",
		},
		{
			name:       "explicit mapped max",
			body:       []byte(`{"model":"sol","reasoning":{"effort":"max"},"input":"hello"}`),
			candidates: []string{"gpt-5.6-sol", "sol"},
			want:       "max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractOpenAIReasoningEffortFromBody(tt.body, tt.candidates...)
			require.NotNil(t, got)
			require.Equal(t, tt.want, *got)
		})
	}
}

func TestExtractOpenAIReasoningEffortModelCandidates(t *testing.T) {
	got := extractOpenAIReasoningEffort(
		map[string]any{"model": "gpt-5.3-codex-high", "input": "hello"},
		"gpt-5.3-codex", "gpt-5.3-codex-high",
	)
	require.NotNil(t, got)
	require.Equal(t, "high", *got)
}
