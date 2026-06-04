package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) ListRequestDetails(ctx context.Context, filter *service.OpsRequestDetailFilter) ([]*service.OpsRequestDetail, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("nil ops repository")
	}

	page, pageSize, startTime, endTime := filter.Normalize()
	offset := (page - 1) * pageSize

	conditions := make([]string, 0, 24)
	args := make([]any, 0, 32)

	// Placeholders $1/$2 reserved for time window inside the CTE.
	args = append(args, startTime.UTC(), endTime.UTC())

	addCondition := func(condition string, values ...any) {
		conditions = append(conditions, condition)
		args = append(args, values...)
	}

	if filter != nil {
		if kind := strings.TrimSpace(strings.ToLower(filter.Kind)); kind != "" && kind != "all" {
			if kind != string(service.OpsRequestKindSuccess) && kind != string(service.OpsRequestKindError) {
				return nil, 0, fmt.Errorf("invalid kind")
			}
			addCondition(fmt.Sprintf("kind = $%d", len(args)+1), kind)
		}

		if platform := strings.TrimSpace(strings.ToLower(filter.Platform)); platform != "" {
			addCondition(fmt.Sprintf("platform = $%d", len(args)+1), platform)
		}
		if filter.GroupID != nil && *filter.GroupID > 0 {
			addCondition(fmt.Sprintf("group_id = $%d", len(args)+1), *filter.GroupID)
		}

		if filter.UserID != nil && *filter.UserID > 0 {
			addCondition(fmt.Sprintf("user_id = $%d", len(args)+1), *filter.UserID)
		}
		if filter.APIKeyID != nil && *filter.APIKeyID > 0 {
			addCondition(fmt.Sprintf("api_key_id = $%d", len(args)+1), *filter.APIKeyID)
		}
		if filter.AccountID != nil && *filter.AccountID > 0 {
			addCondition(fmt.Sprintf("account_id = $%d", len(args)+1), *filter.AccountID)
		}
		if filter.ChannelID != nil && *filter.ChannelID > 0 {
			addCondition(fmt.Sprintf("channel_id = $%d", len(args)+1), *filter.ChannelID)
		}

		if model := strings.TrimSpace(filter.Model); model != "" {
			addCondition(
				fmt.Sprintf("(model = $%d OR requested_model = $%d OR upstream_model = $%d)", len(args)+1, len(args)+2, len(args)+3),
				model, model, model,
			)
		}
		if endpoint := strings.TrimSpace(filter.Endpoint); endpoint != "" {
			addCondition(
				fmt.Sprintf("(inbound_endpoint = $%d OR upstream_endpoint = $%d OR request_path = $%d)", len(args)+1, len(args)+2, len(args)+3),
				endpoint, endpoint, endpoint,
			)
		}
		if filter.StatusCode != nil && *filter.StatusCode >= 0 {
			addCondition(fmt.Sprintf("status_code = $%d", len(args)+1), *filter.StatusCode)
		}
		if errorType := strings.TrimSpace(filter.ErrorType); errorType != "" {
			addCondition(fmt.Sprintf("error_type = $%d", len(args)+1), errorType)
		}
		if errorCode := strings.TrimSpace(filter.ErrorCode); errorCode != "" {
			addCondition(fmt.Sprintf("error_code = $%d", len(args)+1), errorCode)
		}
		if requestID := strings.TrimSpace(filter.RequestID); requestID != "" {
			addCondition(fmt.Sprintf("request_id = $%d", len(args)+1), requestID)
		}
		if q := strings.TrimSpace(filter.Query); q != "" {
			like := "%" + strings.ToLower(q) + "%"
			startIdx := len(args) + 1
			addCondition(
				fmt.Sprintf(`(
					LOWER(COALESCE(request_id,'')) LIKE $%d
					OR LOWER(COALESCE(model,'')) LIKE $%d
					OR LOWER(COALESCE(requested_model,'')) LIKE $%d
					OR LOWER(COALESCE(upstream_model,'')) LIKE $%d
					OR LOWER(COALESCE(message,'')) LIKE $%d
					OR LOWER(COALESCE(user_email,'')) LIKE $%d
					OR LOWER(COALESCE(api_key_name,'')) LIKE $%d
					OR LOWER(COALESCE(account_name,'')) LIKE $%d
					OR LOWER(COALESCE(channel_name,'')) LIKE $%d
				)`,
					startIdx, startIdx+1, startIdx+2, startIdx+3, startIdx+4,
					startIdx+5, startIdx+6, startIdx+7, startIdx+8,
				),
				like, like, like, like, like, like, like, like, like,
			)
		}

		if filter.MinDurationMs != nil {
			addCondition(fmt.Sprintf("duration_ms >= $%d", len(args)+1), *filter.MinDurationMs)
		}
		if filter.MaxDurationMs != nil {
			addCondition(fmt.Sprintf("duration_ms <= $%d", len(args)+1), *filter.MaxDurationMs)
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	cte := `
WITH combined AS (
  SELECT
    'success'::TEXT AS kind,
    ul.created_at AS created_at,
    ul.request_id AS request_id,
    COALESCE(NULLIF(g.platform, ''), NULLIF(a.platform, ''), '') AS platform,
    COALESCE(NULLIF(ul.requested_model, ''), NULLIF(ul.model, ''), '') AS model,
    COALESCE(NULLIF(ul.requested_model, ''), NULLIF(ul.model, ''), '') AS requested_model,
    COALESCE(NULLIF(ul.upstream_model, ''), '') AS upstream_model,
    COALESCE(NULLIF(ul.inbound_endpoint, ''), '') AS inbound_endpoint,
    COALESCE(NULLIF(ul.upstream_endpoint, ''), '') AS upstream_endpoint,
    COALESCE(NULLIF(ul.inbound_endpoint, ''), '') AS request_path,
    ul.duration_ms AS duration_ms,
    ul.first_token_ms AS first_token_ms,
    200::INT AS status_code,
    NULL::BIGINT AS error_id,
    NULL::TEXT AS phase,
    NULL::TEXT AS severity,
    NULL::TEXT AS error_code,
    NULL::TEXT AS error_type,
    NULL::TEXT AS message,
    ul.user_id AS user_id,
    COALESCE(u.email, '') AS user_email,
    ul.api_key_id AS api_key_id,
    COALESCE(k.name, '') AS api_key_name,
    ul.account_id AS account_id,
    COALESCE(a.name, '') AS account_name,
    ul.channel_id AS channel_id,
    COALESCE(ch.name, '') AS channel_name,
    ul.group_id AS group_id,
    COALESCE(g.name, '') AS group_name,
    ul.total_cost AS total_cost,
    ul.actual_cost AS actual_cost,
    COALESCE(ul.account_stats_cost, ul.total_cost * COALESCE(ul.account_rate_multiplier, 1)) AS account_cost,
    COALESCE(ul.ip_address, '') AS ip_address,
    COALESCE(ul.user_agent, '') AS user_agent,
    CASE COALESCE(NULLIF(ul.request_type, 0), CASE WHEN ul.openai_ws_mode THEN 3 WHEN ul.stream THEN 2 ELSE 1 END)
      WHEN 1 THEN 'sync'
      WHEN 2 THEN 'stream'
      WHEN 3 THEN 'ws_v2'
      ELSE 'unknown'
    END AS request_type,
    ul.stream AS stream
  FROM usage_logs ul
  LEFT JOIN groups g ON g.id = ul.group_id
  LEFT JOIN accounts a ON a.id = ul.account_id
  LEFT JOIN users u ON u.id = ul.user_id
  LEFT JOIN api_keys k ON k.id = ul.api_key_id
  LEFT JOIN channels ch ON ch.id = ul.channel_id
  WHERE ul.created_at >= $1 AND ul.created_at < $2

  UNION ALL

  SELECT
    'error'::TEXT AS kind,
    o.created_at AS created_at,
    COALESCE(NULLIF(o.request_id,''), NULLIF(o.client_request_id,''), '') AS request_id,
    COALESCE(NULLIF(o.platform, ''), NULLIF(g.platform, ''), NULLIF(a.platform, ''), '') AS platform,
    COALESCE(NULLIF(o.requested_model, ''), NULLIF(o.model, ''), '') AS model,
    COALESCE(NULLIF(o.requested_model, ''), NULLIF(o.model, ''), '') AS requested_model,
    COALESCE(NULLIF(o.upstream_model, ''), '') AS upstream_model,
    COALESCE(NULLIF(o.inbound_endpoint, ''), '') AS inbound_endpoint,
    COALESCE(NULLIF(o.upstream_endpoint, ''), '') AS upstream_endpoint,
    COALESCE(NULLIF(o.request_path, ''), NULLIF(o.inbound_endpoint, ''), '') AS request_path,
    o.duration_ms AS duration_ms,
    o.time_to_first_token_ms::INT AS first_token_ms,
    o.status_code AS status_code,
    o.id AS error_id,
    o.error_phase AS phase,
    o.severity AS severity,
    COALESCE(NULLIF(o.provider_error_code, ''), NULLIF(o.error_type, ''), '') AS error_code,
    o.error_type AS error_type,
    o.error_message AS message,
    o.user_id AS user_id,
    COALESCE(u.email, '') AS user_email,
    o.api_key_id AS api_key_id,
    COALESCE(k.name, '') AS api_key_name,
    o.account_id AS account_id,
    COALESCE(a.name, '') AS account_name,
    cg.channel_id AS channel_id,
    COALESCE(ch.name, '') AS channel_name,
    o.group_id AS group_id,
    COALESCE(g.name, '') AS group_name,
    0::NUMERIC AS total_cost,
    0::NUMERIC AS actual_cost,
    0::NUMERIC AS account_cost,
    COALESCE(o.client_ip::TEXT, '') AS ip_address,
    COALESCE(o.user_agent, '') AS user_agent,
    CASE COALESCE(o.request_type, CASE WHEN o.stream THEN 2 ELSE 1 END)
      WHEN 1 THEN 'sync'
      WHEN 2 THEN 'stream'
      WHEN 3 THEN 'ws_v2'
      ELSE 'unknown'
    END AS request_type,
    o.stream AS stream
  FROM ops_error_logs o
  LEFT JOIN groups g ON g.id = o.group_id
  LEFT JOIN accounts a ON a.id = o.account_id
  LEFT JOIN users u ON u.id = o.user_id
  LEFT JOIN api_keys k ON k.id = o.api_key_id
  LEFT JOIN channel_groups cg ON cg.group_id = o.group_id
  LEFT JOIN channels ch ON ch.id = cg.channel_id
  WHERE o.created_at >= $1 AND o.created_at < $2
    AND COALESCE(o.status_code, 0) >= 400
)
`

	countQuery := fmt.Sprintf(`%s SELECT COUNT(1) FROM combined %s`, cte, where)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		if err == sql.ErrNoRows {
			total = 0
		} else {
			return nil, 0, err
		}
	}

	sort := "ORDER BY created_at DESC"
	if filter != nil {
		switch strings.TrimSpace(strings.ToLower(filter.Sort)) {
		case "", "created_at_desc":
			// default
		case "duration_desc":
			sort = "ORDER BY duration_ms DESC NULLS LAST, created_at DESC"
		default:
			return nil, 0, fmt.Errorf("invalid sort")
		}
	}

	listQuery := fmt.Sprintf(`
%s
SELECT
  kind,
  created_at,
  request_id,
  platform,
  model,
  requested_model,
  upstream_model,
  inbound_endpoint,
  upstream_endpoint,
  request_path,
  duration_ms,
  first_token_ms,
  status_code,
  error_id,
  phase,
  severity,
  error_code,
  error_type,
  message,
  user_id,
  user_email,
  api_key_id,
  api_key_name,
  account_id,
  account_name,
  channel_id,
  channel_name,
  group_id,
  group_name,
  total_cost,
  actual_cost,
  account_cost,
  ip_address,
  user_agent,
  request_type,
  stream
FROM combined
%s
%s
LIMIT $%d OFFSET $%d
`, cte, where, sort, len(args)+1, len(args)+2)

	listArgs := append(append([]any{}, args...), pageSize, offset)
	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	toIntPtr := func(v sql.NullInt64) *int {
		if !v.Valid {
			return nil
		}
		i := int(v.Int64)
		return &i
	}
	toInt64Ptr := func(v sql.NullInt64) *int64 {
		if !v.Valid {
			return nil
		}
		i := v.Int64
		return &i
	}

	out := make([]*service.OpsRequestDetail, 0, pageSize)
	for rows.Next() {
		var (
			kind             string
			createdAt        time.Time
			requestID        sql.NullString
			platform         sql.NullString
			model            sql.NullString
			requestedModel   sql.NullString
			upstreamModel    sql.NullString
			inboundEndpoint  sql.NullString
			upstreamEndpoint sql.NullString
			requestPath      sql.NullString

			durationMs   sql.NullInt64
			firstTokenMs sql.NullInt64
			statusCode   sql.NullInt64
			errorID      sql.NullInt64

			phase     sql.NullString
			severity  sql.NullString
			errorCode sql.NullString
			errorType sql.NullString
			message   sql.NullString

			userID      sql.NullInt64
			userEmail   sql.NullString
			apiKeyID    sql.NullInt64
			apiKeyName  sql.NullString
			accountID   sql.NullInt64
			accountName sql.NullString
			channelID   sql.NullInt64
			channelName sql.NullString
			groupID     sql.NullInt64
			groupName   sql.NullString

			totalCost   float64
			actualCost  float64
			accountCost float64
			ipAddress   sql.NullString
			userAgent   sql.NullString
			requestType sql.NullString

			stream bool
		)

		if err := rows.Scan(
			&kind,
			&createdAt,
			&requestID,
			&platform,
			&model,
			&requestedModel,
			&upstreamModel,
			&inboundEndpoint,
			&upstreamEndpoint,
			&requestPath,
			&durationMs,
			&firstTokenMs,
			&statusCode,
			&errorID,
			&phase,
			&severity,
			&errorCode,
			&errorType,
			&message,
			&userID,
			&userEmail,
			&apiKeyID,
			&apiKeyName,
			&accountID,
			&accountName,
			&channelID,
			&channelName,
			&groupID,
			&groupName,
			&totalCost,
			&actualCost,
			&accountCost,
			&ipAddress,
			&userAgent,
			&requestType,
			&stream,
		); err != nil {
			return nil, 0, err
		}

		item := &service.OpsRequestDetail{
			Kind:             service.OpsRequestKind(kind),
			CreatedAt:        createdAt,
			RequestID:        strings.TrimSpace(requestID.String),
			Platform:         strings.TrimSpace(platform.String),
			Model:            strings.TrimSpace(model.String),
			RequestedModel:   strings.TrimSpace(requestedModel.String),
			UpstreamModel:    strings.TrimSpace(upstreamModel.String),
			InboundEndpoint:  strings.TrimSpace(inboundEndpoint.String),
			UpstreamEndpoint: strings.TrimSpace(upstreamEndpoint.String),
			RequestPath:      strings.TrimSpace(requestPath.String),

			DurationMs:   toIntPtr(durationMs),
			FirstTokenMs: toIntPtr(firstTokenMs),
			StatusCode:   toIntPtr(statusCode),
			ErrorID:      toInt64Ptr(errorID),
			Phase:        phase.String,
			Severity:     severity.String,
			ErrorCode:    strings.TrimSpace(errorCode.String),
			ErrorType:    strings.TrimSpace(errorType.String),
			Message:      message.String,

			UserID:      toInt64Ptr(userID),
			UserEmail:   strings.TrimSpace(userEmail.String),
			APIKeyID:    toInt64Ptr(apiKeyID),
			APIKeyName:  strings.TrimSpace(apiKeyName.String),
			AccountID:   toInt64Ptr(accountID),
			AccountName: strings.TrimSpace(accountName.String),
			ChannelID:   toInt64Ptr(channelID),
			ChannelName: strings.TrimSpace(channelName.String),
			GroupID:     toInt64Ptr(groupID),
			GroupName:   strings.TrimSpace(groupName.String),
			TotalCost:   totalCost,
			ActualCost:  actualCost,
			AccountCost: accountCost,
			IPAddress:   strings.TrimSpace(ipAddress.String),
			UserAgent:   strings.TrimSpace(userAgent.String),
			RequestType: strings.TrimSpace(requestType.String),

			Stream: stream,
		}

		if item.Platform == "" {
			item.Platform = "unknown"
		}

		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return out, total, nil
}
