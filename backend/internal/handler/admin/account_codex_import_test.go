package admin

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestParseCodexSessionImportEntriesSupportsRawTokenJSONAndArray(t *testing.T) {
	token1 := "raw-access-token-1"
	token2 := buildCodexImportTestJWT(t, time.Now().Add(time.Hour), map[string]any{
		"email": "json@example.com",
	})
	token3 := "raw-access-token-3"

	req := CodexSessionImportRequest{
		Content: fmt.Sprintf("%s\n{\"accessToken\":%q}\n[%q]", token1, token2, token3),
	}

	entries, err := parseCodexSessionImportEntries(req)
	if err != nil {
		t.Fatalf("parseCodexSessionImportEntries error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}

	first, err := normalizeCodexImportEntry(entries[0])
	if err != nil {
		t.Fatalf("normalize raw token error = %v", err)
	}
	if first.Credentials["access_token"] != token1 {
		t.Fatalf("raw token access_token = %v, want %s", first.Credentials["access_token"], token1)
	}

	second, err := normalizeCodexImportEntry(entries[1])
	if err != nil {
		t.Fatalf("normalize json token error = %v", err)
	}
	if second.Email != "json@example.com" {
		t.Fatalf("email = %q, want json@example.com", second.Email)
	}

	third, err := normalizeCodexImportEntry(entries[2])
	if err != nil {
		t.Fatalf("normalize array token error = %v", err)
	}
	if third.Credentials["access_token"] != token3 {
		t.Fatalf("array token access_token = %v, want %s", third.Credentials["access_token"], token3)
	}
}

func TestParseCodexSessionImportEntriesFallsBackToLineModeForMixedJSONAndToken(t *testing.T) {
	req := CodexSessionImportRequest{
		Content: "{\"accessToken\":\"json-line-token\"}\nraw-line-token",
	}

	entries, err := parseCodexSessionImportEntries(req)
	if err != nil {
		t.Fatalf("parseCodexSessionImportEntries error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}

	first, err := normalizeCodexImportEntry(entries[0])
	if err != nil {
		t.Fatalf("normalize json line error = %v", err)
	}
	if first.Credentials["access_token"] != "json-line-token" {
		t.Fatalf("json line access_token = %v, want json-line-token", first.Credentials["access_token"])
	}

	second, err := normalizeCodexImportEntry(entries[1])
	if err != nil {
		t.Fatalf("normalize raw line error = %v", err)
	}
	if second.Credentials["access_token"] != "raw-line-token" {
		t.Fatalf("raw line access_token = %v, want raw-line-token", second.Credentials["access_token"])
	}
}

func TestNormalizeCodexSessionJSONExtractsCredentialsAndIgnoresSessionToken(t *testing.T) {
	accessToken := buildCodexImportTestJWT(t, time.Now().Add(time.Hour), map[string]any{
		"email": "claim@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "acct-from-claim",
			"chatgpt_user_id":    "user-from-claim",
			"chatgpt_plan_type":  "plus",
			"poid":               "org-from-claim",
		},
	})
	raw := map[string]any{
		"user": map[string]any{
			"id":    "user-from-json",
			"name":  "Sup OO",
			"email": "json@example.com",
			"image": "https://example.com/avatar.png",
		},
		"account": map[string]any{
			"id":       "acct-from-json",
			"planType": "free",
		},
		"accessToken":  accessToken,
		"sessionToken": "secret-session-token",
		"expires":      "2026-08-05T13:40:42.836Z",
	}

	item, err := normalizeCodexImportEntry(codexImportEntry{Index: 1, Value: raw})
	if err != nil {
		t.Fatalf("normalizeCodexImportEntry error = %v", err)
	}
	if item.Credentials["access_token"] != accessToken {
		t.Fatalf("access_token not stored")
	}
	if item.Credentials["email"] != "json@example.com" {
		t.Fatalf("email = %v, want json@example.com", item.Credentials["email"])
	}
	if item.Credentials["chatgpt_account_id"] != "acct-from-json" {
		t.Fatalf("chatgpt_account_id = %v, want acct-from-json", item.Credentials["chatgpt_account_id"])
	}
	if item.Credentials["chatgpt_user_id"] != "user-from-json" {
		t.Fatalf("chatgpt_user_id = %v, want user-from-json", item.Credentials["chatgpt_user_id"])
	}
	if item.Credentials["plan_type"] != "free" {
		t.Fatalf("plan_type = %v, want free", item.Credentials["plan_type"])
	}
	if _, ok := item.Credentials["session_token"]; ok {
		t.Fatalf("session_token should not be written to credentials")
	}
	if item.Extra["session_token_present"] != true {
		t.Fatalf("session_token_present = %v, want true", item.Extra["session_token_present"])
	}
	if item.Extra["session_expires_at"] != "2026-08-05T13:40:42Z" {
		t.Fatalf("session_expires_at = %v", item.Extra["session_expires_at"])
	}
	if item.TokenExpiresAt == nil {
		t.Fatalf("TokenExpiresAt should be parsed from accessToken")
	}
}

func TestParseCodexSessionImportEntriesSupportsConverterSub2APIAccountsDocument(t *testing.T) {
	token := buildCodexImportTestJWT(t, time.Now().Add(time.Hour), map[string]any{
		"email": "sub2api@example.com",
	})
	req := CodexSessionImportRequest{
		Content: fmt.Sprintf(`{
			"exported_at": "2026-06-08T00:00:00Z",
			"accounts": [{
				"name": "Sub2API Account",
				"platform": "openai",
				"type": "oauth",
				"credentials": {
					"access_token": %q,
					"chatgpt_account_id": "acct-sub2api",
					"chatgpt_user_id": "user-sub2api",
					"plan_type": "plus",
					"expires_at": "2026-08-05T13:40:42Z"
				}
			}]
		}`, token),
	}

	entries, err := parseCodexSessionImportEntries(req)
	if err != nil {
		t.Fatalf("parseCodexSessionImportEntries error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}

	item, err := normalizeCodexImportEntry(entries[0])
	if err != nil {
		t.Fatalf("normalizeCodexImportEntry error = %v", err)
	}
	if item.Name != "Sub2API Account" {
		t.Fatalf("name = %q, want Sub2API Account", item.Name)
	}
	if item.Credentials["chatgpt_account_id"] != "acct-sub2api" {
		t.Fatalf("chatgpt_account_id = %v, want acct-sub2api", item.Credentials["chatgpt_account_id"])
	}
	if item.Credentials["chatgpt_user_id"] != "user-sub2api" {
		t.Fatalf("chatgpt_user_id = %v, want user-sub2api", item.Credentials["chatgpt_user_id"])
	}
}

func TestNormalizeCodexImportSupportsNineRouterAndCodexManagerShapes(t *testing.T) {
	token := buildCodexImportTestJWT(t, time.Now().Add(time.Hour), map[string]any{})

	nineRouter, err := normalizeCodexImportEntry(codexImportEntry{Index: 1, Value: map[string]any{
		"provider":     "codex",
		"authType":     "oauth",
		"id":           "acct-9router",
		"accessToken":  token,
		"refreshToken": "refresh-9router",
		"providerSpecificData": map[string]any{
			"chatgptPlanType": "team",
		},
	}})
	if err != nil {
		t.Fatalf("normalize 9router error = %v", err)
	}
	if nineRouter.Credentials["chatgpt_account_id"] != "acct-9router" {
		t.Fatalf("9router account id = %v, want acct-9router", nineRouter.Credentials["chatgpt_account_id"])
	}
	if nineRouter.Credentials["plan_type"] != "team" {
		t.Fatalf("9router plan type = %v, want team", nineRouter.Credentials["plan_type"])
	}

	codexManager, err := normalizeCodexImportEntry(codexImportEntry{Index: 2, Value: map[string]any{
		"tokens": map[string]any{
			"access_token":        token,
			"refresh_token":       "refresh-cm",
			"chatgpt_account_id":  "acct-cm",
			"chatgpt_accountId":   "ignored",
			"chatgpt_account_id2": "ignored",
		},
		"meta": map[string]any{
			"label":              "codex-manager@example.com",
			"chatgpt_account_id": "acct-cm-meta",
		},
	}})
	if err != nil {
		t.Fatalf("normalize codex manager error = %v", err)
	}
	if codexManager.Credentials["email"] != "codex-manager@example.com" {
		t.Fatalf("codex manager email = %v, want codex-manager@example.com", codexManager.Credentials["email"])
	}
	if codexManager.Credentials["chatgpt_account_id"] != "acct-cm" {
		t.Fatalf("codex manager account id = %v, want acct-cm", codexManager.Credentials["chatgpt_account_id"])
	}
}

func TestMergeCodexImportCredentialsClearsStaleRefreshFieldsWhenIncomingHasNoRefreshToken(t *testing.T) {
	existing := map[string]any{
		"access_token":       "old-access-token",
		"refresh_token":      "old-refresh-token",
		"client_id":          "old-client-id",
		"id_token":           "old-id-token",
		"model_mapping":      map[string]any{"from": "existing"},
		"chatgpt_account_id": "acct-old",
		"unrelated_existing": "keep",
	}
	incoming := map[string]any{
		"access_token":       "new-access-token",
		"expires_at":         "2026-08-05T13:40:42Z",
		"chatgpt_account_id": "acct-new",
	}
	item := &codexImportAccount{
		AccessToken: "new-access-token",
	}

	merged := mergeCodexImportCredentials(existing, incoming, item)

	if merged["access_token"] != "new-access-token" {
		t.Fatalf("access_token = %v, want new-access-token", merged["access_token"])
	}
	if merged["chatgpt_account_id"] != "acct-new" {
		t.Fatalf("chatgpt_account_id = %v, want acct-new", merged["chatgpt_account_id"])
	}
	if _, ok := merged["refresh_token"]; ok {
		t.Fatalf("refresh_token should be cleared")
	}
	if _, ok := merged["client_id"]; ok {
		t.Fatalf("client_id should be cleared")
	}
	if _, ok := merged["id_token"]; ok {
		t.Fatalf("id_token should be cleared")
	}
	if merged["unrelated_existing"] != "keep" {
		t.Fatalf("unrelated_existing = %v, want keep", merged["unrelated_existing"])
	}
	if _, ok := merged["model_mapping"]; !ok {
		t.Fatalf("model_mapping should be preserved")
	}
}

func TestMergeCodexImportCredentialsKeepsRefreshFieldsWhenIncomingHasRefreshToken(t *testing.T) {
	existing := map[string]any{
		"refresh_token": "old-refresh-token",
		"client_id":     "old-client-id",
		"id_token":      "old-id-token",
	}
	incoming := map[string]any{
		"access_token":  "new-access-token",
		"refresh_token": "new-refresh-token",
		"client_id":     "new-client-id",
		"id_token":      "new-id-token",
	}
	item := &codexImportAccount{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		IDToken:      "new-id-token",
	}

	merged := mergeCodexImportCredentials(existing, incoming, item)

	if merged["refresh_token"] != "new-refresh-token" {
		t.Fatalf("refresh_token = %v, want new-refresh-token", merged["refresh_token"])
	}
	if merged["client_id"] != "new-client-id" {
		t.Fatalf("client_id = %v, want new-client-id", merged["client_id"])
	}
	if merged["id_token"] != "new-id-token" {
		t.Fatalf("id_token = %v, want new-id-token", merged["id_token"])
	}
}

func TestNormalizeCodexImportRejectsExpiredAccessToken(t *testing.T) {
	expiredToken := buildCodexImportTestJWT(t, time.Now().Add(-time.Hour), map[string]any{})

	_, err := normalizeCodexImportEntry(codexImportEntry{Index: 1, Value: expiredToken})
	if err == nil {
		t.Fatal("normalizeCodexImportEntry error = nil, want expired token error")
	}
	if !strings.Contains(err.Error(), "已过期") {
		t.Fatalf("error = %v, want expired token message", err)
	}
}

func TestResolveCodexImportExpiryForNoRefreshTokenUsesTokenExpiry(t *testing.T) {
	tokenExpiresAt := time.Now().Add(time.Hour).UTC()
	item := &codexImportAccount{
		AccessToken:    "access-token",
		Credentials:    map[string]any{"access_token": "access-token"},
		TokenExpiresAt: &tokenExpiresAt,
		WarningTexts:   []string{},
	}
	disabled := false
	req := CodexSessionImportRequest{AutoPauseOnExpired: &disabled}

	accountExpiresAt, credentialExpiresAt, autoPause, warnings, err := resolveCodexImportExpiry(req, item)
	if err != nil {
		t.Fatalf("resolveCodexImportExpiry error = %v", err)
	}
	if accountExpiresAt == nil || *accountExpiresAt != tokenExpiresAt.Unix() {
		t.Fatalf("account expires_at = %v, want %d", accountExpiresAt, tokenExpiresAt.Unix())
	}
	if credentialExpiresAt == nil || credentialExpiresAt.Unix() != tokenExpiresAt.Unix() {
		t.Fatalf("credential expires_at = %v, want %s", credentialExpiresAt, tokenExpiresAt)
	}
	if autoPause == nil || !*autoPause {
		t.Fatalf("autoPause = %v, want true", autoPause)
	}
	if len(warnings) == 0 {
		t.Fatalf("warnings should not be empty")
	}
}

func TestResolveCodexImportExpiryForNoRefreshTokenRequiresExpiry(t *testing.T) {
	item := &codexImportAccount{
		AccessToken:  "opaque-access-token",
		Credentials:  map[string]any{"access_token": "opaque-access-token"},
		WarningTexts: []string{},
	}

	_, _, _, _, err := resolveCodexImportExpiry(CodexSessionImportRequest{}, item)
	if err == nil {
		t.Fatal("resolveCodexImportExpiry error = nil, want missing expiry error")
	}
	if !strings.Contains(err.Error(), "无法解析 accessToken 过期时间") {
		t.Fatalf("error = %v, want missing expiry message", err)
	}
}

func TestResolveCodexImportExpiryForNoRefreshTokenUsesEarlierRequestExpiry(t *testing.T) {
	tokenExpiresAt := time.Now().Add(2 * time.Hour).UTC()
	requestExpiresAt := time.Now().Add(time.Hour).UTC()
	item := &codexImportAccount{
		AccessToken:    "access-token",
		Credentials:    map[string]any{"access_token": "access-token"},
		TokenExpiresAt: &tokenExpiresAt,
		WarningTexts:   []string{},
	}
	reqUnix := requestExpiresAt.Unix()
	req := CodexSessionImportRequest{ExpiresAt: &reqUnix}

	accountExpiresAt, credentialExpiresAt, _, _, err := resolveCodexImportExpiry(req, item)
	if err != nil {
		t.Fatalf("resolveCodexImportExpiry error = %v", err)
	}
	if accountExpiresAt == nil || *accountExpiresAt != requestExpiresAt.Unix() {
		t.Fatalf("account expires_at = %v, want %d", accountExpiresAt, requestExpiresAt.Unix())
	}
	if credentialExpiresAt == nil || credentialExpiresAt.Unix() != requestExpiresAt.Unix() {
		t.Fatalf("credential expires_at = %v, want %s", credentialExpiresAt, requestExpiresAt)
	}
}

func TestCodexIdentityKeysPreferStrongIdentifiers(t *testing.T) {
	keys := buildCodexIdentityKeys("acct-1", "user-1", "same@example.com", "token")
	for _, key := range keys {
		if strings.HasPrefix(key, "email:") || strings.HasPrefix(key, "account:") {
			t.Fatalf("user identity should not include weaker account/email fallback: %v", keys)
		}
	}
	if keys[0] != "user:user-1" {
		t.Fatalf("first key = %q, want user:user-1", keys[0])
	}

	keys = buildCodexIdentityKeys("acct-1", "", "same@example.com", "token")
	if keys[0] != "account:acct-1" {
		t.Fatalf("account fallback key = %q, want account:acct-1", keys[0])
	}

	keys = buildCodexIdentityKeys("", "", "same@example.com", "token")
	hasEmail := false
	for _, key := range keys {
		if key == "email:same@example.com" {
			hasEmail = true
		}
	}
	if !hasEmail {
		t.Fatalf("weak identity should include email fallback: %v", keys)
	}
}

func TestCodexIdentityDoesNotMergeDifferentUsersInSameAccount(t *testing.T) {
	first := buildCodexIdentityKeys("shared-account", "user-a", "a@example.com", "token-a")
	second := buildCodexIdentityKeys("shared-account", "user-b", "b@example.com", "token-b")
	seen := map[string]int{}
	markCodexIdentitySeen(seen, first, 1)

	if duplicateIndex, ok := firstSeenCodexIdentity(seen, second); ok {
		t.Fatalf("different users in same account matched duplicate index %d; first=%v second=%v", duplicateIndex, first, second)
	}
}

func TestPreviewCodexSessionImportReturnsSafeHTTPStatus401ItemsAndSourceCounts(t *testing.T) {
	req := CodexSessionImportRequest{
		Content: `[
			{"http_status":401,"name":"first","email":"first@example.com","account_id":"acct-first","access_token":"secret-first"},
			{"httpStatus":200,"accessToken":"secret-ok"}
		]`,
		Contents: []string{
			`{"meta":{"httpStatus":"401","label":"second@example.com","chatgpt_account_id":"acct-second"},"accessToken":"secret-second"}`,
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/preview", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	(&AccountHandler{}).PreviewCodexSessionImport(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Code int                       `json:"code"`
		Data CodexSessionPreviewResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Code != 0 {
		t.Fatalf("response code = %d", response.Code)
	}
	if response.Data.Total != 3 {
		t.Fatalf("total = %d, want 3", response.Data.Total)
	}
	if fmt.Sprint(response.Data.SourceEntryCounts) != "[2 1]" {
		t.Fatalf("source_entry_counts = %v, want [2 1]", response.Data.SourceEntryCounts)
	}
	if len(response.Data.HTTPStatus401) != 2 {
		t.Fatalf("len(http_status_401) = %d, want 2", len(response.Data.HTTPStatus401))
	}
	first := response.Data.HTTPStatus401[0]
	if first.Index != 1 || first.Name != "first" || first.Email != "first@example.com" || first.AccountID != "acct-first" || first.HTTPStatus != 401 {
		t.Fatalf("first preview item = %+v", first)
	}
	second := response.Data.HTTPStatus401[1]
	if second.Index != 3 || second.Email != "second@example.com" || second.AccountID != "acct-second" || second.HTTPStatus != 401 {
		t.Fatalf("second preview item = %+v", second)
	}
	responseText := recorder.Body.String()
	if strings.Contains(responseText, "secret-first") || strings.Contains(responseText, "secret-second") {
		t.Fatalf("preview response leaked credentials: %s", responseText)
	}
}

func TestParseCodexSessionImportEntriesAppliesGlobalIndexOffset(t *testing.T) {
	entries, sourceCounts, err := parseCodexSessionImportEntriesWithSourceCounts(CodexSessionImportRequest{
		Content:     `[{"accessToken":"first"},{"accessToken":"second"}]`,
		Contents:    []string{`{"accessToken":"third"}`},
		IndexOffset: 50,
	})
	if err != nil {
		t.Fatalf("parse entries: %v", err)
	}
	if len(entries) != 3 || entries[0].Index != 51 || entries[1].Index != 52 || entries[2].Index != 53 {
		t.Fatalf("entry indexes = %+v", entries)
	}
	if fmt.Sprint(sourceCounts) != "[2 1]" {
		t.Fatalf("source counts = %v, want [2 1]", sourceCounts)
	}
}

func TestCodexImportHTTPStatusSupportsCommonNumberAndStringFields(t *testing.T) {
	tests := []struct {
		name  string
		value map[string]any
	}{
		{name: "root snake number", value: map[string]any{"http_status": json.Number("401")}},
		{name: "root snake string", value: map[string]any{"http_status": "401"}},
		{name: "root camel number", value: map[string]any{"httpStatus": float64(401)}},
		{name: "root camel string", value: map[string]any{"httpStatus": "401"}},
		{name: "meta snake", value: map[string]any{"meta": map[string]any{"http_status": "401"}}},
		{name: "meta camel", value: map[string]any{"meta": map[string]any{"httpStatus": json.Number("401")}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, ok := codexImportHTTPStatus(tt.value)
			if !ok || status != 401 {
				t.Fatalf("codexImportHTTPStatus() = %d, %v; want 401, true", status, ok)
			}
		})
	}
}

func TestImportCodexSessionsFiltersHTTPStatus401BeforeNormalization(t *testing.T) {
	adminSvc := newStubAdminService()
	handler := &AccountHandler{adminService: adminSvc}
	req := CodexSessionImportRequest{
		Content:             `{"http_status":"401","name":"unauthorized","email":"bad@example.com"}`,
		Name:                "batch",
		IndexOffset:         50,
		TotalItems:          100,
		FilterHTTPStatus401: true,
	}
	entries, err := parseCodexSessionImportEntries(req)
	if err != nil {
		t.Fatalf("parse entries: %v", err)
	}

	result, err := handler.importCodexSessions(t.Context(), req, entries)
	if err != nil {
		t.Fatalf("importCodexSessions: %v", err)
	}
	if result.Total != 1 || result.Skipped != 1 || result.Created != 0 || result.Updated != 0 || result.Failed != 0 {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Items) != 1 || result.Items[0].Index != 51 || result.Items[0].Action != "skipped" || result.Items[0].Name != "batch #51" {
		t.Fatalf("items = %+v", result.Items)
	}
	if len(result.Warnings) != 1 || result.Warnings[0].Index != 51 || !strings.Contains(result.Warnings[0].Message, "401") {
		t.Fatalf("warnings = %+v", result.Warnings)
	}
	if len(adminSvc.createdAccounts) != 0 {
		t.Fatalf("created accounts = %d, want 0", len(adminSvc.createdAccounts))
	}
}

func TestImportCodexSessionsUsesTotalItemsForSingleItemFinalBatchName(t *testing.T) {
	adminSvc := newStubAdminService()
	handler := &AccountHandler{adminService: adminSvc}
	token := buildCodexImportTestJWT(t, time.Now().Add(time.Hour), map[string]any{
		"email": "last@example.com",
	})
	req := CodexSessionImportRequest{
		Content:     fmt.Sprintf(`{"access_token":%q,"refresh_token":"refresh-last"}`, token),
		Name:        "batch",
		IndexOffset: 100,
		TotalItems:  101,
	}
	entries, err := parseCodexSessionImportEntries(req)
	if err != nil {
		t.Fatalf("parse entries: %v", err)
	}
	if err := validateCodexImportWindow(req, len(entries)); err != nil {
		t.Fatalf("validate window: %v", err)
	}

	result, err := handler.importCodexSessions(t.Context(), req, entries)
	if err != nil {
		t.Fatalf("importCodexSessions: %v", err)
	}
	if result.Created != 1 || len(result.Items) != 1 || result.Items[0].Index != 101 || result.Items[0].Name != "batch #101" {
		t.Fatalf("result = %+v", result)
	}
	if len(adminSvc.createdAccounts) != 1 || adminSvc.createdAccounts[0].Name != "batch #101" {
		t.Fatalf("created accounts = %+v", adminSvc.createdAccounts)
	}
}

func TestImportCodexSessionsSkipsExistingAccountWhenUpdateDisabled(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{{
		ID:       42,
		Name:     "existing",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"chatgpt_user_id": "user-existing",
		},
	}}
	handler := &AccountHandler{adminService: adminSvc}
	updateExisting := false
	token := buildCodexImportTestJWT(t, time.Now().Add(time.Hour), map[string]any{
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_user_id": "user-existing",
		},
	})
	req := CodexSessionImportRequest{
		Content:        fmt.Sprintf(`{"access_token":%q,"refresh_token":"refresh"}`, token),
		UpdateExisting: &updateExisting,
	}
	entries, err := parseCodexSessionImportEntries(req)
	if err != nil {
		t.Fatalf("parse entries: %v", err)
	}

	result, err := handler.importCodexSessions(t.Context(), req, entries)
	if err != nil {
		t.Fatalf("importCodexSessions: %v", err)
	}
	if result.Skipped != 1 || result.Created != 0 || result.Updated != 0 || result.Failed != 0 {
		t.Fatalf("result = %+v", result)
	}
	if len(adminSvc.createdAccounts) != 0 {
		t.Fatalf("created accounts = %d, want 0", len(adminSvc.createdAccounts))
	}
	if len(result.Items) != 1 || result.Items[0].AccountID != 42 || result.Items[0].Action != "skipped" {
		t.Fatalf("items = %+v", result.Items)
	}
}

func TestValidateCodexImportWindow(t *testing.T) {
	tests := []struct {
		name       string
		req        CodexSessionImportRequest
		entryCount int
		wantError  bool
	}{
		{name: "negative offset", req: CodexSessionImportRequest{IndexOffset: -1}, entryCount: 1, wantError: true},
		{name: "negative total", req: CodexSessionImportRequest{TotalItems: -1}, entryCount: 1, wantError: true},
		{name: "offset past total", req: CodexSessionImportRequest{IndexOffset: 3, TotalItems: 2}, entryCount: 0, wantError: true},
		{name: "batch past total", req: CodexSessionImportRequest{IndexOffset: 1, TotalItems: 2}, entryCount: 2, wantError: true},
		{name: "valid final batch", req: CodexSessionImportRequest{IndexOffset: 100, TotalItems: 101}, entryCount: 1},
		{name: "unspecified total", req: CodexSessionImportRequest{IndexOffset: 100}, entryCount: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCodexImportWindow(tt.req, tt.entryCount)
			if (err != nil) != tt.wantError {
				t.Fatalf("validateCodexImportWindow error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSelectCodexImportWindowKeepsGlobalIndexes(t *testing.T) {
	entries := make([]codexImportEntry, 101)
	for index := range entries {
		entries[index] = codexImportEntry{Index: index + 1, Value: fmt.Sprintf("item-%d", index+1)}
	}

	selected, err := selectCodexImportWindow(entries, 50, 50, 50)
	if err != nil {
		t.Fatalf("selectCodexImportWindow: %v", err)
	}
	if len(selected) != 50 || selected[0].Index != 51 || selected[49].Index != 100 {
		t.Fatalf("selected indexes = first:%d last:%d len:%d", selected[0].Index, selected[len(selected)-1].Index, len(selected))
	}
	if selected[0].Value != "item-51" || selected[49].Value != "item-100" {
		t.Fatalf("selected values = first:%v last:%v", selected[0].Value, selected[49].Value)
	}

	final, err := selectCodexImportWindow(entries, 100, 1, 100)
	if err != nil {
		t.Fatalf("select final window: %v", err)
	}
	if len(final) != 1 || final[0].Index != 101 || final[0].Value != "item-101" {
		t.Fatalf("final window = %+v", final)
	}
}

func buildCodexImportTestJWT(t *testing.T, exp time.Time, extraClaims map[string]any) string {
	t.Helper()
	header := map[string]any{
		"alg": "none",
		"typ": "JWT",
	}
	claims := map[string]any{
		"sub": "user-from-sub",
		"exp": exp.Unix(),
		"iat": time.Now().Unix(),
	}
	for k, v := range extraClaims {
		claims[k] = v
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	claimBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimBytes) + "."
}
