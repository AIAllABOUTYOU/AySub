package service

import "context"

type accountCredentialsUpdater interface {
	UpdateCredentials(ctx context.Context, id int64, credentials map[string]any) error
}

func persistAccountCredentials(ctx context.Context, repo AccountRepository, account *Account, credentials map[string]any) error {
	if repo == nil || account == nil {
		return nil
	}
	if account.IsCredentialShadow() {
		return nil
	}

	account.Credentials = cloneCredentials(credentials)
	if updater, ok := any(repo).(accountCredentialsUpdater); ok {
		return updater.UpdateCredentials(ctx, account.ID, account.Credentials)
	}
	return repo.Update(ctx, account)
}

var sparkShadowAllowedCredentialKeys = map[string]struct{}{
	"model_mapping":         {},
	"compact_model_mapping": {},
}

func isAllowedSparkShadowCredentialsUpdate(credentials map[string]any) bool {
	for key := range credentials {
		if _, ok := sparkShadowAllowedCredentialKeys[key]; !ok {
			return false
		}
	}
	return true
}

func sanitizeSparkShadowCredentials(credentials map[string]any) map[string]any {
	out := make(map[string]any, len(sparkShadowAllowedCredentialKeys))
	for key := range sparkShadowAllowedCredentialKeys {
		if value, ok := credentials[key]; ok && value != nil {
			out[key] = value
		}
	}
	return out
}

func cloneCredentials(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
