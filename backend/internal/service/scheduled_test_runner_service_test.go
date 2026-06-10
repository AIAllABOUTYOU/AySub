package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectScheduledTestPlansSkipsDuplicateAccounts(t *testing.T) {
	plans := []*ScheduledTestPlan{
		{ID: 1, AccountID: 10},
		{ID: 2, AccountID: 20},
		{ID: 3, AccountID: 10},
		{ID: 4, AccountID: 20},
		{ID: 5, AccountID: 30},
	}

	runnable, skipped := selectScheduledTestPlans(plans)

	require.Equal(t, []*ScheduledTestPlan{plans[0], plans[1], plans[4]}, runnable)
	require.Equal(t, []*ScheduledTestPlan{plans[2], plans[3]}, skipped)
}
