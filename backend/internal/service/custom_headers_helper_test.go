package service

import (
	"net/http"
	"testing"
)

func TestApplyCustomHeaders(t *testing.T) {
	tests := []struct {
		name           string
		account        *Account
		expectedHeader map[string]string
	}{
		{
			name: "应用单个自定义请求头",
			account: &Account{
				Extra: map[string]interface{}{
					"custom_headers": map[string]interface{}{
						"X-Custom-Header": "test-value",
					},
				},
			},
			expectedHeader: map[string]string{
				"X-Custom-Header": "test-value",
			},
		},
		{
			name: "过滤禁止覆盖的认证头",
			account: &Account{
				Extra: map[string]interface{}{
					"custom_headers": map[string]interface{}{
						"X-Custom-Header": "test-value",
						"Authorization":   "Bearer token123",
						"X-API-Key":       "api-key-value",
					},
				},
			},
			expectedHeader: map[string]string{
				"X-Custom-Header": "test-value",
			},
		},
		{
			name: "账号没有custom_headers",
			account: &Account{
				Extra: map[string]interface{}{},
			},
			expectedHeader: map[string]string{},
		},
		{
			name:           "账号Extra为nil",
			account:        &Account{},
			expectedHeader: map[string]string{},
		},
		{
			name:           "账号为nil",
			account:        nil,
			expectedHeader: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "https://api.example.com", nil)
			if err != nil {
				t.Fatalf("创建请求失败: %v", err)
			}

			ApplyCustomHeaders(req, tt.account)

			// 验证请求头
			for key, expectedValue := range tt.expectedHeader {
				actualValue := req.Header.Get(key)
				if actualValue != expectedValue {
					t.Errorf("请求头 %s: 期望 %s, 实际 %s", key, expectedValue, actualValue)
				}
			}

			// 确保没有多余的自定义请求头
			if len(tt.expectedHeader) == 0 && len(req.Header) > 0 {
				// 检查是否只有系统默认的请求头
				for key := range req.Header {
					if key != "User-Agent" { // http.NewRequest 会自动添加 User-Agent
						t.Errorf("不应该有额外的请求头: %s", key)
					}
				}
			}
		})
	}
}

func TestApplyCustomHeaders_RemovesAllExistingCasingAndWritesSingleValue(t *testing.T) {
	account := &Account{Extra: map[string]interface{}{"custom_headers": map[string]interface{}{"x-test": "new"}}}
	req, _ := http.NewRequest("GET", "https://api.example.com", nil)
	req.Header["X-Test"] = []string{"old"}
	req.Header["x-test"] = []string{"duplicate"}
	ApplyCustomHeaders(req, account)
	if got := req.Header.Values("X-Test"); len(got) != 1 || got[0] != "new" {
		t.Fatalf("expected one normalized value, got %#v", got)
	}
}

func TestApplyCustomHeaders_BlocksTransportAndSessionHeaders(t *testing.T) {
	blocked := []string{"Authorization", "Content-Type", "Connection", "Sec-WebSocket-Key", "session_id", "chatgpt-account-id"}
	headers := map[string]interface{}{"X-Safe": "ok"}
	for _, name := range blocked {
		headers[name] = "unsafe"
	}
	req, _ := http.NewRequest("GET", "https://api.example.com", nil)
	ApplyCustomHeaders(req, &Account{Extra: map[string]interface{}{"custom_headers": headers}})
	for _, name := range blocked {
		if req.Header.Get(name) != "" {
			t.Fatalf("blocked header %s was applied", name)
		}
	}
	if req.Header.Get("X-Safe") != "ok" {
		t.Fatal("safe header was not applied")
	}
}

func TestApplyCustomHeaders_IgnoresInvalidTypes(t *testing.T) {
	account := &Account{
		Extra: map[string]interface{}{
			"custom_headers": map[string]interface{}{
				"Valid-Header":   "valid-value",
				"Invalid-Number": 123,         // 应该被忽略
				"Invalid-Bool":   true,        // 应该被忽略
				"Empty-Key":      "",          // 应该被忽略
				"":               "empty-key", // 应该被忽略
			},
		},
	}

	req, _ := http.NewRequest("GET", "https://api.example.com", nil)
	ApplyCustomHeaders(req, account)

	// 只有有效的字符串值应该被设置
	if req.Header.Get("Valid-Header") != "valid-value" {
		t.Errorf("有效的请求头未被设置")
	}

	// 无效的类型应该被忽略
	if req.Header.Get("Invalid-Number") != "" {
		t.Errorf("数字类型的请求头不应该被设置")
	}

	if req.Header.Get("Invalid-Bool") != "" {
		t.Errorf("布尔类型的请求头不应该被设置")
	}
}

func TestNormalizeCustomHeaders(t *testing.T) {
	extra := map[string]any{"custom_headers": map[string]any{" X-Test ": " value "}}
	if err := NormalizeCustomHeaders(extra); err != nil {
		t.Fatal(err)
	}
	got := extra["custom_headers"].(map[string]any)
	if got["x-test"] != "value" {
		t.Fatalf("unexpected normalized headers: %#v", got)
	}

	for _, name := range []string{"Authorization", "Content-Type", "Sec-WebSocket-Key"} {
		err := NormalizeCustomHeaders(map[string]any{"custom_headers": map[string]any{name: "unsafe"}})
		if err == nil {
			t.Fatalf("expected blocked header %s to fail", name)
		}
	}
}
