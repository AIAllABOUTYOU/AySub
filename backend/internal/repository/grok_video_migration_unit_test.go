package repository

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestGrokVideoBillingMigrationKeepsSchemaAndConstraintInSync(t *testing.T) {
	content, err := fs.ReadFile(migrations.FS, "158_add_grok_video_billing_metadata.sql")
	require.NoError(t, err)
	sql := string(content)

	for _, column := range []string{
		"video_rate_independent",
		"video_rate_multiplier",
		"video_price_480p",
		"video_price_720p",
		"video_price_1080p",
		"video_count",
		"video_resolution",
		"video_duration_seconds",
	} {
		require.Contains(t, sql, column)
	}
	require.Contains(t, sql, "COALESCE(video_count, 0) > 0")
	require.Contains(t, sql, "billing_mode = 'video'")
	require.Contains(t, sql, "NOT VALID")
	require.Equal(t, 1, strings.Count(sql, "DROP CONSTRAINT IF EXISTS usage_logs_image_billing_size_check"))
}
