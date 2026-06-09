package service

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const grokDynamicStatsigHeader = "x-statsig-id"

func applyGrokWebCompatibilityHeaders(headers http.Header, account *Account, cfg *config.Config) {
	if headers == nil {
		return
	}
	ua := strings.TrimSpace(headers.Get("User-Agent"))
	if ua == "" && account != nil {
		ua = grokFirstNonEmpty(account.GetCredential("user_agent"), defaultGrokWebUserAgent())
	}
	applyGrokWebBrowserHeaders(headers, ua)
	if resolveGrokDynamicStatsigEnabled(account, cfg) {
		headers.Set(grokDynamicStatsigHeader, buildGrokDynamicStatsigID())
	}
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
	if cfg != nil {
		return cfg.Grok.DynamicStatsigEnabled
	}
	return true
}

func applyGrokWebBrowserHeaders(headers http.Header, userAgent string) {
	headers.Set("Baggage", "sentry-environment=production,sentry-release=d6add6fb0460641fd482d767a335ef72b9b6abb8,sentry-public_key=b311e0f2690c81f25e2c4cf6d4f7ce1c")
	headers.Set("Priority", "u=1, i")

	hints := grokWebClientHints(userAgent)
	for key, value := range hints {
		headers.Set(key, value)
	}
}

func grokWebClientHints(userAgent string) map[string]string {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" || strings.Contains(ua, "firefox") || (strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") && !strings.Contains(ua, "chromium") && !strings.Contains(ua, "edg")) {
		return nil
	}
	if !strings.Contains(ua, "chrome") && !strings.Contains(ua, "chromium") && !strings.Contains(ua, "edg") {
		return nil
	}

	version := grokWebMajorVersion(userAgent)
	if version == "" {
		return nil
	}
	brand := "Google Chrome"
	if strings.Contains(ua, "edg") {
		brand = "Microsoft Edge"
	} else if strings.Contains(ua, "chromium") {
		brand = "Chromium"
	}

	platform := grokWebUAPlatform(ua)
	mobile := "?0"
	if strings.Contains(ua, "mobile") || platform == "Android" || platform == "iOS" {
		mobile = "?1"
	}

	hints := map[string]string{
		"Sec-Ch-Ua":        `"` + brand + `";v="` + version + `", "Chromium";v="` + version + `", "Not(A:Brand";v="24"`,
		"Sec-Ch-Ua-Mobile": mobile,
		"Sec-Ch-Ua-Model":  "",
	}
	if platform != "" {
		hints["Sec-Ch-Ua-Platform"] = `"` + platform + `"`
	}
	if arch := grokWebUAArch(ua); arch != "" {
		hints["Sec-Ch-Ua-Arch"] = arch
		hints["Sec-Ch-Ua-Bitness"] = "64"
	}
	return hints
}

var grokWebMajorVersionRE = regexp.MustCompile(`(?i)(?:chrome|chromium|crios|edg)/(\d{2,3})`)

func grokWebMajorVersion(userAgent string) string {
	matches := grokWebMajorVersionRE.FindStringSubmatch(userAgent)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func grokWebUAPlatform(ua string) string {
	switch {
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "mac os x") || strings.Contains(ua, "macintosh"):
		return "macOS"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		return "iOS"
	case strings.Contains(ua, "linux"):
		return "Linux"
	default:
		return ""
	}
}

func grokWebUAArch(ua string) string {
	switch {
	case strings.Contains(ua, "aarch64") || strings.Contains(ua, "arm"):
		return "arm"
	case strings.Contains(ua, "x86_64") || strings.Contains(ua, "x64") || strings.Contains(ua, "win64") || strings.Contains(ua, "intel"):
		return "x86"
	default:
		return ""
	}
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
