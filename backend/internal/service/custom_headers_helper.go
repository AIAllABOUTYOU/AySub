package service

import (
	"net/http"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/net/http/httpguts"
)

const (
	maxCustomHeaderNameLength  = 200
	maxCustomHeaderValueLength = 8192
	maxCustomHeaderEntries     = 64
)

var blockedCustomHeaderNames = map[string]struct{}{
	"host": {}, "content-length": {}, "content-type": {}, "transfer-encoding": {},
	"connection": {}, "keep-alive": {}, "proxy-authenticate": {}, "proxy-authorization": {},
	"proxy-connection": {}, "te": {}, "trailer": {}, "upgrade": {},
	"authorization": {}, "x-api-key": {}, "x-goog-api-key": {}, "cookie": {},
	"accept-encoding": {}, "sec-websocket-key": {}, "sec-websocket-version": {},
	"sec-websocket-extensions": {}, "sec-websocket-protocol": {}, "sec-websocket-accept": {},
	"session_id": {}, "conversation_id": {}, "x-codex-turn-state": {}, "x-codex-turn-metadata": {},
	"chatgpt-account-id": {}, "x-claude-code-session-id": {}, "x-client-request-id": {},
}

// NormalizeCustomHeaders validates and normalizes account.extra.custom_headers
// at the persistence boundary. Runtime filtering remains as defense in depth for
// legacy rows written before these rules existed.
func NormalizeCustomHeaders(extra map[string]any) error {
	if extra == nil {
		return nil
	}
	raw, ok := extra["custom_headers"]
	if !ok || raw == nil {
		return nil
	}
	entries, ok := raw.(map[string]any)
	if !ok {
		return infraerrors.New(http.StatusBadRequest, "INVALID_CUSTOM_HEADERS", "custom_headers must be an object of header name to string value")
	}
	if len(entries) > maxCustomHeaderEntries {
		return infraerrors.Newf(http.StatusBadRequest, "INVALID_CUSTOM_HEADERS", "custom_headers supports at most %d entries", maxCustomHeaderEntries)
	}
	normalized := make(map[string]any, len(entries))
	for rawName, rawValue := range entries {
		value, ok := rawValue.(string)
		if !ok {
			return infraerrors.Newf(http.StatusBadRequest, "INVALID_CUSTOM_HEADERS", "header %q value must be a string", rawName)
		}
		name := strings.ToLower(strings.TrimSpace(rawName))
		value = strings.TrimSpace(value)
		if name == "" {
			if value == "" {
				continue
			}
			return infraerrors.New(http.StatusBadRequest, "INVALID_CUSTOM_HEADERS", "header name must not be empty")
		}
		if _, blocked := blockedCustomHeaderNames[name]; blocked {
			return infraerrors.Newf(http.StatusBadRequest, "INVALID_CUSTOM_HEADERS", "header %q is not allowed to be overridden", name)
		}
		if len(name) > maxCustomHeaderNameLength || !httpguts.ValidHeaderFieldName(name) {
			return infraerrors.Newf(http.StatusBadRequest, "INVALID_CUSTOM_HEADERS", "invalid header name %q", name)
		}
		if len(value) > maxCustomHeaderValueLength || !httpguts.ValidHeaderFieldValue(value) {
			return infraerrors.Newf(http.StatusBadRequest, "INVALID_CUSTOM_HEADERS", "header %q has an invalid value", name)
		}
		if _, duplicate := normalized[name]; duplicate {
			return infraerrors.Newf(http.StatusBadRequest, "INVALID_CUSTOM_HEADERS", "duplicate header name %q", name)
		}
		normalized[name] = value
	}
	extra["custom_headers"] = normalized
	return nil
}

// ApplyCustomHeaders 从账号 extra 中读取自定义请求头并应用到 HTTP 请求
// 这个函数应该在发送上游请求之前调用
func ApplyCustomHeaders(req *http.Request, account *Account) {
	if req == nil || account == nil {
		return
	}
	ApplyCustomHeaderValues(req.Header, account)
}

func ApplyCustomHeaderValues(header http.Header, account *Account) {
	if header == nil || account == nil {
		return
	}

	// 从账号的 Extra 字段中读取 custom_headers
	if account.Extra == nil {
		return
	}

	customHeaders, ok := account.Extra["custom_headers"]
	if !ok {
		return
	}

	// 类型断言为 map[string]interface{}
	headersMap, ok := customHeaders.(map[string]interface{})
	if !ok {
		return
	}

	// 遍历并设置每个请求头
	for key, value := range headersMap {
		strValue, ok := value.(string)
		name := strings.ToLower(strings.TrimSpace(key))
		strValue = strings.TrimSpace(strValue)
		_, blocked := blockedCustomHeaderNames[name]
		if !ok || name == "" || strValue == "" || blocked || len(name) > maxCustomHeaderNameLength || len(strValue) > maxCustomHeaderValueLength || !httpguts.ValidHeaderFieldName(name) || !httpguts.ValidHeaderFieldValue(strValue) {
			continue
		}
		for existing := range header {
			if strings.EqualFold(existing, name) {
				delete(header, existing)
			}
		}
		header[http.CanonicalHeaderKey(name)] = []string{strValue}
	}
}
