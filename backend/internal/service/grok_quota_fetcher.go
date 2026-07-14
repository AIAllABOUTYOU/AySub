package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

const grokQuotaSnapshotExtraKey = "grok_usage_snapshot"

type GrokQuotaFetcher struct{}

func NewGrokQuotaFetcher() *GrokQuotaFetcher {
	return &GrokQuotaFetcher{}
}

func (f *GrokQuotaFetcher) BuildUsageInfo(account *Account) *UsageInfo {
	now := time.Now()
	usage := &UsageInfo{
		Source:    "passive",
		UpdatedAt: &now,
	}
	if account == nil {
		usage.ErrorCode = "quota_unknown"
		usage.Error = "Grok quota is unknown until the first upstream response includes xAI rate-limit headers"
		return usage
	}

	snapshot, err := grokQuotaSnapshotFromExtra(account.Extra)
	if err != nil || snapshot == nil {
		usage.ErrorCode = "quota_unknown"
		usage.Error = "Grok quota is unknown until the first upstream response includes xAI rate-limit headers"
		return usage
	}

	if parsedAt, err := time.Parse(time.RFC3339, snapshot.UpdatedAt); err == nil {
		usage.UpdatedAt = &parsedAt
	}
	usage.GrokQuota = grokAPIQuotaProjection(snapshot)
	usage.SubscriptionTier = snapshot.SubscriptionTier
	usage.SubscriptionTierRaw = snapshot.SubscriptionTier
	if !snapshot.HasObservedHeaders() {
		usage.ErrorCode = "quota_unknown"
		usage.Error = "No xAI quota headers observed on the latest Grok probe"
	}

	switch snapshot.StatusCode {
	case 401:
		usage.NeedsReauth = true
		usage.ErrorCode = "unauthenticated"
	case 403:
		usage.IsForbidden = true
		usage.ForbiddenType = "forbidden"
		usage.ErrorCode = "forbidden"
	case 429:
		usage.ErrorCode = "rate_limited"
	}
	return usage
}

func grokAPIQuotaProjection(snapshot *xai.QuotaSnapshot) map[string]*GrokModelQuota {
	if snapshot == nil || snapshot.Requests == nil {
		return nil
	}
	quota := &GrokModelQuota{}
	if snapshot.Requests.Limit != nil {
		quota.TotalQueries = int(*snapshot.Requests.Limit)
	}
	if snapshot.Requests.Remaining != nil {
		quota.RemainingQueries = int(*snapshot.Requests.Remaining)
	}
	if quota.TotalQueries > 0 {
		used := quota.TotalQueries - quota.RemainingQueries
		if used < 0 {
			used = 0
		}
		quota.Utilization = used * 100 / quota.TotalQueries
	}
	quota.ResetTime = snapshot.Requests.ResetAt
	return map[string]*GrokModelQuota{"xai_api": quota}
}

func grokQuotaSnapshotFromExtra(extra map[string]any) (*xai.QuotaSnapshot, error) {
	if extra == nil {
		return nil, nil
	}
	raw, ok := extra[grokQuotaSnapshotExtraKey]
	if !ok || raw == nil {
		return nil, nil
	}
	switch snapshot := raw.(type) {
	case *xai.QuotaSnapshot:
		return snapshot, nil
	case xai.QuotaSnapshot:
		return &snapshot, nil
	case map[string]any:
		data, err := json.Marshal(snapshot)
		if err != nil {
			return nil, err
		}
		var out xai.QuotaSnapshot
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return &out, nil
	default:
		data, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("marshal grok quota snapshot: %w", err)
		}
		var out xai.QuotaSnapshot
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}
}
