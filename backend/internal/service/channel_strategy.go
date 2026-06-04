package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type ChannelStrategyGroup struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	Platform           string  `json:"platform"`
	Status             string  `json:"status"`
	RateMultiplier     float64 `json:"rate_multiplier"`
	AccountCount       int64   `json:"account_count"`
	ActiveAccountCount int64   `json:"active_account_count"`
	PriorityMin        *int    `json:"priority_min,omitempty"`
	PriorityMax        *int    `json:"priority_max,omitempty"`
}

type ChannelStrategyStats struct {
	ChannelID     int64
	Groups        []ChannelStrategyGroup
	RequestCount  int64
	SuccessCount  int64
	ErrorCount    int64
	ErrorRate     float64
	AvgDurationMs float64
	ActualCost    float64
	AccountCost   float64
	LastErrorAt   *time.Time
	LastError     string
}

type ChannelStrategyRow struct {
	ChannelID                     int64                  `json:"channel_id"`
	ChannelName                   string                 `json:"channel_name"`
	Description                   string                 `json:"description"`
	Status                        string                 `json:"status"`
	BillingModelSource            string                 `json:"billing_model_source"`
	RestrictModels                bool                   `json:"restrict_models"`
	GroupCount                    int                    `json:"group_count"`
	Groups                        []ChannelStrategyGroup `json:"groups"`
	Platforms                     []string               `json:"platforms"`
	ModelMappingCount             int                    `json:"model_mapping_count"`
	ModelPricingCount             int                    `json:"model_pricing_count"`
	PricingModelCount             int                    `json:"pricing_model_count"`
	BillingModes                  []string               `json:"billing_modes"`
	ModelSamples                  []string               `json:"model_samples"`
	ApplyPricingToAccountStats    bool                   `json:"apply_pricing_to_account_stats"`
	AccountStatsPricingRulesCount int                    `json:"account_stats_pricing_rules_count"`
	RequestCount                  int64                  `json:"request_count"`
	SuccessCount                  int64                  `json:"success_count"`
	ErrorCount                    int64                  `json:"error_count"`
	ErrorRate                     float64                `json:"error_rate"`
	AvgDurationMs                 float64                `json:"avg_duration_ms"`
	ActualCost                    float64                `json:"actual_cost"`
	AccountCost                   float64                `json:"account_cost"`
	LastErrorAt                   *time.Time             `json:"last_error_at,omitempty"`
	LastError                     string                 `json:"last_error,omitempty"`
}

type ChannelStrategyView struct {
	Items     []ChannelStrategyRow `json:"items"`
	StartTime time.Time            `json:"start_time"`
	EndTime   time.Time            `json:"end_time"`
}

type ChannelStrategyCopyOptions struct {
	CopyModelPricing        bool
	CopyModelMapping        bool
	CopyFlags               bool
	CopyAccountStatsPricing bool
}

type channelStrategyRepository interface {
	GetChannelStrategyStats(ctx context.Context, startTime, endTime time.Time) (map[int64]ChannelStrategyStats, error)
	BatchUpdateChannelStatus(ctx context.Context, channelIDs []int64, status string) (int64, error)
	BatchReplaceChannelModelPricing(ctx context.Context, channelIDs []int64, pricing []ChannelModelPricing) error
	CopyChannelStrategy(ctx context.Context, sourceChannelID int64, targetChannelIDs []int64, opts ChannelStrategyCopyOptions) error
}

func (s *ChannelService) strategyRepo() (channelStrategyRepository, error) {
	repo, ok := s.repo.(channelStrategyRepository)
	if !ok {
		return nil, infraerrors.InternalServer("CHANNEL_STRATEGY_REPOSITORY_UNAVAILABLE", "channel strategy repository is unavailable")
	}
	return repo, nil
}

func (s *ChannelService) GetStrategyView(ctx context.Context, startTime, endTime time.Time) (*ChannelStrategyView, error) {
	if !startTime.Before(endTime) {
		return nil, infraerrors.BadRequest("INVALID_TIME_RANGE", "start_time must be before end_time")
	}
	repo, err := s.strategyRepo()
	if err != nil {
		return nil, err
	}

	channels, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	stats, err := repo.GetChannelStrategyStats(ctx, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get channel strategy stats: %w", err)
	}

	rows := make([]ChannelStrategyRow, 0, len(channels))
	for i := range channels {
		channels[i].normalizeBillingModelSource()
		st := stats[channels[i].ID]
		rows = append(rows, channelToStrategyRow(&channels[i], st))
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Status != rows[j].Status {
			return rows[i].Status == StatusActive
		}
		if rows[i].RequestCount != rows[j].RequestCount {
			return rows[i].RequestCount > rows[j].RequestCount
		}
		return rows[i].ChannelID < rows[j].ChannelID
	})

	return &ChannelStrategyView{
		Items:     rows,
		StartTime: startTime,
		EndTime:   endTime,
	}, nil
}

func channelToStrategyRow(ch *Channel, st ChannelStrategyStats) ChannelStrategyRow {
	platforms := setFromGroups(st.Groups)
	for _, p := range ch.ModelPricing {
		if p.Platform != "" {
			platforms[p.Platform] = struct{}{}
		}
	}
	row := ChannelStrategyRow{
		ChannelID:                     ch.ID,
		ChannelName:                   ch.Name,
		Description:                   ch.Description,
		Status:                        ch.Status,
		BillingModelSource:            ch.BillingModelSource,
		RestrictModels:                ch.RestrictModels,
		GroupCount:                    len(ch.GroupIDs),
		Groups:                        st.Groups,
		Platforms:                     sortedStrategyKeys(platforms),
		ModelMappingCount:             countModelMappings(ch.ModelMapping),
		ModelPricingCount:             len(ch.ModelPricing),
		PricingModelCount:             countPricingModels(ch.ModelPricing),
		BillingModes:                  pricingBillingModes(ch.ModelPricing),
		ModelSamples:                  pricingModelSamples(ch.ModelPricing, 16),
		ApplyPricingToAccountStats:    ch.ApplyPricingToAccountStats,
		AccountStatsPricingRulesCount: len(ch.AccountStatsPricingRules),
		RequestCount:                  st.RequestCount,
		SuccessCount:                  st.SuccessCount,
		ErrorCount:                    st.ErrorCount,
		ErrorRate:                     st.ErrorRate,
		AvgDurationMs:                 st.AvgDurationMs,
		ActualCost:                    st.ActualCost,
		AccountCost:                   st.AccountCost,
		LastErrorAt:                   st.LastErrorAt,
		LastError:                     st.LastError,
	}
	if row.GroupCount == 0 {
		row.GroupCount = len(st.Groups)
	}
	return row
}

func setFromGroups(groups []ChannelStrategyGroup) map[string]struct{} {
	result := make(map[string]struct{})
	for _, g := range groups {
		if g.Platform != "" {
			result[g.Platform] = struct{}{}
		}
	}
	return result
}

func countModelMappings(mapping map[string]map[string]string) int {
	total := 0
	for _, platformMapping := range mapping {
		total += len(platformMapping)
	}
	return total
}

func countPricingModels(pricing []ChannelModelPricing) int {
	total := 0
	for _, p := range pricing {
		total += len(p.Models)
	}
	return total
}

func pricingBillingModes(pricing []ChannelModelPricing) []string {
	seen := make(map[string]struct{})
	for _, p := range pricing {
		mode := string(p.BillingMode)
		if mode == "" {
			mode = string(BillingModeToken)
		}
		seen[mode] = struct{}{}
	}
	return sortedStrategyKeys(seen)
}

func pricingModelSamples(pricing []ChannelModelPricing, limit int) []string {
	seen := make(map[string]struct{})
	for _, p := range pricing {
		for _, model := range p.Models {
			model = strings.TrimSpace(model)
			if model != "" {
				seen[model] = struct{}{}
			}
		}
	}
	models := sortedStrategyKeys(seen)
	if limit > 0 && len(models) > limit {
		return models[:limit]
	}
	return models
}

func sortedStrategyKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *ChannelService) BatchUpdateStatus(ctx context.Context, channelIDs []int64, status string) (int64, error) {
	ids, err := normalizeChannelIDs(channelIDs)
	if err != nil {
		return 0, err
	}
	if status != StatusActive && status != StatusDisabled {
		return 0, infraerrors.BadRequest("INVALID_CHANNEL_STATUS", "status must be active or disabled")
	}
	repo, err := s.strategyRepo()
	if err != nil {
		return 0, err
	}
	updated, err := repo.BatchUpdateChannelStatus(ctx, ids, status)
	if err != nil {
		return 0, fmt.Errorf("batch update channel status: %w", err)
	}
	s.invalidateCache()
	return updated, nil
}

func (s *ChannelService) BatchReplaceModelPricing(ctx context.Context, channelIDs []int64, pricing []ChannelModelPricing) error {
	ids, err := normalizeChannelIDs(channelIDs)
	if err != nil {
		return err
	}
	if len(pricing) == 0 {
		return infraerrors.BadRequest("EMPTY_MODEL_PRICING", "model_pricing must not be empty")
	}
	if err := validatePricingEntries(pricing); err != nil {
		return err
	}
	repo, err := s.strategyRepo()
	if err != nil {
		return err
	}
	if err := repo.BatchReplaceChannelModelPricing(ctx, ids, pricing); err != nil {
		return fmt.Errorf("batch replace channel model pricing: %w", err)
	}
	s.invalidateCache()
	return nil
}

func (s *ChannelService) CopyStrategy(ctx context.Context, sourceChannelID int64, targetChannelIDs []int64, opts ChannelStrategyCopyOptions) error {
	if sourceChannelID <= 0 {
		return infraerrors.BadRequest("INVALID_SOURCE_CHANNEL", "source_channel_id is required")
	}
	ids, err := normalizeChannelIDs(targetChannelIDs)
	if err != nil {
		return err
	}
	if !opts.CopyModelPricing && !opts.CopyModelMapping && !opts.CopyFlags && !opts.CopyAccountStatsPricing {
		return infraerrors.BadRequest("EMPTY_COPY_STRATEGY", "at least one copy option must be enabled")
	}
	for _, id := range ids {
		if id == sourceChannelID {
			return infraerrors.BadRequest("INVALID_TARGET_CHANNEL", "target channels must not include source channel")
		}
	}
	source, err := s.repo.GetByID(ctx, sourceChannelID)
	if err != nil {
		return fmt.Errorf("get source channel: %w", err)
	}
	if opts.CopyModelPricing {
		for i := range source.ModelPricing {
			if source.ModelPricing[i].Platform == "" {
				source.ModelPricing[i].Platform = PlatformAnthropic
			}
		}
		if err := validatePricingEntries(source.ModelPricing); err != nil {
			return err
		}
	}
	if opts.CopyModelMapping {
		if err := validateNoConflictingMappings(source.ModelMapping); err != nil {
			return err
		}
	}
	if opts.CopyAccountStatsPricing {
		for i, rule := range source.AccountStatsPricingRules {
			if err := validatePricingEntries(rule.Pricing); err != nil {
				return fmt.Errorf("account stats pricing rule #%d: %w", i+1, err)
			}
		}
	}

	repo, err := s.strategyRepo()
	if err != nil {
		return err
	}
	if err := repo.CopyChannelStrategy(ctx, sourceChannelID, ids, opts); err != nil {
		return fmt.Errorf("copy channel strategy: %w", err)
	}
	s.invalidateCache()
	return nil
}

func normalizeChannelIDs(channelIDs []int64) ([]int64, error) {
	seen := make(map[int64]struct{}, len(channelIDs))
	ids := make([]int64, 0, len(channelIDs))
	for _, id := range channelIDs {
		if id <= 0 {
			return nil, infraerrors.BadRequest("INVALID_CHANNEL_ID", "channel_ids must contain positive ids")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, infraerrors.BadRequest("EMPTY_CHANNEL_IDS", "channel_ids must not be empty")
	}
	return ids, nil
}
