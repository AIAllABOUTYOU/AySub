package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeCRSOpenAILongContextBillingExtra(t *testing.T) {
	tests := []struct {
		name     string
		existing map[string]any
		incoming map[string]any
		want     bool
	}{
		{name: "new account defaults false", incoming: map[string]any{}, want: false},
		{name: "new account ignores imported true", incoming: map[string]any{openAILongContextBillingEnabledKey: true}, want: false},
		{name: "update preserves existing true", existing: map[string]any{openAILongContextBillingEnabledKey: true}, incoming: map[string]any{"crs_kind": "oauth"}, want: true},
		{name: "update preserves existing true over imported false", existing: map[string]any{openAILongContextBillingEnabledKey: true}, incoming: map[string]any{openAILongContextBillingEnabledKey: false}, want: true},
		{name: "invalid incoming preserves existing false", existing: map[string]any{openAILongContextBillingEnabledKey: false}, incoming: map[string]any{openAILongContextBillingEnabledKey: "true"}, want: false},
		{name: "invalid existing falls back false", existing: map[string]any{openAILongContextBillingEnabledKey: "true"}, incoming: map[string]any{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCRSOpenAILongContextBillingExtra(tt.existing, tt.incoming)
			require.Equal(t, tt.want, got[openAILongContextBillingEnabledKey])
		})
	}
}
