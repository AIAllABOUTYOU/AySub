package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryGetUserBreakdownStats_AggregatesTokenColumns(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := newUsageLogRepositoryWithSQL(nil, db)
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	rows := sqlmock.NewRows([]string{
		"user_id", "email", "requests", "input_tokens", "output_tokens", "cache_tokens",
		"total_tokens", "cost", "actual_cost", "account_cost",
	}).AddRow(int64(7), "alice@example.com", int64(3), int64(120), int64(30), int64(50), int64(200), 1.5, 1.2, 0.9)

	mock.ExpectQuery(`(?s)COALESCE\(SUM\(ul\.input_tokens\), 0\) as input_tokens,.*COALESCE\(SUM\(ul\.output_tokens\), 0\) as output_tokens,.*COALESCE\(SUM\(ul\.cache_creation_tokens \+ ul\.cache_read_tokens\), 0\) as cache_tokens,.*ORDER BY actual_cost DESC LIMIT 50`).
		WithArgs(start, end).
		WillReturnRows(rows)

	result, err := repo.GetUserBreakdownStats(context.Background(), start, end, usagestats.UserBreakdownDimension{}, 50)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, int64(120), result[0].InputTokens)
	require.Equal(t, int64(30), result[0].OutputTokens)
	require.Equal(t, int64(50), result[0].CacheTokens)
	require.Equal(t, int64(200), result[0].TotalTokens)
	require.InDelta(t, 1.5, result[0].Cost, 0.0001)
	require.InDelta(t, 1.2, result[0].ActualCost, 0.0001)
	require.InDelta(t, 0.9, result[0].AccountCost, 0.0001)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetUserBreakdownStats_SortAllowlist(t *testing.T) {
	tests := []struct {
		name       string
		sortBy     string
		wantColumn string
	}{
		{name: "requests", sortBy: "requests", wantColumn: "requests"},
		{name: "input tokens", sortBy: "input_tokens", wantColumn: "input_tokens"},
		{name: "output tokens", sortBy: "output_tokens", wantColumn: "output_tokens"},
		{name: "cache tokens", sortBy: "cache_tokens", wantColumn: "cache_tokens"},
		{name: "total tokens", sortBy: "total_tokens", wantColumn: "total_tokens"},
		{name: "cost", sortBy: "cost", wantColumn: "cost"},
		{name: "actual cost", sortBy: "actual_cost", wantColumn: "actual_cost"},
		{name: "empty fallback", sortBy: "", wantColumn: "actual_cost"},
		{name: "invalid fallback", sortBy: "total_tokens; DROP TABLE users", wantColumn: "actual_cost"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			repo := newUsageLogRepositoryWithSQL(nil, db)
			start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
			end := start.Add(time.Hour)

			mock.ExpectQuery(fmt.Sprintf(`(?s)GROUP BY ul\.user_id, u\.email ORDER BY %s DESC LIMIT 20`, tc.wantColumn)).
				WithArgs(start, end).
				WillReturnRows(sqlmock.NewRows([]string{
					"user_id", "email", "requests", "input_tokens", "output_tokens", "cache_tokens",
					"total_tokens", "cost", "actual_cost", "account_cost",
				}))

			result, err := repo.GetUserBreakdownStats(context.Background(), start, end, usagestats.UserBreakdownDimension{SortBy: tc.sortBy}, 20)
			require.NoError(t, err)
			require.Empty(t, result)
			require.Equal(t, tc.wantColumn, resolveUserBreakdownSortColumn(tc.sortBy))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
