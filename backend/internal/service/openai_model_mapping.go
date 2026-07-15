package service

import "strings"

// resolveOpenAIForwardModel 解析 OpenAI 兼容转发使用的模型。
// messagesDispatchMappedModel 只服务于 /v1/messages 的 Claude 系列显式调度映射，
// 不作为普通 OpenAI 请求（如 gpt-5.5 等）的未知模型兜底——后者应原样透传。
func resolveOpenAIForwardModel(account *Account, requestedModel, messagesDispatchMappedModel string) string {
	messagesDispatchMappedModel = strings.TrimSpace(messagesDispatchMappedModel)
	if account == nil {
		if messagesDispatchMappedModel != "" && isClaudeFamilyModel(requestedModel) {
			return messagesDispatchMappedModel
		}
		return requestedModel
	}

	mappedModel, matched := account.ResolveMappedModel(requestedModel)
	if !matched && messagesDispatchMappedModel != "" && isClaudeFamilyModel(requestedModel) {
		return messagesDispatchMappedModel
	}
	return mappedModel
}

// isClaudeFamilyModel 判断请求模型是否属于 Claude 家族（含 opus/sonnet/haiku 及
// claude-fable-5 等新型号）。仅这类模型才允许回退到 /v1/messages 的调度映射默认值；
// gpt-5.5 等 OpenAI 原生模型必须原样透传，避免被分组默认值覆盖。
func isClaudeFamilyModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "claude")
}

var openAIOAuthForeignModelPrefixes = []string{
	"deepseek",
	"glm-",
	"kimi-",
	"moonshot",
	"qwen",
	"qwq-",
	"minimax",
	"gemini-",
	"gemma-",
	"grok-",
	"doubao-",
	"hunyuan-",
	"llama",
	"meta-llama",
	"mistral",
	"mixtral",
	"baichuan",
	"ernie-",
	"step-",
	"seed-",
	"yi-",
}

func isOpenAIOAuthServableModel(requestedModel string) bool {
	model := strings.ToLower(lastOpenAIModelSegment(requestedModel))
	if model == "" {
		return true
	}
	for _, prefix := range openAIOAuthForeignModelPrefixes {
		if strings.HasPrefix(model, prefix) {
			return false
		}
	}
	return true
}

// resolveOpenAICompactForwardModel determines the compact-only upstream model
// for /responses/compact requests. It never affects normal /responses traffic.
// When no compact-specific mapping matches, the input model is returned as-is.
func resolveOpenAICompactForwardModel(account *Account, model string) string {
	trimmedModel := strings.TrimSpace(model)
	if trimmedModel == "" || account == nil {
		return trimmedModel
	}

	mappedModel, matched := account.ResolveCompactMappedModel(trimmedModel)
	if !matched {
		return trimmedModel
	}
	if trimmedMapped := strings.TrimSpace(mappedModel); trimmedMapped != "" {
		return trimmedMapped
	}
	return trimmedModel
}
