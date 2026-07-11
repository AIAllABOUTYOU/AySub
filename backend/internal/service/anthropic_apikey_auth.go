package service

import (
	"net/http"
	"strings"
)

const (
	anthropicAPIKeyAuthSchemeExtraKey = "anthropic_apikey_auth_scheme"

	AnthropicAPIKeyAuthSchemeXAPIKey             = "x_api_key"
	AnthropicAPIKeyAuthSchemeAuthorizationBearer = "authorization_bearer"
)

func (a *Account) GetAnthropicAPIKeyAuthScheme() string {
	if a == nil || a.Platform != PlatformAnthropic || a.Type != AccountTypeAPIKey {
		return AnthropicAPIKeyAuthSchemeXAPIKey
	}
	if strings.TrimSpace(a.GetExtraString(anthropicAPIKeyAuthSchemeExtraKey)) == AnthropicAPIKeyAuthSchemeAuthorizationBearer {
		return AnthropicAPIKeyAuthSchemeAuthorizationBearer
	}
	return AnthropicAPIKeyAuthSchemeXAPIKey
}

func setAnthropicAPIKeyAuthHeader(header http.Header, account *Account, token string) {
	if account.GetAnthropicAPIKeyAuthScheme() == AnthropicAPIKeyAuthSchemeAuthorizationBearer {
		header.Set("Authorization", "Bearer "+token)
		return
	}
	header.Set("x-api-key", token)
}
