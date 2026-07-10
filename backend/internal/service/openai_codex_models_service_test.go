package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func newCodexModelsTestAccount() *Account {
	return &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "test-access-token",
			"chatgpt_account_id": "acc-123",
		},
	}
}

func TestFetchCodexModelsManifestPassthrough(t *testing.T) {
	manifestBody := `{"models":[{"slug":"gpt-5.6","display_name":"GPT-5.6"}]}`
	var gotAuth, gotAccountID, gotOriginator, gotClientVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccountID = r.Header.Get("chatgpt-account-id")
		gotOriginator = r.Header.Get("Originator")
		gotClientVersion = r.URL.Query().Get("client_version")
		w.Header().Set("ETag", `W/"abc123"`)
		_, _ = w.Write([]byte(manifestBody))
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	defer func() { chatgptCodexModelsURL = original }()

	manifest, err := (&OpenAIGatewayService{}).FetchCodexModelsManifest(context.Background(), newCodexModelsTestAccount(), "0.144.1", "")
	require.NoError(t, err)
	require.Equal(t, manifestBody, string(manifest.Body))
	require.Equal(t, `W/"abc123"`, manifest.ETag)
	require.Equal(t, "Bearer test-access-token", gotAuth)
	require.Equal(t, "acc-123", gotAccountID)
	require.Equal(t, "codex_cli_rs", gotOriginator)
	require.Equal(t, "0.144.1", gotClientVersion)
}

func TestFetchCodexModelsManifestNotModified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, `W/"abc123"`, r.Header.Get("If-None-Match"))
		w.Header().Set("ETag", `W/"abc123"`)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	defer func() { chatgptCodexModelsURL = original }()

	manifest, err := (&OpenAIGatewayService{}).FetchCodexModelsManifest(context.Background(), newCodexModelsTestAccount(), "", `W/"abc123"`)
	require.NoError(t, err)
	require.True(t, manifest.NotModified)
	require.Equal(t, `W/"abc123"`, manifest.ETag)
}

func TestFetchCodexModelsManifestMissingToken(t *testing.T) {
	account := newCodexModelsTestAccount()
	delete(account.Credentials, "access_token")
	_, err := (&OpenAIGatewayService{}).FetchCodexModelsManifest(context.Background(), account, "", "")
	require.Error(t, err)
}
