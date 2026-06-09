package service

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const grokDynamicStatsigHeader = "x-statsig-id"

func applyGrokWebCompatibilityHeaders(headers http.Header, account *Account, cfg *config.Config) {
	if headers == nil || !resolveGrokDynamicStatsigEnabled(account, cfg) {
		return
	}
	headers.Set(grokDynamicStatsigHeader, buildGrokDynamicStatsigID())
}

func resolveGrokDynamicStatsigEnabled(account *Account, cfg *config.Config) bool {
	if account == nil || !account.IsXAICookie() {
		return false
	}
	for _, key := range []string{"dynamic_statsig_enabled", "grok_dynamic_statsig_enabled"} {
		if v, ok := accountBoolOverride(account.Credentials, key); ok {
			return v
		}
		if v, ok := accountBoolOverride(account.Extra, key); ok {
			return v
		}
	}
	return cfg != nil && cfg.Grok.DynamicStatsigEnabled
}

func buildGrokDynamicStatsigID() string {
	seed := strconv.FormatInt(time.Now().UnixNano(), 36)
	msg := "x1:TypeError: Cannot read properties of undefined (reading '" + seed + "')"
	return base64.StdEncoding.EncodeToString([]byte(msg))
}

func accountBoolOverride(values map[string]any, key string) (bool, bool) {
	if values == nil {
		return false, false
	}
	v, ok := values[key]
	if !ok || v == nil {
		return false, false
	}
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "1", "true", "yes", "on":
			return true, true
		case "0", "false", "no", "off":
			return false, true
		}
	case json.Number:
		if i, err := strconv.ParseInt(x.String(), 10, 64); err == nil {
			return i != 0, true
		}
	case float64:
		return x != 0, true
	case int:
		return x != 0, true
	case int64:
		return x != 0, true
	}
	return false, false
}
