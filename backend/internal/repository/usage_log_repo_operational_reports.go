package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

func (r *usageLogRepository) GetOperationalReports(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) (*usagestats.OperationalReportsResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	if granularity != "hour" && granularity != "day" {
		granularity = "day"
	}

	requestTrend, err := r.getOperationalRequestTrend(ctx, startTime, endTime, granularity)
	if err != nil {
		return nil, fmt.Errorf("get operational request trend: %w", err)
	}
	errorTrend, err := r.getOperationalErrorTrend(ctx, startTime, endTime, granularity)
	if err != nil {
		return nil, fmt.Errorf("get operational error trend: %w", err)
	}
	modelRanking, err := r.getOperationalModelCostRanking(ctx, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("get operational model cost ranking: %w", err)
	}
	channelRanking, err := r.getOperationalChannelHealthRanking(ctx, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("get operational channel health ranking: %w", err)
	}
	userRanking, err := r.GetUserSpendingRanking(ctx, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("get operational user spending ranking: %w", err)
	}
	keyRanking, err := r.getOperationalAPIKeySpendingRanking(ctx, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("get operational api key spending ranking: %w", err)
	}

	var totalRequests int64
	var totalTokens int64
	var totalActualCost float64
	for _, p := range requestTrend {
		totalRequests += p.TotalRequests
		totalTokens += p.TotalTokens
		totalActualCost += p.ActualCost
	}

	var users []usagestats.UserSpendingRankingItem
	if userRanking != nil {
		users = userRanking.Ranking
	}

	return &usagestats.OperationalReportsResponse{
		RequestTrend:          requestTrend,
		ErrorTrend:            errorTrend,
		ModelCostRanking:      modelRanking,
		ChannelHealthRanking:  channelRanking,
		UserSpendingRanking:   users,
		APIKeySpendingRanking: keyRanking,
		TotalActualCost:       totalActualCost,
		TotalRequests:         totalRequests,
		TotalTokens:           totalTokens,
	}, nil
}

func (r *usageLogRepository) getOperationalRequestTrend(ctx context.Context, startTime, endTime time.Time, granularity string) (results []usagestats.OperationalRequestTrendPoint, err error) {
	dateFormat := safeDateFormat(granularity)
	query := fmt.Sprintf(`
WITH success AS (
  SELECT
    TO_CHAR(created_at, '%s') AS bucket,
    COUNT(*) AS success_count,
    COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) AS total_tokens,
    COALESCE(SUM(actual_cost), 0) AS actual_cost,
    COALESCE(SUM(duration_ms), 0) AS duration_sum,
    COUNT(duration_ms) AS duration_count
  FROM usage_logs
  WHERE created_at >= $1 AND created_at < $2
  GROUP BY 1
),
errors AS (
  SELECT
    TO_CHAR(created_at, '%s') AS bucket,
    COUNT(*) AS error_count,
    COALESCE(SUM(duration_ms), 0) AS duration_sum,
    COUNT(duration_ms) AS duration_count
  FROM ops_error_logs
  WHERE created_at >= $1 AND created_at < $2
    AND COALESCE(status_code, 0) >= 400
    AND is_count_tokens = FALSE
  GROUP BY 1
),
combined AS (
  SELECT
    COALESCE(s.bucket, e.bucket) AS bucket,
    COALESCE(s.success_count, 0) AS success_count,
    COALESCE(e.error_count, 0) AS error_count,
    COALESCE(s.total_tokens, 0) AS total_tokens,
    COALESCE(s.actual_cost, 0) AS actual_cost,
    COALESCE(s.duration_sum, 0) + COALESCE(e.duration_sum, 0) AS duration_sum,
    COALESCE(s.duration_count, 0) + COALESCE(e.duration_count, 0) AS duration_count
  FROM success s
  FULL OUTER JOIN errors e ON e.bucket = s.bucket
)
SELECT
  bucket,
  success_count,
  error_count,
  success_count + error_count AS total_requests,
  total_tokens,
  actual_cost,
  CASE WHEN duration_count > 0 THEN duration_sum::float / duration_count ELSE 0 END AS avg_duration_ms
FROM combined
ORDER BY bucket ASC
`, dateFormat, dateFormat)

	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()

	results = make([]usagestats.OperationalRequestTrendPoint, 0)
	for rows.Next() {
		var row usagestats.OperationalRequestTrendPoint
		if err = rows.Scan(&row.Date, &row.SuccessCount, &row.ErrorCount, &row.TotalRequests, &row.TotalTokens, &row.ActualCost, &row.AvgDurationMs); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *usageLogRepository) getOperationalErrorTrend(ctx context.Context, startTime, endTime time.Time, granularity string) (results []usagestats.OperationalErrorTrendPoint, err error) {
	dateFormat := safeDateFormat(granularity)
	query := fmt.Sprintf(`
SELECT
  TO_CHAR(created_at, '%s') AS bucket,
  COUNT(*) FILTER (WHERE COALESCE(status_code, 0) >= 400) AS error_total,
  COUNT(*) FILTER (WHERE COALESCE(status_code, 0) >= 400 AND is_business_limited) AS business_limited,
  COUNT(*) FILTER (WHERE COALESCE(status_code, 0) >= 400 AND NOT is_business_limited) AS error_sla,
  COUNT(*) FILTER (WHERE error_owner = 'provider' AND NOT is_business_limited AND COALESCE(upstream_status_code, status_code, 0) NOT IN (429, 529)) AS upstream_excl,
  COUNT(*) FILTER (WHERE error_owner = 'provider' AND NOT is_business_limited AND COALESCE(upstream_status_code, status_code, 0) = 429) AS upstream_429,
  COUNT(*) FILTER (WHERE error_owner = 'provider' AND NOT is_business_limited AND COALESCE(upstream_status_code, status_code, 0) = 529) AS upstream_529
FROM ops_error_logs
WHERE created_at >= $1 AND created_at < $2
  AND COALESCE(status_code, 0) >= 400
  AND is_count_tokens = FALSE
GROUP BY 1
ORDER BY 1 ASC
`, dateFormat)

	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()

	results = make([]usagestats.OperationalErrorTrendPoint, 0)
	for rows.Next() {
		var row usagestats.OperationalErrorTrendPoint
		if err = rows.Scan(
			&row.Date,
			&row.ErrorCountTotal,
			&row.BusinessLimitedCount,
			&row.ErrorCountSLA,
			&row.UpstreamErrorCountExcl429,
			&row.Upstream429Count,
			&row.Upstream529Count,
		); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *usageLogRepository) getOperationalModelCostRanking(ctx context.Context, startTime, endTime time.Time, limit int) (results []usagestats.ModelCostRankingItem, err error) {
	query := `
SELECT
  COALESCE(NULLIF(ul.requested_model, ''), NULLIF(ul.model, ''), '') AS model,
  COALESCE(NULLIF(g.platform, ''), NULLIF(a.platform, ''), '') AS platform,
  COUNT(*) AS requests,
  COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0) AS tokens,
  COALESCE(SUM(ul.total_cost), 0) AS cost,
  COALESCE(SUM(ul.actual_cost), 0) AS actual_cost,
  COALESCE(SUM(COALESCE(ul.account_stats_cost, ul.total_cost) * COALESCE(ul.account_rate_multiplier, 1)), 0) AS account_cost
FROM usage_logs ul
LEFT JOIN groups g ON g.id = ul.group_id
LEFT JOIN accounts a ON a.id = ul.account_id
WHERE ul.created_at >= $1 AND ul.created_at < $2
GROUP BY 1, 2
ORDER BY actual_cost DESC, tokens DESC, requests DESC
LIMIT $3`

	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()

	results = make([]usagestats.ModelCostRankingItem, 0)
	for rows.Next() {
		var row usagestats.ModelCostRankingItem
		if err = rows.Scan(&row.Model, &row.Platform, &row.Requests, &row.Tokens, &row.Cost, &row.ActualCost, &row.AccountCost); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *usageLogRepository) getOperationalChannelHealthRanking(ctx context.Context, startTime, endTime time.Time, limit int) (results []usagestats.ChannelHealthRankingItem, err error) {
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
),
group_counts AS (
  SELECT channel_id, COUNT(*) AS group_count
  FROM channel_groups
  GROUP BY channel_id
)
SELECT
  ch.id,
  ch.name,
  ch.status,
  COALESCE(gc.group_count, 0) AS group_count,
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
LEFT JOIN group_counts gc ON gc.channel_id = ch.id
LEFT JOIN success s ON s.channel_id = ch.id
LEFT JOIN errors e ON e.channel_id = ch.id
LEFT JOIN latest_error le ON le.channel_id = ch.id
ORDER BY error_rate DESC, request_count DESC, actual_cost DESC, ch.id ASC
LIMIT $3`

	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()

	results = make([]usagestats.ChannelHealthRankingItem, 0)
	for rows.Next() {
		var row usagestats.ChannelHealthRankingItem
		var lastErrorAt sql.NullTime
		if err = rows.Scan(
			&row.ChannelID,
			&row.ChannelName,
			&row.Status,
			&row.GroupCount,
			&row.SuccessCount,
			&row.ErrorCount,
			&row.RequestCount,
			&row.ErrorRate,
			&row.AvgDurationMs,
			&row.ActualCost,
			&row.AccountCost,
			&lastErrorAt,
			&row.LastError,
		); err != nil {
			return nil, err
		}
		if lastErrorAt.Valid {
			row.LastErrorAt = lastErrorAt.Time.UTC().Format(time.RFC3339)
		}
		results = append(results, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *usageLogRepository) getOperationalAPIKeySpendingRanking(ctx context.Context, startTime, endTime time.Time, limit int) (results []usagestats.APIKeySpendingRankingItem, err error) {
	query := `
SELECT
  ul.api_key_id,
  COALESCE(k.name, '') AS key_name,
  ul.user_id,
  COALESCE(u.email, '') AS user_email,
  COALESCE(SUM(ul.actual_cost), 0) AS actual_cost,
  COUNT(*) AS requests,
  COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0) AS tokens
FROM usage_logs ul
LEFT JOIN api_keys k ON k.id = ul.api_key_id
LEFT JOIN users u ON u.id = ul.user_id
WHERE ul.created_at >= $1 AND ul.created_at < $2
GROUP BY ul.api_key_id, k.name, ul.user_id, u.email
ORDER BY actual_cost DESC, tokens DESC, requests DESC
LIMIT $3`

	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()

	results = make([]usagestats.APIKeySpendingRankingItem, 0)
	for rows.Next() {
		var row usagestats.APIKeySpendingRankingItem
		if err = rows.Scan(&row.APIKeyID, &row.KeyName, &row.UserID, &row.UserEmail, &row.ActualCost, &row.Requests, &row.Tokens); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
