package service

import (
	"context"
	"strings"
)

// ModelAvailabilityDiagnosis describes configured model support while ignoring
// transient scheduling state such as rate limits and runtime blocks.
type ModelAvailabilityDiagnosis struct {
	HasAccountsInPool bool
	HasModelSupport   bool
}

type ModelAvailabilityDiagnoser interface {
	DiagnoseModelAvailabilityForPlatform(
		ctx context.Context,
		groupID *int64,
		requestedModel string,
		platform string,
	) ModelAvailabilityDiagnosis
}

// DiagnoseModelAvailabilityForPlatform is conservative on invalid input or
// lookup errors so callers keep returning a retryable 503 instead of a false
// model_not_found response.
func (s *GatewayService) DiagnoseModelAvailabilityForPlatform(
	ctx context.Context,
	groupID *int64,
	requestedModel string,
	platform string,
) ModelAvailabilityDiagnosis {
	if s == nil || strings.TrimSpace(requestedModel) == "" || strings.TrimSpace(platform) == "" {
		return ModelAvailabilityDiagnosis{HasAccountsInPool: true, HasModelSupport: true}
	}

	accounts, _, err := s.listSchedulableAccounts(ctx, groupID, platform, false)
	if err != nil {
		return ModelAvailabilityDiagnosis{HasAccountsInPool: true, HasModelSupport: true}
	}

	diagnosis := ModelAvailabilityDiagnosis{}
	for i := range accounts {
		diagnosis.HasAccountsInPool = true
		if s.isModelSupportedByAccountWithContext(ctx, &accounts[i], requestedModel) {
			diagnosis.HasModelSupport = true
			return diagnosis
		}
	}
	return diagnosis
}
