package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWSPassthroughUsageMetaUsesMappedGPT56ModelForMaxEffort(t *testing.T) {
	body := []byte(`{"type":"response.create","model":"sol","reasoning":{"effort":"max"}}`)
	meta := newOpenAIWSPassthroughUsageMeta("sol", body)
	meta.initFromFirstFrame(body, "gpt-5.6-sol")
	require.NotNil(t, meta.reasoningEffort.Load())
	require.Equal(t, "max", *meta.reasoningEffort.Load())

	meta.updateFromResponseCreate(body, "gpt-5.4", "sol")
	require.Equal(t, "xhigh", *meta.reasoningEffort.Load())
}
