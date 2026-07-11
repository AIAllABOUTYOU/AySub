package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func updateSettingsCodexStatus(t *testing.T, existing map[string]string, body map[string]any) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	values := map[string]string{service.SettingKeyPromoCodeEnabled: "true"}
	for key, value := range existing {
		values[key] = value
	}
	repo := &settingHandlerRepoStub{values: values}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(raw))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler.UpdateSettings(ctx)
	return recorder.Code
}

func TestUpdateSettings_CodexWhitelistValidation(t *testing.T) {
	require.Equal(t, http.StatusBadRequest, updateSettingsCodexStatus(t, nil, map[string]any{
		"codex_cli_only_whitelist": `[{"originator":"opencode"}]`,
	}))
	require.Equal(t, http.StatusOK, updateSettingsCodexStatus(t, nil, map[string]any{
		"codex_cli_only_whitelist": `[{"originator":"opencode","ua_contains":["opencode/"]}]`,
	}))
	require.Equal(t, http.StatusOK, updateSettingsCodexStatus(t, nil, map[string]any{
		"codex_cli_only_blacklist": `[{"originator":"evil"}]`,
	}))
}

func TestUpdateSettings_CodexVersionBoundsUseExistingPeer(t *testing.T) {
	require.Equal(t, http.StatusBadRequest, updateSettingsCodexStatus(t, map[string]string{
		service.SettingKeyMaxCodexVersion: "0.150.0",
	}, map[string]any{"min_codex_version": "0.160.0"}))
	require.Equal(t, http.StatusBadRequest, updateSettingsCodexStatus(t, map[string]string{
		service.SettingKeyMinCodexVersion: "0.160.0",
	}, map[string]any{"max_codex_version": "0.150.0"}))
	require.Equal(t, http.StatusOK, updateSettingsCodexStatus(t, map[string]string{
		service.SettingKeyMinCodexVersion: "0.150.0",
	}, map[string]any{"max_codex_version": "0.160.0"}))
}
