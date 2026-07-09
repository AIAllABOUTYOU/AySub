//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func newOpenAIOAuthAccountForModelTest() *Account {
	return &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	}
}

func TestIsModelSupported_OpenAIOAuthEmptyMapping_ServableModels(t *testing.T) {
	account := newOpenAIOAuthAccountForModelTest()

	servable := []string{
		"",
		"gpt-5.6-sol",
		"gpt-5.4",
		"gpt-5.4-high",
		"gpt-5.3-codex",
		"gpt-5.1-codex-mini",
		"gpt-5",
		"codex-mini-latest",
		"gpt5.3codexspark",
		"gpt-image-1",
		"claude-sonnet-4-6",
		"claude-3-opus-20240229",
		"gpt-4o",
		"my-custom-alias",
	}
	for _, model := range servable {
		require.True(t, account.IsModelSupported(model), "expected %q to be servable by empty-mapping OpenAI OAuth account", model)
	}
}

func TestIsModelSupported_OpenAIOAuthEmptyMapping_RejectsForeignModels(t *testing.T) {
	account := newOpenAIOAuthAccountForModelTest()

	foreign := []string{
		"deepseek-v4",
		"deepseek-chat",
		"glm-4.7",
		"kimi-k2",
		"moonshot-v1-128k",
		"gemini-3.0-pro",
		"grok-4",
		"qwen3-max",
		"minimax-m2.5",
		"llama-3.3-70b",
		"provider/deepseek-v4",
	}
	for _, model := range foreign {
		require.False(t, account.IsModelSupported(model), "expected %q to be rejected by empty-mapping OpenAI OAuth account", model)
	}
}

func TestIsModelSupported_OpenAIOAuthExplicitMappingUnchanged(t *testing.T) {
	account := newOpenAIOAuthAccountForModelTest()
	account.Credentials = map[string]any{
		"model_mapping": map[string]any{"deepseek-v4": "gpt-5.4"},
	}

	require.True(t, account.IsModelSupported("deepseek-v4"))
	require.False(t, account.IsModelSupported("glm-4.7"))
}

func TestIsModelSupported_OpenAIOAuthPassthroughAllowsAll(t *testing.T) {
	account := newOpenAIOAuthAccountForModelTest()
	account.Extra = map[string]any{"openai_passthrough": true}

	require.True(t, account.IsModelSupported("deepseek-v4"))
}

func TestIsModelSupported_OpenAIAPIKeyEmptyMappingAllowsAll(t *testing.T) {
	account := &Account{
		ID:       2,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
	}

	require.True(t, account.IsModelSupported("deepseek-v4"))
	require.True(t, account.IsModelSupported("gpt-5.4"))
}

func TestIsModelSupported_NonOpenAIPlatformsUnchanged(t *testing.T) {
	anthropic := &Account{ID: 3, Platform: PlatformAnthropic, Type: AccountTypeOAuth}
	require.True(t, anthropic.IsModelSupported("claude-sonnet-4-6"))
	require.True(t, anthropic.IsModelSupported("deepseek-v4"))
}

func TestIsOpenAIOAuthServableModel(t *testing.T) {
	require.True(t, isOpenAIOAuthServableModel("gpt-5.6-terra"))
	require.True(t, isOpenAIOAuthServableModel("gpt-5.4-high"))
	require.True(t, isOpenAIOAuthServableModel("  gpt-5.3-codex  "))
	require.True(t, isOpenAIOAuthServableModel("claude-3-5-haiku-20241022"))
	require.True(t, isOpenAIOAuthServableModel("DeepThink-x"))
	require.False(t, isOpenAIOAuthServableModel("DeepSeek-V4"))
	require.False(t, isOpenAIOAuthServableModel("qwen3-235b-thinking"))
}
