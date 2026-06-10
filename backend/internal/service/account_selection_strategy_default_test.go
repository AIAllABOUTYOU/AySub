package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func strategyTestConfig(strategy string) *config.Config {
	cfg := &config.Config{RunMode: config.RunModeStandard}
	cfg.Gateway.Scheduling.SelectionStrategy = strategy
	cfg.Gateway.Scheduling.LoadBatchEnabled = true
	cfg.Gateway.Scheduling.FallbackWaitTimeout = 30 * time.Second
	cfg.Gateway.Scheduling.FallbackMaxWaiting = 100
	cfg.Gateway.Scheduling.StickySessionWaitTimeout = 45 * time.Second
	cfg.Gateway.Scheduling.StickySessionMaxWaiting = 3
	return cfg
}

func strategyTestAccount(id int64, priority int) *Account {
	return &Account{
		ID:          id,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    priority,
	}
}

func openAIStrategyTestAccount(id int64, priority int) Account {
	return Account{
		ID:          id,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    priority,
	}
}

func TestAccountSelectionCursorStore_RoundRobinOrder(t *testing.T) {
	store := newAccountSelectionCursorStore()
	accounts := []*Account{
		strategyTestAccount(30, 2),
		strategyTestAccount(20, 1),
		strategyTestAccount(10, 1),
	}

	require.Equal(t, []int64{10, 20, 30}, accountIDs(store.order("same-key", accounts)))
	require.Equal(t, []int64{20, 30, 10}, accountIDs(store.order("same-key", accounts)))
	require.Equal(t, []int64{30, 10, 20}, accountIDs(store.order("same-key", accounts)))
	require.Equal(t, []int64{10, 20, 30}, accountIDs(store.order("other-key", accounts)))
}

func TestGatewaySelectionStrategy_RoundRobinAndFillFirst(t *testing.T) {
	ctx := context.Background()
	candidates := []*Account{
		strategyTestAccount(30, 2),
		strategyTestAccount(20, 1),
		strategyTestAccount(10, 1),
	}

	t.Run("round robin rotates in priority id order", func(t *testing.T) {
		svc := &GatewayService{
			concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
			selectionCursors:   newAccountSelectionCursorStore(),
		}

		var selected []int64
		for i := 0; i < 3; i++ {
			selection, ok, err := svc.tryAcquireBySelectionStrategy(ctx, candidates, nil, "", false, AccountSelectionStrategyRoundRobin, "gateway-test")
			require.NoError(t, err)
			require.True(t, ok)
			require.NotNil(t, selection)
			require.NotNil(t, selection.Account)
			selected = append(selected, selection.Account.ID)
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
		}
		require.Equal(t, []int64{10, 20, 30}, selected)
	})

	t.Run("fill first skips busy accounts", func(t *testing.T) {
		svc := &GatewayService{
			concurrencyService: NewConcurrencyService(stubConcurrencyCache{
				acquireResults: map[int64]bool{10: false},
			}),
		}

		selection, ok, err := svc.tryAcquireBySelectionStrategy(ctx, candidates, nil, "", false, AccountSelectionStrategyFillFirst, "unused")
		require.NoError(t, err)
		require.True(t, ok)
		require.NotNil(t, selection)
		require.NotNil(t, selection.Account)
		require.True(t, selection.Acquired)
		require.Equal(t, int64(20), selection.Account.ID)
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
	})

	t.Run("fill first returns wait plan for first ordered account when full", func(t *testing.T) {
		svc := &GatewayService{
			cfg: strategyTestConfig(string(AccountSelectionStrategyFillFirst)),
			concurrencyService: NewConcurrencyService(stubConcurrencyCache{
				acquireResults: map[int64]bool{10: false, 20: false, 30: false},
			}),
		}

		selection, ok, err := svc.tryAcquireBySelectionStrategy(ctx, candidates, nil, "", false, AccountSelectionStrategyFillFirst, "unused")
		require.NoError(t, err)
		require.True(t, ok)
		require.NotNil(t, selection)
		require.False(t, selection.Acquired)
		require.NotNil(t, selection.WaitPlan)
		require.Equal(t, int64(10), selection.WaitPlan.AccountID)
	})
}

func TestOpenAISelectionStrategy_RoundRobinSkipsExcludedAndCooling(t *testing.T) {
	now := time.Now()
	coolingUntil := now.Add(time.Minute)
	accounts := []Account{
		openAIStrategyTestAccount(10, 1),
		openAIStrategyTestAccount(20, 1),
		openAIStrategyTestAccount(30, 2),
	}
	accounts[0].RateLimitResetAt = &coolingUntil

	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: accounts},
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
		cfg:                strategyTestConfig(string(AccountSelectionStrategyRoundRobin)),
		selectionCursors:   newAccountSelectionCursorStore(),
	}

	selection, err := svc.SelectAccountWithLoadAwareness(context.Background(), nil, "", "gpt-4", map[int64]struct{}{20: {}})
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(30), selection.Account.ID)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}

	selection, err = svc.SelectAccountWithLoadAwareness(context.Background(), nil, "", "gpt-4", nil)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(20), selection.Account.ID)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAISelectionStrategy_FillFirstSkipsBusyAccount(t *testing.T) {
	accounts := []Account{
		openAIStrategyTestAccount(30, 2),
		openAIStrategyTestAccount(20, 1),
		openAIStrategyTestAccount(10, 1),
	}
	svc := &OpenAIGatewayService{
		accountRepo: stubOpenAIAccountRepo{accounts: accounts},
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{
			acquireResults: map[int64]bool{10: false},
		}),
		cfg: strategyTestConfig(string(AccountSelectionStrategyFillFirst)),
	}

	selection, err := svc.SelectAccountWithLoadAwareness(context.Background(), nil, "", "gpt-4", nil)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.True(t, selection.Acquired)
	require.Equal(t, int64(20), selection.Account.ID)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func accountIDs(accounts []*Account) []int64 {
	ids := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		if account != nil {
			ids = append(ids, account.ID)
		}
	}
	return ids
}
