package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestSortAPIKeysByCurrentConcurrency(t *testing.T) {
	keys := []APIKey{
		{ID: 1, CurrentConcurrency: 2},
		{ID: 3, CurrentConcurrency: 2},
		{ID: 2, CurrentConcurrency: 5},
	}

	sortAPIKeysByCurrentConcurrency(keys, pagination.SortOrderDesc)
	require.Equal(t, []int64{2, 3, 1}, []int64{keys[0].ID, keys[1].ID, keys[2].ID})

	sortAPIKeysByCurrentConcurrency(keys, pagination.SortOrderAsc)
	require.Equal(t, []int64{1, 3, 2}, []int64{keys[0].ID, keys[1].ID, keys[2].ID})
}

func TestPaginateAPIKeys(t *testing.T) {
	keys := []APIKey{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}}
	page := paginateAPIKeys(keys, pagination.PaginationParams{Page: 2, PageSize: 2})
	require.Equal(t, []int64{3, 4}, []int64{page[0].ID, page[1].ID})
}
