package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *channelRepository) GetChannelStrategyStats(ctx context.Context, startTime, endTime time.Time) (map[int64]service.ChannelStrategyStats, error) {
	stats, err := r.loadChannelStrategyHealth(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}
	groups, err := r.loadChannelStrategyGroups(ctx)
	if err != nil {
		return nil, err
	}
	for channelID, groupRows := range groups {
		st := stats[channelID]
		st.ChannelID = channelID
		st.Groups = groupRows
		stats[channelID] = st
	}
	return stats, nil
}

func (r *channelRepository) loadChannelStrategyHealth(ctx context.Context, startTime, endTime time.Time) (map[int64]service.ChannelStrategyStats, error) {
	query := `
WITH success AS (
  SELECT
    COALESCE(ul.channel_id, cg.channel_id) AS channel_id,
    COUNT(*) AS success_count,
    COALESCE(SUM(ul.actual_cost), 0) AS actual_cost,
    COALESCE(SUM(COALESCE(ul.account_stats_cost, ul.total_cost) * COALESCE(ul.account_rate_multiplier, 1)), 0) AS account_cost,
    COALESCE(SUM(ul.duration_ms), 0) AS duration_sum,
    COUNT(ul.duration_ms) AS duration_count
  FROM usage_logs ul
  LEFT JOIN channel_groups cg ON cg.group_id = ul.group_id
  WHERE ul.created_at >= $1 AND ul.created_at < $2
  GROUP BY 1
),
errors AS (
  SELECT
    cg.channel_id AS channel_id,
    COUNT(*) AS error_count,
    COALESCE(SUM(o.duration_ms), 0) AS duration_sum,
    COUNT(o.duration_ms) AS duration_count
  FROM ops_error_logs o
  LEFT JOIN channel_groups cg ON cg.group_id = o.group_id
  WHERE o.created_at >= $1 AND o.created_at < $2
    AND COALESCE(o.status_code, 0) >= 400
    AND o.is_count_tokens = FALSE
  GROUP BY 1
),
latest_error AS (
  SELECT DISTINCT ON (cg.channel_id)
    cg.channel_id AS channel_id,
    o.created_at AS last_error_at,
    COALESCE(NULLIF(o.provider_error_code, ''), NULLIF(o.error_type, ''), '') || CASE
      WHEN COALESCE(o.error_message, '') = '' THEN ''
      ELSE ': ' || LEFT(o.error_message, 180)
    END AS last_error
  FROM ops_error_logs o
  LEFT JOIN channel_groups cg ON cg.group_id = o.group_id
  WHERE o.created_at >= $1 AND o.created_at < $2
    AND COALESCE(o.status_code, 0) >= 400
    AND o.is_count_tokens = FALSE
    AND cg.channel_id IS NOT NULL
  ORDER BY cg.channel_id, o.created_at DESC
)
SELECT
  ch.id,
  COALESCE(s.success_count, 0) AS success_count,
  COALESCE(e.error_count, 0) AS error_count,
  COALESCE(s.success_count, 0) + COALESCE(e.error_count, 0) AS request_count,
  CASE
    WHEN COALESCE(s.success_count, 0) + COALESCE(e.error_count, 0) > 0
      THEN COALESCE(e.error_count, 0)::float / (COALESCE(s.success_count, 0) + COALESCE(e.error_count, 0))
    ELSE 0
  END AS error_rate,
  CASE
    WHEN COALESCE(s.duration_count, 0) + COALESCE(e.duration_count, 0) > 0
      THEN (COALESCE(s.duration_sum, 0) + COALESCE(e.duration_sum, 0))::float / (COALESCE(s.duration_count, 0) + COALESCE(e.duration_count, 0))
    ELSE 0
  END AS avg_duration_ms,
  COALESCE(s.actual_cost, 0) AS actual_cost,
  COALESCE(s.account_cost, 0) AS account_cost,
  le.last_error_at,
  COALESCE(le.last_error, '') AS last_error
FROM channels ch
LEFT JOIN success s ON s.channel_id = ch.id
LEFT JOIN errors e ON e.channel_id = ch.id
LEFT JOIN latest_error le ON le.channel_id = ch.id`

	rows, err := r.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("load channel strategy health: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int64]service.ChannelStrategyStats)
	for rows.Next() {
		var st service.ChannelStrategyStats
		var lastErrorAt sql.NullTime
		if err := rows.Scan(
			&st.ChannelID,
			&st.SuccessCount,
			&st.ErrorCount,
			&st.RequestCount,
			&st.ErrorRate,
			&st.AvgDurationMs,
			&st.ActualCost,
			&st.AccountCost,
			&lastErrorAt,
			&st.LastError,
		); err != nil {
			return nil, fmt.Errorf("scan channel strategy health: %w", err)
		}
		if lastErrorAt.Valid {
			t := lastErrorAt.Time.UTC()
			st.LastErrorAt = &t
		}
		result[st.ChannelID] = st
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate channel strategy health: %w", err)
	}
	return result, nil
}

func (r *channelRepository) loadChannelStrategyGroups(ctx context.Context) (map[int64][]service.ChannelStrategyGroup, error) {
	query := `
SELECT
  cg.channel_id,
  g.id,
  g.name,
  g.platform,
  g.status,
  g.rate_multiplier,
  COUNT(ag.account_id) AS account_count,
  COUNT(ag.account_id) FILTER (WHERE a.status = 'active' AND COALESCE(a.schedulable, TRUE)) AS active_account_count,
  MIN(ag.priority) AS priority_min,
  MAX(ag.priority) AS priority_max
FROM channel_groups cg
JOIN groups g ON g.id = cg.group_id
LEFT JOIN account_groups ag ON ag.group_id = g.id
LEFT JOIN accounts a ON a.id = ag.account_id AND a.deleted_at IS NULL
WHERE g.deleted_at IS NULL
GROUP BY cg.channel_id, g.id, g.name, g.platform, g.status, g.rate_multiplier
ORDER BY cg.channel_id, g.sort_order ASC, g.id ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load channel strategy groups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int64][]service.ChannelStrategyGroup)
	for rows.Next() {
		var channelID int64
		var group service.ChannelStrategyGroup
		var priorityMin, priorityMax sql.NullInt64
		if err := rows.Scan(
			&channelID,
			&group.ID,
			&group.Name,
			&group.Platform,
			&group.Status,
			&group.RateMultiplier,
			&group.AccountCount,
			&group.ActiveAccountCount,
			&priorityMin,
			&priorityMax,
		); err != nil {
			return nil, fmt.Errorf("scan channel strategy group: %w", err)
		}
		if priorityMin.Valid {
			v := int(priorityMin.Int64)
			group.PriorityMin = &v
		}
		if priorityMax.Valid {
			v := int(priorityMax.Int64)
			group.PriorityMax = &v
		}
		result[channelID] = append(result[channelID], group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate channel strategy groups: %w", err)
	}
	return result, nil
}

func (r *channelRepository) BatchUpdateChannelStatus(ctx context.Context, channelIDs []int64, status string) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE channels SET status = $1, updated_at = NOW() WHERE id = ANY($2)`,
		status, pq.Array(channelIDs),
	)
	if err != nil {
		return 0, fmt.Errorf("batch update channel status: %w", err)
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

func (r *channelRepository) BatchReplaceChannelModelPricing(ctx context.Context, channelIDs []int64, pricing []service.ChannelModelPricing) error {
	return r.runInTx(ctx, func(tx *sql.Tx) error {
		for _, channelID := range channelIDs {
			if err := ensureChannelExistsTx(ctx, tx, channelID); err != nil {
				return err
			}
			copiedPricing := cloneModelPricingForWrite(pricing)
			if err := replaceModelPricingTx(ctx, tx, channelID, copiedPricing); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *channelRepository) CopyChannelStrategy(ctx context.Context, sourceChannelID int64, targetChannelIDs []int64, opts service.ChannelStrategyCopyOptions) error {
	return r.runInTx(ctx, func(tx *sql.Tx) error {
		source, err := getChannelForStrategyCopyTx(ctx, tx, sourceChannelID)
		if err != nil {
			return err
		}
		for _, targetID := range targetChannelIDs {
			if err := ensureChannelExistsTx(ctx, tx, targetID); err != nil {
				return err
			}
			if opts.CopyFlags {
				if err := copyChannelFlagsTx(ctx, tx, targetID, source); err != nil {
					return err
				}
			}
			if opts.CopyModelMapping {
				if err := copyChannelModelMappingTx(ctx, tx, targetID, source.ModelMapping); err != nil {
					return err
				}
			}
			if opts.CopyModelPricing {
				copiedPricing := cloneModelPricingForWrite(source.ModelPricing)
				if err := replaceModelPricingTx(ctx, tx, targetID, copiedPricing); err != nil {
					return err
				}
			}
			if opts.CopyAccountStatsPricing {
				copiedRules := cloneAccountStatsPricingRulesForWrite(source.AccountStatsPricingRules)
				if err := replaceAccountStatsPricingRulesTx(ctx, tx, targetID, copiedRules); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func getChannelForStrategyCopyTx(ctx context.Context, tx *sql.Tx, channelID int64) (*service.Channel, error) {
	ch := &service.Channel{ID: channelID}
	var modelMappingJSON, featuresConfigJSON []byte
	err := tx.QueryRowContext(ctx,
		`SELECT model_mapping, billing_model_source, restrict_models, features, features_config, apply_pricing_to_account_stats
		 FROM channels WHERE id = $1`,
		channelID,
	).Scan(&modelMappingJSON, &ch.BillingModelSource, &ch.RestrictModels, &ch.Features, &featuresConfigJSON, &ch.ApplyPricingToAccountStats)
	if err == sql.ErrNoRows {
		return nil, service.ErrChannelNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get source channel for strategy copy: %w", err)
	}
	ch.ModelMapping = unmarshalModelMapping(modelMappingJSON)
	ch.FeaturesConfig = unmarshalFeaturesConfig(featuresConfigJSON)

	pricing, err := listModelPricingTx(ctx, tx, channelID)
	if err != nil {
		return nil, err
	}
	ch.ModelPricing = pricing

	rules, err := loadAccountStatsPricingRulesTx(ctx, tx, channelID)
	if err != nil {
		return nil, err
	}
	ch.AccountStatsPricingRules = rules
	return ch, nil
}

func ensureChannelExistsTx(ctx context.Context, tx *sql.Tx, channelID int64) error {
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM channels WHERE id = $1)`, channelID).Scan(&exists); err != nil {
		return fmt.Errorf("check channel exists: %w", err)
	}
	if !exists {
		return service.ErrChannelNotFound
	}
	return nil
}

func copyChannelFlagsTx(ctx context.Context, tx *sql.Tx, targetID int64, source *service.Channel) error {
	featuresConfigJSON, err := marshalFeaturesConfig(source.FeaturesConfig)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE channels
		 SET billing_model_source = $1,
		     restrict_models = $2,
		     features = $3,
		     features_config = $4,
		     apply_pricing_to_account_stats = $5,
		     updated_at = NOW()
		 WHERE id = $6`,
		source.BillingModelSource,
		source.RestrictModels,
		source.Features,
		featuresConfigJSON,
		source.ApplyPricingToAccountStats,
		targetID,
	)
	if err != nil {
		return fmt.Errorf("copy channel flags: %w", err)
	}
	return nil
}

func copyChannelModelMappingTx(ctx context.Context, tx *sql.Tx, targetID int64, mapping map[string]map[string]string) error {
	modelMappingJSON, err := marshalModelMapping(mapping)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE channels SET model_mapping = $1, updated_at = NOW() WHERE id = $2`,
		modelMappingJSON,
		targetID,
	)
	if err != nil {
		return fmt.Errorf("copy channel model mapping: %w", err)
	}
	return nil
}

func listModelPricingTx(ctx context.Context, tx *sql.Tx, channelID int64) ([]service.ChannelModelPricing, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id, channel_id, platform, models, billing_mode, input_price, output_price, cache_write_price, cache_read_price, image_output_price, per_request_price, created_at, updated_at
		 FROM channel_model_pricing WHERE channel_id = $1 ORDER BY id`, channelID,
	)
	if err != nil {
		return nil, fmt.Errorf("list model pricing for strategy copy: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result, pricingIDs, err := scanModelPricingRows(rows)
	if err != nil {
		return nil, err
	}
	if len(pricingIDs) > 0 {
		intervals, err := batchLoadIntervalsExec(ctx, tx, pricingIDs)
		if err != nil {
			return nil, err
		}
		for i := range result {
			result[i].Intervals = intervals[result[i].ID]
		}
	}
	return result, nil
}

func loadAccountStatsPricingRulesTx(ctx context.Context, tx *sql.Tx, channelID int64) ([]service.AccountStatsPricingRule, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id, channel_id, name, group_ids, account_ids, sort_order, created_at, updated_at
		 FROM channel_account_stats_pricing_rules WHERE channel_id = $1 ORDER BY sort_order, id`,
		channelID,
	)
	if err != nil {
		return nil, fmt.Errorf("load account stats pricing rules for strategy copy: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var rules []service.AccountStatsPricingRule
	var ruleIDs []int64
	for rows.Next() {
		var rule service.AccountStatsPricingRule
		if err := rows.Scan(
			&rule.ID,
			&rule.ChannelID,
			&rule.Name,
			pq.Array(&rule.GroupIDs),
			pq.Array(&rule.AccountIDs),
			&rule.SortOrder,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan account stats pricing rule for strategy copy: %w", err)
		}
		ruleIDs = append(ruleIDs, rule.ID)
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	pricingMap, err := batchLoadAccountStatsModelPricingExec(ctx, tx, ruleIDs)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].Pricing = pricingMap[rules[i].ID]
	}
	return rules, nil
}

func batchLoadIntervalsExec(ctx context.Context, exec dbExec, pricingIDs []int64) (map[int64][]service.PricingInterval, error) {
	rows, err := exec.QueryContext(ctx,
		`SELECT id, pricing_id, min_tokens, max_tokens, tier_label,
		        input_price, output_price, cache_write_price, cache_read_price,
		        per_request_price, sort_order, created_at, updated_at
		 FROM channel_pricing_intervals
		 WHERE pricing_id = ANY($1) ORDER BY pricing_id, sort_order, id`,
		pq.Array(pricingIDs),
	)
	if err != nil {
		return nil, fmt.Errorf("batch load intervals for strategy copy: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanPricingIntervals(rows)
}

func batchLoadAccountStatsModelPricingExec(ctx context.Context, exec dbExec, ruleIDs []int64) (map[int64][]service.ChannelModelPricing, error) {
	if len(ruleIDs) == 0 {
		return make(map[int64][]service.ChannelModelPricing), nil
	}
	rows, err := exec.QueryContext(ctx,
		`SELECT id, rule_id, platform, models, billing_mode, input_price, output_price,
		        cache_write_price, cache_read_price, image_output_price, per_request_price, created_at, updated_at
		 FROM channel_account_stats_model_pricing WHERE rule_id = ANY($1) ORDER BY rule_id, id`,
		pq.Array(ruleIDs),
	)
	if err != nil {
		return nil, fmt.Errorf("batch load account stats model pricing for strategy copy: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int64][]service.ChannelModelPricing, len(ruleIDs))
	var pricingIDs []int64
	for rows.Next() {
		var p service.ChannelModelPricing
		var ruleID int64
		var modelsJSON []byte
		if err := rows.Scan(
			&p.ID, &ruleID, &p.Platform, &modelsJSON, &p.BillingMode,
			&p.InputPrice, &p.OutputPrice, &p.CacheWritePrice, &p.CacheReadPrice,
			&p.ImageOutputPrice, &p.PerRequestPrice, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan account stats model pricing for strategy copy: %w", err)
		}
		if err := json.Unmarshal(modelsJSON, &p.Models); err != nil {
			p.Models = []string{}
		}
		pricingIDs = append(pricingIDs, p.ID)
		result[ruleID] = append(result[ruleID], p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(pricingIDs) > 0 {
		intervals, err := batchLoadAccountStatsIntervalsExec(ctx, exec, pricingIDs)
		if err != nil {
			return nil, err
		}
		for ruleID, pricings := range result {
			for i := range pricings {
				pricings[i].Intervals = intervals[pricings[i].ID]
			}
			result[ruleID] = pricings
		}
	}
	return result, nil
}

func batchLoadAccountStatsIntervalsExec(ctx context.Context, exec dbExec, pricingIDs []int64) (map[int64][]service.PricingInterval, error) {
	rows, err := exec.QueryContext(ctx,
		`SELECT id, pricing_id, min_tokens, max_tokens, tier_label,
		        input_price, output_price, cache_write_price, cache_read_price,
		        per_request_price, sort_order, created_at, updated_at
		 FROM channel_account_stats_pricing_intervals
		 WHERE pricing_id = ANY($1) ORDER BY pricing_id, sort_order, id`,
		pq.Array(pricingIDs),
	)
	if err != nil {
		return nil, fmt.Errorf("batch load account stats intervals for strategy copy: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanPricingIntervals(rows)
}

func scanPricingIntervals(rows *sql.Rows) (map[int64][]service.PricingInterval, error) {
	result := make(map[int64][]service.PricingInterval)
	for rows.Next() {
		var iv service.PricingInterval
		if err := rows.Scan(
			&iv.ID, &iv.PricingID, &iv.MinTokens, &iv.MaxTokens, &iv.TierLabel,
			&iv.InputPrice, &iv.OutputPrice, &iv.CacheWritePrice, &iv.CacheReadPrice,
			&iv.PerRequestPrice, &iv.SortOrder, &iv.CreatedAt, &iv.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pricing interval: %w", err)
		}
		result[iv.PricingID] = append(result[iv.PricingID], iv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pricing intervals: %w", err)
	}
	return result, nil
}

func cloneModelPricingForWrite(src []service.ChannelModelPricing) []service.ChannelModelPricing {
	dst := make([]service.ChannelModelPricing, len(src))
	for i := range src {
		dst[i] = src[i].Clone()
		dst[i].ID = 0
		dst[i].ChannelID = 0
		dst[i].CreatedAt = time.Time{}
		dst[i].UpdatedAt = time.Time{}
		for j := range dst[i].Intervals {
			dst[i].Intervals[j].ID = 0
			dst[i].Intervals[j].PricingID = 0
			dst[i].Intervals[j].CreatedAt = time.Time{}
			dst[i].Intervals[j].UpdatedAt = time.Time{}
		}
	}
	return dst
}

func cloneAccountStatsPricingRulesForWrite(src []service.AccountStatsPricingRule) []service.AccountStatsPricingRule {
	dst := make([]service.AccountStatsPricingRule, len(src))
	for i := range src {
		dst[i] = src[i]
		dst[i].ID = 0
		dst[i].ChannelID = 0
		dst[i].CreatedAt = time.Time{}
		dst[i].UpdatedAt = time.Time{}
		if src[i].GroupIDs != nil {
			dst[i].GroupIDs = append([]int64(nil), src[i].GroupIDs...)
		}
		if src[i].AccountIDs != nil {
			dst[i].AccountIDs = append([]int64(nil), src[i].AccountIDs...)
		}
		dst[i].Pricing = cloneModelPricingForWrite(src[i].Pricing)
	}
	return dst
}
