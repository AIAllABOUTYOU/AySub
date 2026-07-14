//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newSchedulerCacheUnitTest(t *testing.T) (context.Context, *schedulerCache, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return context.Background(), newSchedulerCacheWithChunkSizes(rdb, 8, 8).(*schedulerCache), mr, rdb
}

func TestSchedulerCacheSkipsAndClearsUnencodableAccounts(t *testing.T) {
	ctx, cache, _, _ := newSchedulerCacheUnitTest(t)
	invalidTime := time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)

	cacheable, err := cache.writeAccounts(ctx, []service.Account{
		{ID: 111, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey},
		{ID: 112, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, ExpiresAt: &invalidTime},
	})
	require.NoError(t, err)
	require.Len(t, cacheable, 1)

	account := service.Account{ID: 113, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	require.NoError(t, cache.SetAccount(ctx, &account))
	account.ExpiresAt = &invalidTime
	require.NoError(t, cache.SetAccount(ctx, &account))
	cached, err := cache.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, cached)
}

func TestSchedulerCacheListBucketsReconcilesActiveIndex(t *testing.T) {
	ctx, cache, _, rdb := newSchedulerCacheUnitTest(t)
	active := service.SchedulerBucket{GroupID: 1, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	stale := service.SchedulerBucket{GroupID: 2, Platform: service.PlatformGemini, Mode: service.SchedulerModeMixed}

	require.NoError(t, cache.SetSnapshot(ctx, active, []service.Account{{ID: 11}}))
	require.NoError(t, rdb.SAdd(ctx, schedulerBucketSetKey, stale.String(), "invalid-bucket").Err())

	buckets, err := cache.ListBuckets(ctx)
	require.NoError(t, err)
	require.Equal(t, []service.SchedulerBucket{active}, buckets)
	require.False(t, rdb.SIsMember(ctx, schedulerBucketSetKey, stale.String()).Val())
	require.False(t, rdb.SIsMember(ctx, schedulerBucketSetKey, "invalid-bucket").Val())
}

func TestSchedulerCacheExpiredOwnerCannotReleaseNewLock(t *testing.T) {
	ctx, cache, mr, rdb := newSchedulerCacheUnitTest(t)
	bucket := service.SchedulerBucket{GroupID: 3, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}

	oldOwner, acquired, err := cache.TryLockBucket(ctx, bucket, time.Second)
	require.NoError(t, err)
	require.True(t, acquired)
	mr.FastForward(2 * time.Second)

	newOwner, acquired, err := cache.TryLockBucket(ctx, bucket, time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotEqual(t, oldOwner, newOwner)
	require.NoError(t, cache.UnlockBucket(ctx, bucket, oldOwner))

	value, err := rdb.Get(ctx, schedulerBucketKey(schedulerLockPrefix, bucket)).Result()
	require.NoError(t, err)
	require.Equal(t, newOwner, value)
}

func TestSchedulerCacheReconcileExpiredBucketLocks(t *testing.T) {
	ctx, cache, mr, rdb := newSchedulerCacheUnitTest(t)
	expired := service.SchedulerBucket{GroupID: 4, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeMixed}
	anomalous := service.SchedulerBucket{GroupID: 5, Platform: service.PlatformGemini, Mode: service.SchedulerModeSingle}

	_, acquired, err := cache.TryLockBucket(ctx, expired, time.Second)
	require.NoError(t, err)
	require.True(t, acquired)
	mr.FastForward(2 * time.Second)

	now, err := rdb.Time(ctx).Result()
	require.NoError(t, err)
	require.NoError(t, rdb.ZAdd(ctx, schedulerLockIndexKey, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: expired.String(),
	}).Err())
	require.NoError(t, rdb.Set(ctx, schedulerBucketKey(schedulerLockPrefix, anomalous), "orphan-owner", 0).Err())
	_, acquired, err = cache.TryLockBucket(ctx, anomalous, time.Minute)
	require.NoError(t, err)
	require.False(t, acquired)
	require.NoError(t, rdb.ZScore(ctx, schedulerLockIndexKey, anomalous.String()).Err())

	cleaned, err := cache.reconcileExpiredBucketLocks(ctx, schedulerLockCleanupBatchSize)
	require.NoError(t, err)
	require.Equal(t, 1, cleaned)
	require.EqualValues(t, 0, rdb.Exists(ctx, schedulerBucketKey(schedulerLockPrefix, anomalous)).Val())
	require.ErrorIs(t, rdb.ZScore(ctx, schedulerLockIndexKey, expired.String()).Err(), redis.Nil)
	require.ErrorIs(t, rdb.ZScore(ctx, schedulerLockIndexKey, anomalous.String()).Err(), redis.Nil)
}

func TestBuildSchedulerMetadataAccount_KeepsOpenAIWSFlags(t *testing.T) {
	account := service.Account{
		ID:       42,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Extra: map[string]any{
			"openai_oauth_responses_websockets_v2_enabled": true,
			"openai_oauth_responses_websockets_v2_mode":    service.OpenAIWSIngressModePassthrough,
			"openai_ws_force_http":                         true,
			"openai_responses_mode":                        "force_chat_completions",
			"openai_responses_supported":                   false,
			"mixed_scheduling":                             true,
			"unused_large_field":                           "drop-me",
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.Equal(t, true, got.Extra["openai_oauth_responses_websockets_v2_enabled"])
	require.Equal(t, service.OpenAIWSIngressModePassthrough, got.Extra["openai_oauth_responses_websockets_v2_mode"])
	require.Equal(t, true, got.Extra["openai_ws_force_http"])
	require.Equal(t, "force_chat_completions", got.Extra["openai_responses_mode"])
	require.Equal(t, false, got.Extra["openai_responses_supported"])
	require.Equal(t, true, got.Extra["mixed_scheduling"])
	require.Nil(t, got.Extra["unused_large_field"])
}

func TestBuildSchedulerMetadataAccount_KeepsSlimGroupMembership(t *testing.T) {
	account := service.Account{
		ID:       42,
		Platform: service.PlatformAnthropic,
		GroupIDs: []int64{7, 9, 7, 0},
		AccountGroups: []service.AccountGroup{
			{
				AccountID: 42,
				GroupID:   7,
				Priority:  2,
				Account:   &service.Account{ID: 42, Name: "drop-from-metadata"},
				Group:     &service.Group{ID: 7, Name: "drop-from-metadata"},
			},
			{
				AccountID: 42,
				GroupID:   11,
				Priority:  3,
				Group:     &service.Group{ID: 11, Name: "drop-from-metadata"},
			},
			{
				AccountID: 42,
				GroupID:   0,
				Priority:  4,
			},
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.Equal(t, []int64{7, 9, 11}, got.GroupIDs)
	require.Len(t, got.AccountGroups, 2)
	require.Equal(t, int64(42), got.AccountGroups[0].AccountID)
	require.Equal(t, int64(7), got.AccountGroups[0].GroupID)
	require.Equal(t, 2, got.AccountGroups[0].Priority)
	require.Nil(t, got.AccountGroups[0].Account)
	require.Nil(t, got.AccountGroups[0].Group)
	require.Equal(t, int64(11), got.AccountGroups[1].GroupID)
	require.Nil(t, got.Groups)
}

func TestBuildSchedulerMetadataAccount_KeepsQuotaAutoPauseFields(t *testing.T) {
	account := service.Account{
		ID: 88,
		Extra: map[string]any{
			"codex_5h_used_percent":        12.34,
			"codex_7d_used_percent":        56.78,
			"codex_5h_reset_at":            "2026-05-29T10:00:00Z",
			"codex_7d_reset_at":            "2026-06-01T10:00:00Z",
			"codex_5h_reset_after_seconds": 300,
			"codex_7d_reset_after_seconds": 600,
			"codex_usage_updated_at":       "2026-05-29T09:00:00Z",
			"auto_pause_5h_threshold":      0.95,
			"auto_pause_7d_threshold":      0.96,
			"auto_pause_5h_disabled":       true,
			"auto_pause_7d_disabled":       false,
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.Equal(t, 12.34, got.Extra["codex_5h_used_percent"])
	require.Equal(t, 56.78, got.Extra["codex_7d_used_percent"])
	require.Equal(t, "2026-05-29T10:00:00Z", got.Extra["codex_5h_reset_at"])
	require.Equal(t, "2026-06-01T10:00:00Z", got.Extra["codex_7d_reset_at"])
	require.Equal(t, 300, got.Extra["codex_5h_reset_after_seconds"])
	require.Equal(t, 600, got.Extra["codex_7d_reset_after_seconds"])
	require.Equal(t, "2026-05-29T09:00:00Z", got.Extra["codex_usage_updated_at"])
	require.Equal(t, 0.95, got.Extra["auto_pause_5h_threshold"])
	require.Equal(t, 0.96, got.Extra["auto_pause_7d_threshold"])
	require.Equal(t, true, got.Extra["auto_pause_5h_disabled"])
	require.Equal(t, false, got.Extra["auto_pause_7d_disabled"])
}

func TestBuildSchedulerMetadataAccount_KeepsModelRateLimits(t *testing.T) {
	account := service.Account{
		ID:       90,
		Platform: service.PlatformAntigravity,
		Extra: map[string]any{
			"model_rate_limits": map[string]any{
				"gemini-3-flash": map[string]any{
					"rate_limit_reset_at": "2026-05-30T10:10:00Z",
				},
				"antigravity:gemini": map[string]any{
					"rate_limit_reset_at": "2026-05-30T10:10:00Z",
				},
			},
			"unused_large_field": "drop-me",
		},
	}

	got := buildSchedulerMetadataAccount(account)

	limits, ok := got.Extra["model_rate_limits"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, limits, "gemini-3-flash")
	require.Contains(t, limits, "antigravity:gemini")
	require.Nil(t, got.Extra["unused_large_field"])
}

func TestBuildSchedulerMetadataAccount_KeepsSparkShadowRoutingIdentity(t *testing.T) {
	parentID := int64(100)
	account := service.Account{
		ID:              200,
		Platform:        service.PlatformOpenAI,
		Type:            service.AccountTypeOAuth,
		ParentAccountID: &parentID,
		QuotaDimension:  service.QuotaDimensionSpark,
		Credentials: map[string]any{
			"model_mapping":         map[string]any{"gpt-5.3-codex-spark": "gpt-5.3-codex-spark"},
			"compact_model_mapping": map[string]any{"gpt-5.4": "gpt-5.4-openai-compact"},
			"access_token":          "drop-me",
		},
	}
	got := buildSchedulerMetadataAccount(account)
	require.Equal(t, parentID, *got.ParentAccountID)
	require.Equal(t, service.QuotaDimensionSpark, got.QuotaDimension)
	require.NotNil(t, got.Credentials["model_mapping"])
	require.NotNil(t, got.Credentials["compact_model_mapping"])
	require.Nil(t, got.Credentials["access_token"])
}
