package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type grokConsole429Repo struct {
	AccountRepository
	account           *Account
	updateExtraCalls  []map[string]any
	tempUnschedID     int64
	tempUnschedUntil  time.Time
	tempUnschedReason string
}

func (r *grokConsole429Repo) GetByID(_ context.Context, id int64) (*Account, error) {
	if r.account != nil && r.account.ID == id {
		return r.account, nil
	}
	return nil, ErrAccountNotFound
}

func (r *grokConsole429Repo) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	copied := make(map[string]any, len(updates))
	for key, value := range updates {
		copied[key] = value
	}
	r.updateExtraCalls = append(r.updateExtraCalls, copied)
	if r.account != nil {
		if r.account.Extra == nil {
			r.account.Extra = map[string]any{}
		}
		for key, value := range updates {
			r.account.Extra[key] = value
		}
	}
	return nil
}

func (r *grokConsole429Repo) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	r.tempUnschedID = id
	r.tempUnschedUntil = until
	r.tempUnschedReason = reason
	return nil
}

func TestRecordGrokConsole429TriggersTempUnschedulableAtThreshold(t *testing.T) {
	now := time.Now().UTC()
	account := &Account{
		ID:       9001,
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Extra: map[string]any{
			grokConsole429CountKey:  2,
			grokConsole429LastAtKey: now.Add(-30 * time.Minute).Unix(),
		},
	}
	repo := &grokConsole429Repo{account: account}
	svc := &OpenAIGatewayService{
		accountRepo:      repo,
		rateLimitService: &RateLimitService{accountRepo: repo},
	}

	before := time.Now().UTC()
	svc.recordGrokConsole429(context.Background(), account, "grok-4.3", []byte(`{"error":{"message":"rate limited"}}`))
	after := time.Now().UTC()

	if repo.tempUnschedID != account.ID {
		t.Fatalf("tempUnschedID = %d, want %d", repo.tempUnschedID, account.ID)
	}
	if repo.tempUnschedUntil.Before(before.Add(grokConsole429ThresholdCooldown)) || repo.tempUnschedUntil.After(after.Add(grokConsole429ThresholdCooldown)) {
		t.Fatalf("tempUnschedUntil = %v, want about +%s", repo.tempUnschedUntil, grokConsole429ThresholdCooldown)
	}
	if len(repo.updateExtraCalls) != 1 {
		t.Fatalf("updateExtraCalls = %d, want 1", len(repo.updateExtraCalls))
	}
	updates := repo.updateExtraCalls[0]
	if updates[grokConsole429CountKey] != 3 {
		t.Fatalf("count = %v, want 3", updates[grokConsole429CountKey])
	}
	if updates[grokConsole429ReasonKey] != grokConsole429ThresholdReason {
		t.Fatalf("reason = %v, want %s", updates[grokConsole429ReasonKey], grokConsole429ThresholdReason)
	}
	if updates[grokConsole429LastModelKey] != "grok-4.3" {
		t.Fatalf("last model = %v", updates[grokConsole429LastModelKey])
	}

	var state TempUnschedState
	if err := json.Unmarshal([]byte(repo.tempUnschedReason), &state); err != nil {
		t.Fatalf("temp reason should be JSON: %v", err)
	}
	if state.StatusCode != 429 || state.MatchedKeyword != grokConsole429ThresholdReason {
		t.Fatalf("state = %+v", state)
	}
}

func TestRecordGrokConsole429ResetsOutsideSlidingWindow(t *testing.T) {
	account := &Account{
		ID:       9002,
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Extra: map[string]any{
			grokConsole429CountKey:  2,
			grokConsole429LastAtKey: time.Now().Add(-13 * time.Hour).Unix(),
		},
	}
	repo := &grokConsole429Repo{account: account}
	svc := &OpenAIGatewayService{accountRepo: repo}

	svc.recordGrokConsole429(context.Background(), account, "grok-4.3", nil)

	if repo.tempUnschedID != 0 {
		t.Fatalf("tempUnschedID = %d, want 0", repo.tempUnschedID)
	}
	if len(repo.updateExtraCalls) != 1 {
		t.Fatalf("updateExtraCalls = %d, want 1", len(repo.updateExtraCalls))
	}
	if repo.updateExtraCalls[0][grokConsole429CountKey] != 1 {
		t.Fatalf("count = %v, want 1", repo.updateExtraCalls[0][grokConsole429CountKey])
	}
}
