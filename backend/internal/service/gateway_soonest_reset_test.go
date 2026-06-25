//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func accountWithWindowEndForTest(id int64, end *time.Time) accountWithLoad {
	return accountWithLoad{
		account: &Account{
			ID:               id,
			Schedulable:      true,
			Status:           StatusActive,
			SessionWindowEnd: end,
		},
		loadInfo: &AccountLoadInfo{AccountID: id},
	}
}

func TestFilterBySoonestReset_PicksSoonestFutureWindow(t *testing.T) {
	now := time.Now()
	soon := now.Add(1 * time.Hour)
	later := now.Add(24 * time.Hour)
	accounts := []accountWithLoad{
		accountWithWindowEndForTest(1, testTimePtr(later)),
		accountWithWindowEndForTest(2, testTimePtr(soon)),
		accountWithWindowEndForTest(3, testTimePtr(later)),
	}

	got := filterBySoonestReset(accounts)

	require.Len(t, got, 1)
	require.Equal(t, int64(2), got[0].account.ID)
}

func TestFilterBySoonestReset_IgnoresNilAndExpiredWindows(t *testing.T) {
	now := time.Now()
	expired := now.Add(-1 * time.Hour)
	active := now.Add(2 * time.Hour)
	accounts := []accountWithLoad{
		accountWithWindowEndForTest(1, nil),
		accountWithWindowEndForTest(2, testTimePtr(expired)),
		accountWithWindowEndForTest(3, testTimePtr(active)),
	}

	got := filterBySoonestReset(accounts)

	require.Len(t, got, 1)
	require.Equal(t, int64(3), got[0].account.ID)
}

func TestFilterBySoonestReset_NoActiveWindowReturnsAll(t *testing.T) {
	now := time.Now()
	expired := now.Add(-30 * time.Minute)
	accounts := []accountWithLoad{
		accountWithWindowEndForTest(1, nil),
		accountWithWindowEndForTest(2, testTimePtr(expired)),
	}

	got := filterBySoonestReset(accounts)

	require.Len(t, got, 2)
}

func TestFilterBySoonestReset_TiedSoonestKeepsAll(t *testing.T) {
	now := time.Now()
	end := now.Add(90 * time.Minute)
	accounts := []accountWithLoad{
		accountWithWindowEndForTest(1, testTimePtr(end)),
		accountWithWindowEndForTest(2, testTimePtr(end)),
		accountWithWindowEndForTest(3, testTimePtr(now.Add(5*time.Hour))),
	}

	got := filterBySoonestReset(accounts)

	require.Len(t, got, 2)
	ids := map[int64]bool{got[0].account.ID: true, got[1].account.ID: true}
	require.True(t, ids[1] && ids[2])
}

func TestFilterBySoonestReset_SingleOrEmptyUnchanged(t *testing.T) {
	require.Empty(t, filterBySoonestReset(nil))
	single := []accountWithLoad{accountWithWindowEndForTest(1, nil)}
	require.Len(t, filterBySoonestReset(single), 1)
}
