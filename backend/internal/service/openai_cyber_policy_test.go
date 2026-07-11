package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAICyberPolicyMarkAndDetection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	hit, code, message := detectOpenAICyberPolicy([]byte(`{"response":{"error":{"code":"Cyber_Policy","message":" blocked "}}}`))
	require.True(t, hit)
	require.Equal(t, "cyber_policy", code)
	require.Equal(t, "blocked", message)
	MarkOpsCyberPolicy(c, CyberPolicyMark{Message: message, UpstreamStatus: 400})
	MarkOpsCyberPolicy(c, CyberPolicyMark{Message: "second"})
	require.Equal(t, "blocked", GetOpsCyberPolicy(c).Message)
	ClearOpsCyberPolicy(c)
	require.Nil(t, GetOpsCyberPolicy(c))
}

func TestDetectOpenAICyberPolicyRejectsMessageOnly(t *testing.T) {
	hit, _, _ := detectOpenAICyberPolicy([]byte(`{"error":{"type":"safety_error","message":"high-risk cyber activity"}}`))
	require.False(t, hit)
}
