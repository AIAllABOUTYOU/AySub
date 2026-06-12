package service

import (
	"net/http"
)

// ApplyCustomHeaders 从账号 extra 中读取自定义请求头并应用到 HTTP 请求
// 这个函数应该在发送上游请求之前调用
func ApplyCustomHeaders(req *http.Request, account *Account) {
	if req == nil || account == nil {
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
		if strValue, ok := value.(string); ok && key != "" && strValue != "" {
			req.Header.Set(key, strValue)
		}
	}
}
