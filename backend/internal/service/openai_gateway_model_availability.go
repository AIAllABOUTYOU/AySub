package service

import (
	"context"
	"strings"
)

// DiagnoseModelAvailabilityForPlatform scopes diagnosis to the requested
// OpenAI-compatible platform so OpenAI and xAI pools cannot contaminate each
// other's model-not-found result.
func (s *OpenAIGatewayService) DiagnoseModelAvailabilityForPlatform(
	ctx context.Context,
	groupID *int64,
	requestedModel string,
	platform string,
) ModelAvailabilityDiagnosis {
	if s == nil || strings.TrimSpace(requestedModel) == "" || !IsOpenAICompatiblePlatform(platform) {
		return ModelAvailabilityDiagnosis{HasAccountsInPool: true, HasModelSupport: true}
	}

	ctx = WithOpenAICompatiblePlatform(ctx, platform)
	accounts, err := s.listSchedulableAccounts(ctx, groupID)
	if err != nil {
		return ModelAvailabilityDiagnosis{HasAccountsInPool: true, HasModelSupport: true}
	}

	diagnosis := ModelAvailabilityDiagnosis{}
	for i := range accounts {
		diagnosis.HasAccountsInPool = true
		if accounts[i].IsModelSupported(requestedModel) {
			diagnosis.HasModelSupport = true
			return diagnosis
		}
	}
	return diagnosis
}
