package service

import (
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	APIKeyPermissionModeInherit  = "inherit"
	APIKeyPermissionModeRestrict = "restrict"

	EndpointPermissionMessages            = "messages"
	EndpointPermissionChatCompletions     = "chat_completions"
	EndpointPermissionResponses           = "responses"
	EndpointPermissionEmbeddings          = "embeddings"
	EndpointPermissionImages              = "images"
	EndpointPermissionVideos              = "videos"
	EndpointPermissionAudioSpeech         = "audio_speech"
	EndpointPermissionAudioTranscriptions = "audio_transcriptions"
	EndpointPermissionAudioTranslations   = "audio_translations"
	EndpointPermissionLiveKit             = "livekit"
	EndpointPermissionGeminiNative        = "gemini_native"
)

var (
	ErrInvalidAPIKeyPermissionMode     = infraerrors.BadRequest("INVALID_API_KEY_PERMISSION_MODE", "invalid api key permission mode")
	ErrInvalidAPIKeyPermissionEndpoint = infraerrors.BadRequest("INVALID_API_KEY_PERMISSION_ENDPOINT", "invalid api key permission endpoint")
)

var validAPIKeyPermissionEndpoints = map[string]struct{}{
	EndpointPermissionMessages:            {},
	EndpointPermissionChatCompletions:     {},
	EndpointPermissionResponses:           {},
	EndpointPermissionEmbeddings:          {},
	EndpointPermissionImages:              {},
	EndpointPermissionVideos:              {},
	EndpointPermissionAudioSpeech:         {},
	EndpointPermissionAudioTranscriptions: {},
	EndpointPermissionAudioTranslations:   {},
	EndpointPermissionLiveKit:             {},
	EndpointPermissionGeminiNative:        {},
}

// NormalizeAPIKeyPermissionMode returns the canonical mode. Empty mode inherits
// existing group/channel capabilities for backward compatibility.
func NormalizeAPIKeyPermissionMode(mode string) (string, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return APIKeyPermissionModeInherit, nil
	}
	switch mode {
	case APIKeyPermissionModeInherit, APIKeyPermissionModeRestrict:
		return mode, nil
	default:
		return "", ErrInvalidAPIKeyPermissionMode
	}
}

func NormalizeAPIKeyAllowedModels(models []string) []string {
	return normalizeStringSet(models, false)
}

func NormalizeAPIKeyAllowedEndpoints(endpoints []string) ([]string, error) {
	out := normalizeStringSet(endpoints, true)
	for _, endpoint := range out {
		if _, ok := validAPIKeyPermissionEndpoints[endpoint]; !ok {
			return nil, ErrInvalidAPIKeyPermissionEndpoint
		}
	}
	return out, nil
}

func NormalizeAPIKeyAllowedEndpointsNoError(endpoints []string) []string {
	out, err := NormalizeAPIKeyAllowedEndpoints(endpoints)
	if err != nil {
		return []string{}
	}
	return out
}

func APIKeyPermissionEndpoints() []string {
	out := make([]string, 0, len(validAPIKeyPermissionEndpoints))
	for endpoint := range validAPIKeyPermissionEndpoints {
		out = append(out, endpoint)
	}
	return out
}

func normalizeStringSet(values []string, lower bool) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if lower {
			value = strings.ToLower(value)
		}
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func APIKeyPermissionsChanged(k *APIKey, mode string, models, endpoints []string) bool {
	if k == nil {
		return false
	}
	if NormalizeAPIKeyPermissionModeNoError(k.PermissionMode) != mode {
		return true
	}
	if !stringSlicesEqual(k.AllowedModels, models) {
		return true
	}
	return !stringSlicesEqual(k.AllowedEndpoints, endpoints)
}

func NormalizeAPIKeyPermissionModeNoError(mode string) string {
	normalized, err := NormalizeAPIKeyPermissionMode(mode)
	if err != nil {
		return APIKeyPermissionModeInherit
	}
	return normalized
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (k *APIKey) NormalizePermissions() error {
	if k == nil {
		return nil
	}
	mode, err := NormalizeAPIKeyPermissionMode(k.PermissionMode)
	if err != nil {
		return err
	}
	endpoints, err := NormalizeAPIKeyAllowedEndpoints(k.AllowedEndpoints)
	if err != nil {
		return err
	}
	k.PermissionMode = mode
	k.AllowedModels = NormalizeAPIKeyAllowedModels(k.AllowedModels)
	k.AllowedEndpoints = endpoints
	return nil
}

func APIKeyPermissionUpdatedAtNow() *time.Time {
	now := time.Now()
	return &now
}

func (k *APIKey) AllowsEndpoint(endpoint string) bool {
	if k == nil || NormalizeAPIKeyPermissionModeNoError(k.PermissionMode) != APIKeyPermissionModeRestrict {
		return true
	}
	endpoint = strings.ToLower(strings.TrimSpace(endpoint))
	if endpoint == "" || len(k.AllowedEndpoints) == 0 {
		return true
	}
	for _, allowed := range k.AllowedEndpoints {
		if allowed == endpoint {
			return true
		}
	}
	return false
}

func (k *APIKey) AllowsModel(model string) bool {
	if k == nil || NormalizeAPIKeyPermissionModeNoError(k.PermissionMode) != APIKeyPermissionModeRestrict {
		return true
	}
	model = strings.TrimSpace(model)
	if model == "" || len(k.AllowedModels) == 0 {
		return true
	}
	for _, allowed := range k.AllowedModels {
		if allowed == model {
			return true
		}
		if strings.HasSuffix(allowed, "*") && strings.HasPrefix(model, strings.TrimSuffix(allowed, "*")) {
			return true
		}
	}
	return false
}
