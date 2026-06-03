//go:build unit

package service

import (
	"context"
	"testing"
)

// TestSettingKeyDefaultPlatformQuotas 验证新的系统层 JSON key 常量值正确。
func TestSettingKeyDefaultPlatformQuotas(t *testing.T) {
	if SettingKeyDefaultPlatformQuotas != "default_platform_quotas" {
		t.Errorf("SettingKeyDefaultPlatformQuotas = %q, want %q",
			SettingKeyDefaultPlatformQuotas, "default_platform_quotas")
	}
}

// TestSettingKeyAuthSourcePlatformQuotas 验证新的 auth-source JSON key 函数返回值正确。
func TestSettingKeyAuthSourcePlatformQuotas(t *testing.T) {
	if got := SettingKeyAuthSourcePlatformQuotas("email"); got != "auth_source_default_email_platform_quotas" {
		t.Fatalf("got %q, want %q", got, "auth_source_default_email_platform_quotas")
	}
	if got := SettingKeyAuthSourcePlatformQuotas("dingtalk"); got != "auth_source_default_dingtalk_platform_quotas" {
		t.Fatalf("got %q, want %q", got, "auth_source_default_dingtalk_platform_quotas")
	}
}

func TestOpenAICompatiblePlatforms(t *testing.T) {
	if !IsOpenAICompatiblePlatform(PlatformOpenAI) {
		t.Fatal("openai should be OpenAI-compatible")
	}
	if !IsOpenAICompatiblePlatform(PlatformXAI) {
		t.Fatal("xai should be OpenAI-compatible")
	}
	if IsOpenAICompatiblePlatform(PlatformAnthropic) {
		t.Fatal("anthropic should not be OpenAI-compatible")
	}
}

func TestOpenAICompatiblePlatformContext(t *testing.T) {
	ctx := WithOpenAICompatiblePlatform(context.Background(), PlatformXAI)
	if got := OpenAICompatiblePlatformFromContext(ctx); got != PlatformXAI {
		t.Fatalf("got %q, want %q", got, PlatformXAI)
	}

	ctx = WithOpenAICompatiblePlatform(context.Background(), PlatformAnthropic)
	if got := OpenAICompatiblePlatformFromContext(ctx); got != PlatformOpenAI {
		t.Fatalf("invalid compatible platform should fall back to %q, got %q", PlatformOpenAI, got)
	}
}
