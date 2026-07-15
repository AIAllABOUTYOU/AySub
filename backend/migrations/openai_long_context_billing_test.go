package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAILongContextBillingMigrations(t *testing.T) {
	usageLogSQL, err := FS.ReadFile("177_add_usage_log_long_context_billing.sql")
	require.NoError(t, err)
	require.Contains(t, string(usageLogSQL), "long_context_billing_applied BOOLEAN NOT NULL DEFAULT FALSE")

	accountSQL, err := FS.ReadFile("178_default_openai_long_context_billing.sql")
	require.NoError(t, err)
	sql := string(accountSQL)
	require.Contains(t, sql, "NEW.parent_account_id IS NOT NULL AND NEW.quota_dimension = 'spark'")
	require.Contains(t, sql, "OLD.extra->'openai_long_context_billing_enabled'")
	require.Contains(t, sql, "RAISE EXCEPTION 'openai_long_context_billing_enabled must be a boolean'")
	require.Contains(t, sql, "CREATE TRIGGER accounts_propagate_openai_long_context_billing_extra")
	require.Contains(t, sql, "NEW.extra->'openai_long_context_billing_enabled'")
}
