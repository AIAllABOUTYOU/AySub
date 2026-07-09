//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasCompactionTriggerInInput(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want bool
	}{
		{
			name: "detects trigger",
			body: []byte(`{"model":"gpt-5.5","input":[{"type":"message","role":"user","content":"hello"},{"type":"compaction_trigger"}]}`),
			want: true,
		},
		{name: "trigger only", body: []byte(`{"input":[{"type":"compaction_trigger"}]}`), want: true},
		{name: "no trigger", body: []byte(`{"input":[{"type":"message","content":"hello"}]}`), want: false},
		{name: "empty input", body: []byte(`{"input":[]}`), want: false},
		{name: "missing input", body: []byte(`{"model":"gpt-5.5"}`), want: false},
		{name: "string input", body: []byte(`{"input":"compaction_trigger"}`), want: false},
		{name: "empty body", body: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, HasCompactionTriggerInInput(tt.body))
		})
	}
}
