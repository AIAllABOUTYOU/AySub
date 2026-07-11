package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestTypeCyberBlocked(t *testing.T) {
	require.True(t, RequestTypeCyberBlocked.IsValid())
	require.Equal(t, "cyber", RequestTypeCyberBlocked.String())
	rt, err := ParseUsageRequestType("cyber")
	require.NoError(t, err)
	require.Equal(t, RequestTypeCyberBlocked, rt)
	u := &UsageLog{RequestType: RequestTypeCyberBlocked, Stream: true}
	require.Equal(t, RequestTypeCyberBlocked, u.EffectiveRequestType())
}
