package service

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func openAICompatFailedResponseMessage(response *apicompat.ResponsesResponse) string {
	if response != nil && response.Error != nil {
		if message := strings.TrimSpace(response.Error.Message); message != "" {
			return sanitizeUpstreamErrorMessage(message)
		}
	}
	return "OpenAI response failed"
}

func openAIStreamFailedEventSemanticStatus(payload []byte, message string) int {
	if isOpenAIContextWindowError(message, payload) {
		return http.StatusBadRequest
	}

	code := strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "response.error.code").String()))
	if code == "" {
		code = strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "error.code").String()))
	}
	errType := strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "response.error.type").String()))
	if errType == "" {
		errType = strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "error.type").String()))
	}
	combined := strings.TrimSpace(errType + " " + code + " " + strings.ToLower(strings.TrimSpace(message)))
	switch {
	case strings.Contains(errType, "invalid_request"):
		return http.StatusBadRequest
	case strings.Contains(combined, "rate_limit"):
		return http.StatusTooManyRequests
	case strings.Contains(combined, "authentication") || strings.Contains(combined, "unauthorized") || strings.Contains(combined, "invalid_api_key"):
		return http.StatusUnauthorized
	case strings.Contains(combined, "permission") || strings.Contains(combined, "forbidden") || strings.Contains(combined, "access denied"):
		return http.StatusForbidden
	case code == "server_is_overloaded" || code == "slow_down":
		return http.StatusServiceUnavailable
	default:
		return http.StatusBadGateway
	}
}

func openAIStreamFailedEventPassthroughBody(payload []byte, failedMessage string) []byte {
	if len(payload) == 0 || !gjson.ValidBytes(payload) {
		return payload
	}
	if gjson.GetBytes(payload, "error").Exists() {
		return payload
	}
	responseError := gjson.GetBytes(payload, "response.error")
	if !responseError.Exists() {
		if strings.TrimSpace(failedMessage) == "" {
			return payload
		}
		body, err := marshalOpenAIUpstreamJSON(gin.H{"error": gin.H{"message": failedMessage}})
		if err != nil {
			return payload
		}
		return body
	}

	errorPayload := gin.H{}
	if errType := strings.TrimSpace(gjson.Get(responseError.Raw, "type").String()); errType != "" {
		errorPayload["type"] = errType
	}
	if code := strings.TrimSpace(gjson.Get(responseError.Raw, "code").String()); code != "" {
		errorPayload["code"] = code
	}
	if param := strings.TrimSpace(gjson.Get(responseError.Raw, "param").String()); param != "" {
		errorPayload["param"] = param
	}
	message := strings.TrimSpace(gjson.Get(responseError.Raw, "message").String())
	if message == "" {
		message = strings.TrimSpace(failedMessage)
	}
	if message != "" {
		errorPayload["message"] = message
	}
	if len(errorPayload) == 0 {
		return payload
	}
	body, err := marshalOpenAIUpstreamJSON(gin.H{"error": errorPayload})
	if err != nil {
		return payload
	}
	return body
}

func applyOpenAIStreamFailedErrorPassthroughRule(
	c *gin.Context,
	payload []byte,
	failedMessage string,
) (status int, errType string, errMsg string, matched bool) {
	return applyErrorPassthroughRule(
		c,
		PlatformOpenAI,
		openAIStreamFailedEventSemanticStatus(payload, failedMessage),
		openAIStreamFailedEventPassthroughBody(payload, failedMessage),
		http.StatusBadGateway,
		"upstream_error",
		"Upstream request failed",
	)
}

func (s *OpenAIGatewayService) resolveOpenAICompatFailedError(
	c *gin.Context,
	account *Account,
	requestID string,
	payload []byte,
	message string,
	defaultErrType string,
) (status int, errType string, errMsg string, matched bool, failoverErr error) {
	if openAIStreamFailedEventShouldFailover(payload, message) {
		return 0, "", "", false, s.newOpenAIStreamFailoverError(c, account, false, requestID, payload, message)
	}
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		message = "OpenAI response failed"
	}
	setOpsUpstreamError(c, openAIStreamFailedEventSemanticStatus(payload, message), message, "")
	if account != nil {
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: openAIStreamFailedEventSemanticStatus(payload, message),
			UpstreamRequestID:  strings.TrimSpace(requestID),
			Kind:               "http_error",
			Message:            message,
		})
	}
	status, errType, errMsg = http.StatusBadGateway, defaultErrType, message
	if ruleStatus, ruleType, ruleMessage, ruleMatched := applyOpenAIStreamFailedErrorPassthroughRule(c, payload, message); ruleMatched {
		status, errType, errMsg, matched = ruleStatus, ruleType, ruleMessage, true
	}
	if status == 0 {
		status = http.StatusBadGateway
	}
	if strings.TrimSpace(errType) == "" {
		errType = defaultErrType
	}
	if strings.TrimSpace(errMsg) == "" {
		errMsg = message
	}
	return status, errType, errMsg, matched, nil
}

func sanitizeOpenAIResponseFailedEventForClient(payload []byte, eventType string, clientOutputStarted bool) ([]byte, bool) {
	if eventType != "response.failed" || len(payload) == 0 || !gjson.ValidBytes(payload) {
		return payload, false
	}
	updated := payload
	if clientOutputStarted && isOpenAIContextWindowError(extractOpenAISSEErrorMessage(payload), payload) {
		errorPath := ""
		switch {
		case gjson.GetBytes(updated, "response.error").Exists():
			errorPath = "response.error"
		case gjson.GetBytes(updated, "error").Exists():
			errorPath = "error"
		}
		if errorPath != "" {
			next, err := sjson.SetBytes(updated, errorPath+".type", "invalid_request_error")
			if err != nil {
				return payload, false
			}
			updated = next
			next, err = sjson.SetBytes(updated, errorPath+".code", "context_length_exceeded")
			if err != nil {
				return payload, false
			}
			updated = next
		}
	}
	if !gjson.GetBytes(updated, "response").Exists() {
		return updated, !bytes.Equal(updated, payload)
	}
	for _, path := range []string{
		"response.instructions",
		"response.output",
		"response.usage",
		"response.metadata",
		"response.reasoning",
		"response.tools",
		"response.tool_choice",
		"response.parallel_tool_calls",
		"response.text",
		"response.truncation",
		"response.max_output_tokens",
		"response.incomplete_details",
	} {
		next, err := sjson.DeleteBytes(updated, path)
		if err != nil {
			return payload, false
		}
		updated = next
	}
	return updated, !bytes.Equal(updated, payload)
}
