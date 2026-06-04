package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrPublicStatusDisabled = infraerrors.NotFound("PUBLIC_STATUS_DISABLED", "public status page is disabled")

type PublicStatusService struct {
	settingService   *SettingService
	channelService   *ChannelService
	dashboardService *DashboardService
	opsService       *OpsService
}

func NewPublicStatusService(settingService *SettingService, channelService *ChannelService, dashboardService *DashboardService, opsService *OpsService) *PublicStatusService {
	return &PublicStatusService{
		settingService:   settingService,
		channelService:   channelService,
		dashboardService: dashboardService,
		opsService:       opsService,
	}
}

type PublicStatusResponse struct {
	Enabled      bool                 `json:"enabled"`
	Status       string               `json:"status"`
	GeneratedAt  time.Time            `json:"generated_at"`
	Models       PublicStatusModels   `json:"models"`
	Channels     PublicStatusChannels `json:"channels"`
	Last24h      PublicStatusLast24h  `json:"last_24h"`
	RecentEvents []PublicStatusEvent  `json:"recent_events"`
}

type PublicStatusModels struct {
	Visible bool     `json:"visible"`
	Count   int      `json:"count"`
	Names   []string `json:"names,omitempty"`
}

type PublicStatusChannels struct {
	Visible         bool                         `json:"visible"`
	Total           int                          `json:"total"`
	Active          int                          `json:"active"`
	DisabledOrError int                          `json:"disabled_or_error"`
	Summaries       []PublicStatusChannelSummary `json:"summaries,omitempty"`
}

type PublicStatusChannelSummary struct {
	Platform   string `json:"platform"`
	Total      int    `json:"total"`
	Active     int    `json:"active"`
	ModelCount int    `json:"model_count"`
}

type PublicStatusLast24h struct {
	Requests  int64                    `json:"requests"`
	ErrorRate float64                  `json:"error_rate"`
	LatencyMs PublicStatusLatencyRange `json:"latency_ms"`
}

type PublicStatusLatencyRange struct {
	Avg          float64 `json:"avg"`
	MinBucketAvg float64 `json:"min_bucket_avg"`
	MaxBucketAvg float64 `json:"max_bucket_avg"`
}

type PublicStatusEvent struct {
	CreatedAt  time.Time `json:"created_at"`
	Severity   string    `json:"severity"`
	Summary    string    `json:"summary"`
	Endpoint   string    `json:"endpoint,omitempty"`
	StatusCode *int      `json:"status_code,omitempty"`
}

func (s *PublicStatusService) GetPublicStatus(ctx context.Context) (*PublicStatusResponse, error) {
	runtime := PublicStatusRuntime{}
	if s != nil && s.settingService != nil {
		runtime = s.settingService.GetPublicStatusRuntime(ctx)
	}
	if !runtime.Enabled {
		return nil, ErrPublicStatusDisabled
	}

	now := time.Now().UTC()
	resp := &PublicStatusResponse{
		Enabled:     true,
		Status:      "operational",
		GeneratedAt: now,
		Models: PublicStatusModels{
			Visible: runtime.ShowModels,
		},
		Channels: PublicStatusChannels{
			Visible: runtime.ShowChannels,
		},
		RecentEvents: []PublicStatusEvent{},
	}

	degraded := false
	channels, err := s.safeListAvailableChannels(ctx)
	if err != nil {
		degraded = true
	} else {
		if runtime.ShowModels {
			resp.Models = publicStatusModelsFromChannels(channels)
		}
		if runtime.ShowChannels {
			resp.Channels = publicStatusChannelsFromChannels(channels)
			if resp.Channels.Total > 0 && resp.Channels.Active == 0 {
				degraded = true
			}
		}
	}

	last24h, reportErr := s.publicLast24h(ctx, now.Add(-24*time.Hour), now)
	if reportErr != nil {
		degraded = true
	} else {
		resp.Last24h = last24h
		if last24h.Requests > 0 && last24h.ErrorRate >= 0.2 {
			degraded = true
		}
	}

	if runtime.ShowRecentIncidents {
		events, eventErr := s.publicRecentEvents(ctx, now.Add(-24*time.Hour), now)
		if eventErr != nil && !errors.Is(eventErr, ErrOpsDisabled) {
			degraded = true
		}
		resp.RecentEvents = events
	}

	if degraded {
		resp.Status = "degraded"
	}
	return resp, nil
}

func (s *PublicStatusService) safeListAvailableChannels(ctx context.Context) ([]AvailableChannel, error) {
	if s == nil || s.channelService == nil {
		return nil, fmt.Errorf("channel service unavailable")
	}
	return s.channelService.ListAvailable(ctx)
}

func publicStatusModelsFromChannels(channels []AvailableChannel) PublicStatusModels {
	seen := make(map[string]struct{})
	for _, ch := range channels {
		if ch.Status != StatusActive {
			continue
		}
		for _, m := range ch.SupportedModels {
			name := strings.TrimSpace(m.Name)
			if name != "" {
				seen[name] = struct{}{}
			}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return PublicStatusModels{Visible: true, Count: len(names), Names: names}
}

func publicStatusChannelsFromChannels(channels []AvailableChannel) PublicStatusChannels {
	out := PublicStatusChannels{Visible: true, Total: len(channels)}
	byPlatform := make(map[string]*PublicStatusChannelSummary)
	for _, ch := range channels {
		if ch.Status == StatusActive {
			out.Active++
		} else {
			out.DisabledOrError++
		}
		platforms := channelPublicPlatforms(ch)
		for _, platform := range platforms {
			summary := byPlatform[platform]
			if summary == nil {
				summary = &PublicStatusChannelSummary{Platform: platform}
				byPlatform[platform] = summary
			}
			summary.Total++
			if ch.Status == StatusActive {
				summary.Active++
			}
			summary.ModelCount += countChannelModelsForPlatform(ch, platform)
		}
	}
	out.Summaries = make([]PublicStatusChannelSummary, 0, len(byPlatform))
	for _, summary := range byPlatform {
		out.Summaries = append(out.Summaries, *summary)
	}
	sort.Slice(out.Summaries, func(i, j int) bool {
		return out.Summaries[i].Platform < out.Summaries[j].Platform
	})
	return out
}

func channelPublicPlatforms(ch AvailableChannel) []string {
	seen := make(map[string]struct{})
	for _, g := range ch.Groups {
		platform := strings.TrimSpace(g.Platform)
		if platform != "" {
			seen[platform] = struct{}{}
		}
	}
	for _, m := range ch.SupportedModels {
		platform := strings.TrimSpace(m.Platform)
		if platform != "" {
			seen[platform] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return []string{"unknown"}
	}
	out := make([]string, 0, len(seen))
	for platform := range seen {
		out = append(out, platform)
	}
	sort.Strings(out)
	return out
}

func countChannelModelsForPlatform(ch AvailableChannel, platform string) int {
	seen := make(map[string]struct{})
	for _, m := range ch.SupportedModels {
		if strings.TrimSpace(m.Platform) != platform {
			continue
		}
		name := strings.TrimSpace(m.Name)
		if name != "" {
			seen[name] = struct{}{}
		}
	}
	return len(seen)
}

func (s *PublicStatusService) publicLast24h(ctx context.Context, start, end time.Time) (PublicStatusLast24h, error) {
	var out PublicStatusLast24h
	if s == nil || s.dashboardService == nil {
		return out, fmt.Errorf("dashboard service unavailable")
	}
	reports, err := s.dashboardService.GetOperationalReports(ctx, start, end, "hour", 10)
	if err != nil {
		return out, err
	}
	var errorsTotal int64
	var weightedDuration float64
	var durationWeight int64
	for _, point := range reports.RequestTrend {
		out.Requests += point.TotalRequests
		errorsTotal += point.ErrorCount
		if point.TotalRequests > 0 && point.AvgDurationMs > 0 {
			weightedDuration += point.AvgDurationMs * float64(point.TotalRequests)
			durationWeight += point.TotalRequests
			if out.LatencyMs.MinBucketAvg == 0 || point.AvgDurationMs < out.LatencyMs.MinBucketAvg {
				out.LatencyMs.MinBucketAvg = point.AvgDurationMs
			}
			if point.AvgDurationMs > out.LatencyMs.MaxBucketAvg {
				out.LatencyMs.MaxBucketAvg = point.AvgDurationMs
			}
		}
	}
	if out.Requests > 0 {
		out.ErrorRate = float64(errorsTotal) / float64(out.Requests)
	}
	if durationWeight > 0 {
		out.LatencyMs.Avg = weightedDuration / float64(durationWeight)
	}
	return out, nil
}

func (s *PublicStatusService) publicRecentEvents(ctx context.Context, start, end time.Time) ([]PublicStatusEvent, error) {
	if s == nil || s.opsService == nil {
		return []PublicStatusEvent{}, nil
	}
	list, err := s.opsService.ListRequestDetails(ctx, &OpsRequestDetailFilter{
		StartTime: &start,
		EndTime:   &end,
		Kind:      string(OpsRequestKindError),
		Page:      1,
		PageSize:  5,
		Sort:      "created_at_desc",
	})
	if err != nil {
		return []PublicStatusEvent{}, err
	}
	events := make([]PublicStatusEvent, 0, len(list.Items))
	for _, item := range list.Items {
		if item == nil {
			continue
		}
		endpoint := firstNonEmpty(item.InboundEndpoint, item.RequestPath)
		events = append(events, PublicStatusEvent{
			CreatedAt:  item.CreatedAt,
			Severity:   publicStatusSeverity(item.Severity, item.StatusCode),
			Summary:    publicStatusEventSummary(item.StatusCode, endpoint, item.ErrorCode),
			Endpoint:   endpoint,
			StatusCode: item.StatusCode,
		})
	}
	return events, nil
}

func publicStatusSeverity(severity string, statusCode *int) string {
	severity = strings.ToLower(strings.TrimSpace(severity))
	if severity == "critical" || severity == "error" || severity == "warning" || severity == "info" {
		return severity
	}
	if statusCode != nil && *statusCode >= 500 {
		return "error"
	}
	return "warning"
}

func publicStatusEventSummary(statusCode *int, endpoint, errorCode string) string {
	parts := make([]string, 0, 3)
	if statusCode != nil && *statusCode > 0 {
		parts = append(parts, fmt.Sprintf("HTTP %d", *statusCode))
	}
	if endpoint != "" {
		parts = append(parts, endpoint)
	}
	if code := strings.TrimSpace(errorCode); code != "" {
		parts = append(parts, code)
	}
	if len(parts) == 0 {
		return "request error"
	}
	return strings.Join(parts, " · ")
}
