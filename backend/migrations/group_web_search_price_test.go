package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupWebSearchPriceMigration(t *testing.T) {
	content, err := FS.ReadFile("176_group_web_search_price_per_call.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(content))
	require.Contains(t, sql, "add column if not exists web_search_price_per_call")
	require.Contains(t, sql, "decimal(20,8)")
}
