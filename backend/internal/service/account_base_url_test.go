//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/stretchr/testify/require"
)

func TestGetBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		account  Account
		expected string
	}{
		{
			name: "non-apikey type returns empty",
			account: Account{
				Type:     AccountTypeOAuth,
				Platform: PlatformAnthropic,
			},
			expected: "",
		},
		{
			name: "apikey without base_url returns default anthropic",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformAnthropic,
				Credentials: map[string]any{},
			},
			expected: "https://api.anthropic.com",
		},
		{
			name: "apikey with custom base_url",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformAnthropic,
				Credentials: map[string]any{"base_url": "https://custom.example.com"},
			},
			expected: "https://custom.example.com",
		},
		{
			name: "antigravity apikey auto-appends /antigravity",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformAntigravity,
				Credentials: map[string]any{"base_url": "https://upstream.example.com"},
			},
			expected: "https://upstream.example.com/antigravity",
		},
		{
			name: "antigravity apikey trims trailing slash before appending",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformAntigravity,
				Credentials: map[string]any{"base_url": "https://upstream.example.com/"},
			},
			expected: "https://upstream.example.com/antigravity",
		},
		{
			name: "antigravity non-apikey returns empty",
			account: Account{
				Type:        AccountTypeOAuth,
				Platform:    PlatformAntigravity,
				Credentials: map[string]any{"base_url": "https://upstream.example.com"},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.account.GetBaseURL()
			if result != tt.expected {
				t.Errorf("GetBaseURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetGeminiBaseURL(t *testing.T) {
	const defaultGeminiURL = "https://generativelanguage.googleapis.com"

	tests := []struct {
		name     string
		account  Account
		expected string
	}{
		{
			name: "apikey without base_url returns default",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformGemini,
				Credentials: map[string]any{},
			},
			expected: defaultGeminiURL,
		},
		{
			name: "apikey with custom base_url",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformGemini,
				Credentials: map[string]any{"base_url": "https://custom-gemini.example.com"},
			},
			expected: "https://custom-gemini.example.com",
		},
		{
			name: "antigravity apikey auto-appends /antigravity",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformAntigravity,
				Credentials: map[string]any{"base_url": "https://upstream.example.com"},
			},
			expected: "https://upstream.example.com/antigravity",
		},
		{
			name: "antigravity apikey trims trailing slash",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformAntigravity,
				Credentials: map[string]any{"base_url": "https://upstream.example.com/"},
			},
			expected: "https://upstream.example.com/antigravity",
		},
		{
			name: "antigravity oauth does NOT append /antigravity",
			account: Account{
				Type:        AccountTypeOAuth,
				Platform:    PlatformAntigravity,
				Credentials: map[string]any{"base_url": "https://upstream.example.com"},
			},
			expected: "https://upstream.example.com",
		},
		{
			name: "oauth without base_url returns default",
			account: Account{
				Type:        AccountTypeOAuth,
				Platform:    PlatformAntigravity,
				Credentials: map[string]any{},
			},
			expected: defaultGeminiURL,
		},
		{
			name: "nil credentials returns default",
			account: Account{
				Type:     AccountTypeAPIKey,
				Platform: PlatformGemini,
			},
			expected: defaultGeminiURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.account.GetGeminiBaseURL(defaultGeminiURL)
			if result != tt.expected {
				t.Errorf("GetGeminiBaseURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetOpenAIBaseURL_XAI(t *testing.T) {
	tests := []struct {
		name     string
		account  Account
		expected string
	}{
		{
			name: "xai apikey without base_url returns official xai url",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformXAI,
				Credentials: map[string]any{},
			},
			expected: "https://api.x.ai/v1",
		},
		{
			name: "xai apikey custom base_url wins",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformXAI,
				Credentials: map[string]any{"base_url": "https://xai-proxy.example.com"},
			},
			expected: "https://xai-proxy.example.com",
		},
		{
			name: "openai default remains unchanged",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformOpenAI,
				Credentials: map[string]any{},
			},
			expected: "https://api.openai.com",
		},
		{
			name: "non compatible platform returns empty",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformAnthropic,
				Credentials: map[string]any{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.account.GetOpenAIBaseURL()
			if result != tt.expected {
				t.Errorf("GetOpenAIBaseURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetGrokBaseURLKeepsXAIAccountTypesSeparated(t *testing.T) {
	tests := []struct {
		name     string
		account  Account
		expected string
	}{
		{
			name: "oauth third-party base URL stays pinned to CLI proxy by default",
			account: Account{
				Type:        AccountTypeOAuth,
				Platform:    PlatformXAI,
				Credentials: map[string]any{"base_url": "https://grok.example.test/v1"},
			},
			expected: xai.DefaultCLIBaseURL,
		},
		{
			name: "API key third-party base URL remains supported",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformXAI,
				Credentials: map[string]any{"base_url": "https://grok.example.test/v1"},
			},
			expected: "https://grok.example.test/v1",
		},
		{
			name: "Cookie Grok custom base URL remains supported",
			account: Account{
				Type:        AccountTypeCookie,
				Platform:    PlatformXAI,
				Credentials: map[string]any{"base_url": "https://grok.example.test"},
			},
			expected: "https://grok.example.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.account.GetGrokBaseURL())
		})
	}
}

func TestGetGrokBaseURLAllowsExplicitOAuthOverrideWhenUnsafeOverridesEnabled(t *testing.T) {
	t.Setenv(xai.EnvAllowUnsafeURLOverrides, "true")
	account := Account{
		Type:        AccountTypeOAuth,
		Platform:    PlatformXAI,
		Credentials: map[string]any{"base_url": "https://custom.example.com/v1"},
	}

	require.Equal(t, "https://custom.example.com/v1", account.GetGrokBaseURL())
}

func TestGetGrokMediaBaseURLSeparatesOAuthMediaFromCLIProxy(t *testing.T) {
	tests := []struct {
		name     string
		account  Account
		expected string
	}{
		{
			name: "oauth without base URL uses official media API",
			account: Account{
				Type:        AccountTypeOAuth,
				Platform:    PlatformXAI,
				Credentials: map[string]any{},
			},
			expected: xai.DefaultBaseURL,
		},
		{
			name: "oauth stored CLI proxy uses official media API",
			account: Account{
				Type:        AccountTypeOAuth,
				Platform:    PlatformXAI,
				Credentials: map[string]any{"base_url": xai.DefaultCLIBaseURL},
			},
			expected: xai.DefaultBaseURL,
		},
		{
			name: "oauth stored CLI proxy variant uses official media API",
			account: Account{
				Type:        AccountTypeOAuth,
				Platform:    PlatformXAI,
				Credentials: map[string]any{"base_url": "HTTPS://CLI-CHAT-PROXY.GROK.COM:443/%76%31/"},
			},
			expected: xai.DefaultBaseURL,
		},
		{
			name: "oauth untrusted custom base URL is pinned to official media API",
			account: Account{
				Type:        AccountTypeOAuth,
				Platform:    PlatformXAI,
				Credentials: map[string]any{"base_url": "https://custom.example.com/v1"},
			},
			expected: xai.DefaultBaseURL,
		},
		{
			name: "API key retains its configured media API",
			account: Account{
				Type:        AccountTypeAPIKey,
				Platform:    PlatformXAI,
				Credentials: map[string]any{"base_url": "https://grok.example.com/v1"},
			},
			expected: "https://grok.example.com/v1",
		},
		{
			name: "Cookie Grok retains its configured web base URL",
			account: Account{
				Type:        AccountTypeCookie,
				Platform:    PlatformXAI,
				Credentials: map[string]any{"base_url": "https://grok.com"},
			},
			expected: "https://grok.com",
		},
		{
			name: "non-xAI account has no Grok media base URL",
			account: Account{
				Type:        AccountTypeOAuth,
				Platform:    PlatformOpenAI,
				Credentials: map[string]any{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.account.GetGrokMediaBaseURL())
		})
	}
}

func TestGetGrokMediaBaseURLAllowsExplicitOAuthOverrideWhenUnsafeOverridesEnabled(t *testing.T) {
	t.Setenv(xai.EnvAllowUnsafeURLOverrides, "true")
	account := Account{
		Type:        AccountTypeOAuth,
		Platform:    PlatformXAI,
		Credentials: map[string]any{"base_url": "https://custom.example.com/v1"},
	}

	require.Equal(t, "https://custom.example.com/v1", account.GetGrokMediaBaseURL())
}
