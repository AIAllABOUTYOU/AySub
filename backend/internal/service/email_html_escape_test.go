//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildVerifyCodeEmailBody_EscapesSiteName(t *testing.T) {
	svc := &EmailService{}
	body := svc.buildVerifyCodeEmailBody("123456", `</h1><script>alert(1)</script><h1>`)
	assert.NotContains(t, body, "<script>")
	assert.Contains(t, body, "&lt;script&gt;")
	assert.Contains(t, svc.buildVerifyCodeEmailBody("654321", "My Site"), "<h1>My Site</h1>")
}

func TestBuildPasswordResetEmailBody_EscapesHTML(t *testing.T) {
	svc := &EmailService{}
	body := svc.buildPasswordResetEmailBody("https://example.com/reset?a=1&b=2", `</h1><img src=x onerror=alert(1)>`)
	assert.NotContains(t, body, "<img src=x")
	assert.True(t, strings.Contains(body, "&lt;img"))
	assert.Contains(t, body, `href="https://example.com/reset?a=1&amp;b=2"`)
}
