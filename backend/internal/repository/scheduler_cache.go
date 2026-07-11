package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	schedulerBucketSetKey       = "sched:buckets"
	schedulerOutboxWatermarkKey = "sched:outbox:watermark"
	schedulerAccountPrefix      = "sched:acc:"
	schedulerAccountMetaPrefix  = "sched:meta:"
	schedulerActivePrefix       = "sched:active:"
	schedulerReadyPrefix        = "sched:ready:"
	schedulerVersionPrefix      = "sched:ver:"
	schedulerSnapshotPrefix     = "sched:"
	schedulerLockPrefix         = "sched:lock:"
	schedulerLockIndexKey       = "sched:lock:index"

	defaultSchedulerSnapshotMGetChunkSize  = 128
	defaultSchedulerSnapshotWriteChunkSize = 256
	schedulerLockCleanupBatchSize          = 1000

	// snapshotGraceTTLSeconds 旧快照过期的宽限期（秒）。
	// 替代立即 DEL，让正在读取旧版本的 reader 有足够时间完成 ZRANGE。
	snapshotGraceTTLSeconds = 60
)

var (
	// activateSnapshotScript 原子 CAS 切换快照版本。
	// 仅当新版本号 >= 当前激活版本时才切换，防止并发写入导致版本回滚。
	// 旧快照使用 EXPIRE 设置宽限期而非立即 DEL，避免与 reader 竞态。
	//
	// KEYS[1] = activeKey     (sched:active:{bucket})
	// KEYS[2] = readyKey      (sched:ready:{bucket})
	// KEYS[3] = bucketSetKey  (sched:buckets)
	// KEYS[4] = snapshotKey   (新写入的快照 key)
	// ARGV[1] = 新版本号字符串
	// ARGV[2] = bucket 字符串 (用于 SADD)
	// ARGV[3] = 快照 key 前缀 (用于构造旧快照 key)
	// ARGV[4] = 宽限期 TTL 秒数
	//
	// 返回 1 = 已激活, 0 = 版本过旧未激活
	activateSnapshotScript = redis.NewScript(`
local currentActive = redis.call('GET', KEYS[1])
local newVersion = tonumber(ARGV[1])

if currentActive ~= false then
	local curVersion = tonumber(currentActive)
	if curVersion and newVersion < curVersion then
		redis.call('DEL', KEYS[4])
		return 0
	end
end

redis.call('SET', KEYS[1], ARGV[1])
redis.call('SET', KEYS[2], '1')
redis.call('SADD', KEYS[3], ARGV[2])

if currentActive ~= false and currentActive ~= ARGV[1] then
	redis.call('EXPIRE', ARGV[3] .. currentActive, tonumber(ARGV[4]))
end

return 1
`)

	// acquireSchedulerLockScript 使用 Redis 时间获取带 TTL 的 owner lock。
	// 返回 {是否获取成功, 观测到的锁到期 Unix 毫秒}；争锁失败也返回当前持有者的
	// 到期时间，使丢失的索引项能在下一次争用时自动回填。
	acquireSchedulerLockScript = redis.NewScript(`
redis.replicate_commands()
local current = redis.call('GET', KEYS[1])
local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
if current ~= false then
	local pttl = redis.call('PTTL', KEYS[1])
	if pttl and pttl > 0 then
		return {0, now + pttl}
	end
	return {0, now}
end
redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
return {1, now + tonumber(ARGV[2])}
`)

	// releaseSchedulerLockScript 只释放 owner 匹配的锁，避免旧 rebuild 在锁过期后
	// 删除另一个 worker 已重新获取的新锁。
	releaseSchedulerLockScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('DEL', KEYS[1])
end
return 0
`)

	// reconcileSchedulerLockScript 校验到期索引候选的真实锁状态。
	// 返回 {-2,0}=锁不存在，{-1,0}=无 TTL 异常锁已删除，{1,pttl}=锁仍存活。
	reconcileSchedulerLockScript = redis.NewScript(`
local pttl = redis.call('PTTL', KEYS[1])
if pttl == -2 then
	return {-2, 0}
end
if pttl == -1 then
	redis.call('DEL', KEYS[1])
	return {-1, 0}
end
return {1, pttl}
`)
)

type schedulerCache struct {
	rdb            *redis.Client
	mgetChunkSize  int
	writeChunkSize int
}

func NewSchedulerCache(rdb *redis.Client) service.SchedulerCache {
	return newSchedulerCacheWithChunkSizes(rdb, defaultSchedulerSnapshotMGetChunkSize, defaultSchedulerSnapshotWriteChunkSize)
}

func newSchedulerCacheWithChunkSizes(rdb *redis.Client, mgetChunkSize, writeChunkSize int) service.SchedulerCache {
	if mgetChunkSize <= 0 {
		mgetChunkSize = defaultSchedulerSnapshotMGetChunkSize
	}
	if writeChunkSize <= 0 {
		writeChunkSize = defaultSchedulerSnapshotWriteChunkSize
	}
	return &schedulerCache{
		rdb:            rdb,
		mgetChunkSize:  mgetChunkSize,
		writeChunkSize: writeChunkSize,
	}
}

func (c *schedulerCache) GetSnapshot(ctx context.Context, bucket service.SchedulerBucket) ([]*service.Account, bool, error) {
	readyKey := schedulerBucketKey(schedulerReadyPrefix, bucket)
	readyVal, err := c.rdb.Get(ctx, readyKey).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if readyVal != "1" {
		return nil, false, nil
	}

	activeKey := schedulerBucketKey(schedulerActivePrefix, bucket)
	activeVal, err := c.rdb.Get(ctx, activeKey).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	snapshotKey := schedulerSnapshotKey(bucket, activeVal)
	ids, err := c.rdb.ZRange(ctx, snapshotKey, 0, -1).Result()
	if err != nil {
		return nil, false, err
	}
	if len(ids) == 0 {
		// 空快照视为缓存未命中，触发数据库回退查询
		// 这解决了新分组创建后立即绑定账号时的竞态条件问题
		return nil, false, nil
	}

	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, schedulerAccountMetaKey(id))
	}
	values, err := c.mgetChunked(ctx, keys)
	if err != nil {
		return nil, false, err
	}

	accounts := make([]*service.Account, 0, len(values))
	for _, val := range values {
		if val == nil {
			return nil, false, nil
		}
		account, err := decodeCachedAccount(val)
		if err != nil {
			return nil, false, err
		}
		accounts = append(accounts, account)
	}

	return accounts, true, nil
}

func (c *schedulerCache) SetSnapshot(ctx context.Context, bucket service.SchedulerBucket, accounts []service.Account) error {
	// Phase 1: 分配新版本号并写入快照数据。
	// INCR 保证每个调用方获得唯一递增版本号。
	// 写入的 snapshotKey 是新的版本化 key，reader 尚不知晓，因此无竞态。
	versionKey := schedulerBucketKey(schedulerVersionPrefix, bucket)
	version, err := c.rdb.Incr(ctx, versionKey).Result()
	if err != nil {
		return err
	}

	versionStr := strconv.FormatInt(version, 10)
	snapshotKey := schedulerSnapshotKey(bucket, versionStr)

	if err := c.writeAccounts(ctx, accounts); err != nil {
		return err
	}

	if len(accounts) > 0 {
		// 使用序号作为 score，保持数据库返回的排序语义。
		members := make([]redis.Z, 0, len(accounts))
		for idx, account := range accounts {
			members = append(members, redis.Z{
				Score:  float64(idx),
				Member: strconv.FormatInt(account.ID, 10),
			})
		}
		pipe := c.rdb.Pipeline()
		for start := 0; start < len(members); start += c.writeChunkSize {
			end := start + c.writeChunkSize
			if end > len(members) {
				end = len(members)
			}
			pipe.ZAdd(ctx, snapshotKey, members[start:end]...)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
	}

	// Phase 2: 原子 CAS 激活版本。
	// Lua 脚本保证：仅当新版本 >= 当前激活版本时才切换 active 指针，
	// 防止并发写入导致版本回滚。
	// 旧快照使用 EXPIRE 宽限期而非立即 DEL，避免 reader 竞态。
	activeKey := schedulerBucketKey(schedulerActivePrefix, bucket)
	readyKey := schedulerBucketKey(schedulerReadyPrefix, bucket)
	snapshotKeyPrefix := fmt.Sprintf("%s%d:%s:%s:v", schedulerSnapshotPrefix, bucket.GroupID, bucket.Platform, bucket.Mode)

	keys := []string{activeKey, readyKey, schedulerBucketSetKey, snapshotKey}
	args := []any{versionStr, bucket.String(), snapshotKeyPrefix, snapshotGraceTTLSeconds}

	_, err = activateSnapshotScript.Run(ctx, c.rdb, keys, args...).Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *schedulerCache) GetAccount(ctx context.Context, accountID int64) (*service.Account, error) {
	key := schedulerAccountKey(strconv.FormatInt(accountID, 10))
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeCachedAccount(val)
}

func (c *schedulerCache) SetAccount(ctx context.Context, account *service.Account) error {
	if account == nil || account.ID <= 0 {
		return nil
	}
	return c.writeAccounts(ctx, []service.Account{*account})
}

func (c *schedulerCache) DeleteAccount(ctx context.Context, accountID int64) error {
	if accountID <= 0 {
		return nil
	}
	id := strconv.FormatInt(accountID, 10)
	return c.rdb.Del(ctx, schedulerAccountKey(id), schedulerAccountMetaKey(id)).Err()
}

func (c *schedulerCache) UpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	if len(updates) == 0 {
		return nil
	}

	keys := make([]string, 0, len(updates))
	ids := make([]int64, 0, len(updates))
	for id := range updates {
		keys = append(keys, schedulerAccountKey(strconv.FormatInt(id, 10)))
		ids = append(ids, id)
	}

	values, err := c.mgetChunked(ctx, keys)
	if err != nil {
		return err
	}

	pipe := c.rdb.Pipeline()
	for i, val := range values {
		if val == nil {
			continue
		}
		account, err := decodeCachedAccount(val)
		if err != nil {
			return err
		}
		account.LastUsedAt = ptrTime(updates[ids[i]])
		updated, err := json.Marshal(account)
		if err != nil {
			return err
		}
		metaPayload, err := json.Marshal(buildSchedulerMetadataAccount(*account))
		if err != nil {
			return err
		}
		pipe.Set(ctx, keys[i], updated, 0)
		pipe.Set(ctx, schedulerAccountMetaKey(strconv.FormatInt(ids[i], 10)), metaPayload, 0)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (c *schedulerCache) TryLockBucket(ctx context.Context, bucket service.SchedulerBucket, ttl time.Duration) (string, bool, error) {
	if ttl <= 0 {
		return "", false, nil
	}
	key := schedulerBucketKey(schedulerLockPrefix, bucket)
	owner := uuid.NewString()
	result, err := acquireSchedulerLockScript.Run(ctx, c.rdb, []string{key}, owner, ttl.Milliseconds()).Result()
	if err != nil {
		return "", false, err
	}
	acquired, err := schedulerScriptInt64At(result, 0)
	if err != nil {
		return "", false, fmt.Errorf("parse scheduler lock result: %w", err)
	}
	expireAtMs, err := schedulerScriptInt64At(result, 1)
	if err != nil {
		return "", false, fmt.Errorf("parse scheduler lock expiry: %w", err)
	}
	if expireAtMs > 0 {
		if err := c.rdb.ZAdd(ctx, schedulerLockIndexKey, redis.Z{
			Score:  float64(expireAtMs),
			Member: bucket.String(),
		}).Err(); err != nil {
			logger.LegacyPrintf("repository.scheduler_cache", "Warning: update lock index for bucket %s failed: %v", bucket.String(), err)
		}
	}
	if acquired != 1 {
		return "", false, nil
	}
	return owner, true, nil
}

func (c *schedulerCache) UnlockBucket(ctx context.Context, bucket service.SchedulerBucket, owner string) error {
	if owner == "" {
		return nil
	}
	key := schedulerBucketKey(schedulerLockPrefix, bucket)
	released, err := releaseSchedulerLockScript.Run(ctx, c.rdb, []string{key}, owner).Int()
	if err != nil {
		return err
	}
	if released == 1 {
		if err := c.rdb.ZRem(ctx, schedulerLockIndexKey, bucket.String()).Err(); err != nil {
			logger.LegacyPrintf("repository.scheduler_cache", "Warning: remove lock index for bucket %s failed: %v", bucket.String(), err)
		}
	}
	return nil
}

func (c *schedulerCache) ListBuckets(ctx context.Context) ([]service.SchedulerBucket, error) {
	if _, err := c.reconcileExpiredBucketLocks(ctx, schedulerLockCleanupBatchSize); err != nil {
		return nil, err
	}
	raw, err := c.rdb.SMembers(ctx, schedulerBucketSetKey).Result()
	if err != nil {
		return nil, err
	}
	out := make([]service.SchedulerBucket, 0, len(raw))
	stale := make([]any, 0)
	type bucketState struct {
		bucket service.SchedulerBucket
		member string
		ready  *redis.StringCmd
		active *redis.StringCmd
	}
	states := make([]bucketState, 0, len(raw))
	pipe := c.rdb.Pipeline()
	for _, entry := range raw {
		bucket, ok := service.ParseSchedulerBucket(entry)
		if !ok {
			stale = append(stale, entry)
			continue
		}
		states = append(states, bucketState{
			bucket: bucket,
			member: entry,
			ready:  pipe.Get(ctx, schedulerBucketKey(schedulerReadyPrefix, bucket)),
			active: pipe.Get(ctx, schedulerBucketKey(schedulerActivePrefix, bucket)),
		})
	}
	if len(states) > 0 {
		if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
			return nil, err
		}
	}
	for _, state := range states {
		ready, readyErr := state.ready.Result()
		active, activeErr := state.active.Result()
		if readyErr != nil && readyErr != redis.Nil {
			return nil, readyErr
		}
		if activeErr != nil && activeErr != redis.Nil {
			return nil, activeErr
		}
		if ready != "1" || active == "" {
			stale = append(stale, state.member)
			continue
		}
		out = append(out, state.bucket)
	}
	if len(stale) > 0 {
		if err := c.rdb.SRem(ctx, schedulerBucketSetKey, stale...).Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (c *schedulerCache) reconcileExpiredBucketLocks(ctx context.Context, maxCount int64) (int, error) {
	if maxCount <= 0 {
		return 0, nil
	}
	now, err := c.rdb.Time(ctx).Result()
	if err != nil {
		return 0, fmt.Errorf("get scheduler redis time: %w", err)
	}
	nowMs := now.UnixMilli()
	members, err := c.rdb.ZRangeByScore(ctx, schedulerLockIndexKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatInt(nowMs, 10),
		Count: maxCount,
	}).Result()
	if err != nil {
		return 0, fmt.Errorf("read scheduler lock index: %w", err)
	}

	cleaned := 0
	for _, member := range members {
		bucket, ok := service.ParseSchedulerBucket(member)
		if !ok {
			c.removeSchedulerLockIndexMember(ctx, member)
			continue
		}
		result, err := reconcileSchedulerLockScript.Run(ctx, c.rdb, []string{schedulerBucketKey(schedulerLockPrefix, bucket)}).Result()
		if err != nil {
			return cleaned, fmt.Errorf("reconcile scheduler lock %s: %w", member, err)
		}
		status, err := schedulerScriptInt64At(result, 0)
		if err != nil {
			return cleaned, fmt.Errorf("parse scheduler reconcile status: %w", err)
		}
		pttl, err := schedulerScriptInt64At(result, 1)
		if err != nil {
			return cleaned, fmt.Errorf("parse scheduler reconcile pttl: %w", err)
		}
		switch status {
		case -2:
			c.removeSchedulerLockIndexMember(ctx, member)
		case -1:
			c.removeSchedulerLockIndexMember(ctx, member)
			cleaned++
		case 1:
			if err := c.rdb.ZAdd(ctx, schedulerLockIndexKey, redis.Z{
				Score:  float64(nowMs + pttl),
				Member: member,
			}).Err(); err != nil {
				logger.LegacyPrintf("repository.scheduler_cache", "Warning: reschedule lock index member %s failed: %v", member, err)
			}
		}
	}
	return cleaned, nil
}

func (c *schedulerCache) removeSchedulerLockIndexMember(ctx context.Context, member string) {
	if err := c.rdb.ZRem(ctx, schedulerLockIndexKey, member).Err(); err != nil {
		logger.LegacyPrintf("repository.scheduler_cache", "Warning: remove lock index member %s failed: %v", member, err)
	}
}

func schedulerScriptInt64At(result any, index int) (int64, error) {
	values, ok := result.([]any)
	if !ok {
		return 0, fmt.Errorf("expected redis script array, got %T", result)
	}
	if index < 0 || index >= len(values) {
		return 0, fmt.Errorf("redis script array missing index %d", index)
	}
	switch value := values[index].(type) {
	case int64:
		return value, nil
	case int:
		return int64(value), nil
	case string:
		return strconv.ParseInt(value, 10, 64)
	case []byte:
		return strconv.ParseInt(string(value), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected redis script value type %T", values[index])
	}
}

func (c *schedulerCache) GetOutboxWatermark(ctx context.Context) (int64, error) {
	val, err := c.rdb.Get(ctx, schedulerOutboxWatermarkKey).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (c *schedulerCache) SetOutboxWatermark(ctx context.Context, id int64) error {
	return c.rdb.Set(ctx, schedulerOutboxWatermarkKey, strconv.FormatInt(id, 10), 0).Err()
}

func schedulerBucketKey(prefix string, bucket service.SchedulerBucket) string {
	return fmt.Sprintf("%s%d:%s:%s", prefix, bucket.GroupID, bucket.Platform, bucket.Mode)
}

func schedulerSnapshotKey(bucket service.SchedulerBucket, version string) string {
	return fmt.Sprintf("%s%d:%s:%s:v%s", schedulerSnapshotPrefix, bucket.GroupID, bucket.Platform, bucket.Mode, version)
}

func schedulerAccountKey(id string) string {
	return schedulerAccountPrefix + id
}

func schedulerAccountMetaKey(id string) string {
	return schedulerAccountMetaPrefix + id
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func decodeCachedAccount(val any) (*service.Account, error) {
	var payload []byte
	switch raw := val.(type) {
	case string:
		payload = []byte(raw)
	case []byte:
		payload = raw
	default:
		return nil, fmt.Errorf("unexpected account cache type: %T", val)
	}
	var account service.Account
	if err := json.Unmarshal(payload, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

func (c *schedulerCache) writeAccounts(ctx context.Context, accounts []service.Account) error {
	if len(accounts) == 0 {
		return nil
	}

	pipe := c.rdb.Pipeline()
	pending := 0
	flush := func() error {
		if pending == 0 {
			return nil
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
		pipe = c.rdb.Pipeline()
		pending = 0
		return nil
	}

	for _, account := range accounts {
		fullPayload, err := json.Marshal(account)
		if err != nil {
			return err
		}
		metaPayload, err := json.Marshal(buildSchedulerMetadataAccount(account))
		if err != nil {
			return err
		}

		id := strconv.FormatInt(account.ID, 10)
		pipe.Set(ctx, schedulerAccountKey(id), fullPayload, 0)
		pipe.Set(ctx, schedulerAccountMetaKey(id), metaPayload, 0)
		pending++
		if pending >= c.writeChunkSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}

	return flush()
}

func (c *schedulerCache) mgetChunked(ctx context.Context, keys []string) ([]any, error) {
	if len(keys) == 0 {
		return []any{}, nil
	}

	out := make([]any, 0, len(keys))
	chunkSize := c.mgetChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultSchedulerSnapshotMGetChunkSize
	}
	for start := 0; start < len(keys); start += chunkSize {
		end := start + chunkSize
		if end > len(keys) {
			end = len(keys)
		}
		part, err := c.rdb.MGet(ctx, keys[start:end]...).Result()
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	return out, nil
}

func buildSchedulerMetadataAccount(account service.Account) service.Account {
	return service.Account{
		ID:                      account.ID,
		Name:                    account.Name,
		Platform:                account.Platform,
		Type:                    account.Type,
		Concurrency:             account.Concurrency,
		LoadFactor:              account.LoadFactor,
		Priority:                account.Priority,
		RateMultiplier:          account.RateMultiplier,
		Status:                  account.Status,
		LastUsedAt:              account.LastUsedAt,
		ExpiresAt:               account.ExpiresAt,
		AutoPauseOnExpired:      account.AutoPauseOnExpired,
		Schedulable:             account.Schedulable,
		RateLimitedAt:           account.RateLimitedAt,
		RateLimitResetAt:        account.RateLimitResetAt,
		OverloadUntil:           account.OverloadUntil,
		TempUnschedulableUntil:  account.TempUnschedulableUntil,
		TempUnschedulableReason: account.TempUnschedulableReason,
		SessionWindowStart:      account.SessionWindowStart,
		SessionWindowEnd:        account.SessionWindowEnd,
		SessionWindowStatus:     account.SessionWindowStatus,
		ParentAccountID:         account.ParentAccountID,
		QuotaDimension:          account.QuotaDimension,
		AccountGroups:           filterSchedulerAccountGroups(account.AccountGroups),
		GroupIDs:                filterSchedulerGroupIDs(account.GroupIDs, account.AccountGroups),
		Credentials:             filterSchedulerCredentials(account.Credentials),
		Extra:                   filterSchedulerExtra(account.Extra),
	}
}

func filterSchedulerAccountGroups(accountGroups []service.AccountGroup) []service.AccountGroup {
	if len(accountGroups) == 0 {
		return nil
	}

	filtered := make([]service.AccountGroup, 0, len(accountGroups))
	for _, ag := range accountGroups {
		if ag.GroupID <= 0 {
			continue
		}
		filtered = append(filtered, service.AccountGroup{
			AccountID: ag.AccountID,
			GroupID:   ag.GroupID,
			Priority:  ag.Priority,
			CreatedAt: ag.CreatedAt,
		})
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func filterSchedulerGroupIDs(groupIDs []int64, accountGroups []service.AccountGroup) []int64 {
	if len(groupIDs) == 0 && len(accountGroups) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(groupIDs)+len(accountGroups))
	filtered := make([]int64, 0, len(groupIDs)+len(accountGroups))
	for _, id := range groupIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		filtered = append(filtered, id)
	}
	for _, ag := range accountGroups {
		if ag.GroupID <= 0 {
			continue
		}
		if _, ok := seen[ag.GroupID]; ok {
			continue
		}
		seen[ag.GroupID] = struct{}{}
		filtered = append(filtered, ag.GroupID)
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func filterSchedulerCredentials(credentials map[string]any) map[string]any {
	if len(credentials) == 0 {
		return nil
	}
	keys := []string{"model_mapping", "compact_model_mapping", "api_key", "project_id", "oauth_type"}
	filtered := make(map[string]any)
	for _, key := range keys {
		if value, ok := credentials[key]; ok && value != nil {
			filtered[key] = value
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func filterSchedulerExtra(extra map[string]any) map[string]any {
	if len(extra) == 0 {
		return nil
	}
	keys := []string{
		"mixed_scheduling",
		"window_cost_limit",
		"window_cost_sticky_reserve",
		"max_sessions",
		"session_idle_timeout_minutes",
		"openai_oauth_responses_websockets_v2_enabled",
		"openai_oauth_responses_websockets_v2_mode",
		"openai_apikey_responses_websockets_v2_enabled",
		"openai_apikey_responses_websockets_v2_mode",
		"responses_websockets_v2_enabled",
		"openai_ws_enabled",
		"openai_ws_force_http",
		"openai_responses_mode",
		"openai_responses_supported",
		"codex_5h_used_percent",
		"codex_7d_used_percent",
		"codex_5h_reset_at",
		"codex_7d_reset_at",
		"codex_5h_reset_after_seconds",
		"codex_7d_reset_after_seconds",
		"codex_usage_updated_at",
		"auto_pause_5h_threshold",
		"auto_pause_7d_threshold",
		"auto_pause_5h_disabled",
		"auto_pause_7d_disabled",
		"model_rate_limits",
	}
	filtered := make(map[string]any)
	for _, key := range keys {
		if value, ok := extra[key]; ok && value != nil {
			filtered[key] = value
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}
