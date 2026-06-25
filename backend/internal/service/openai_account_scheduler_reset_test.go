//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func openAIResetTestScheduler(reset float64) *defaultOpenAIAccountScheduler {
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights = config.GatewayOpenAIWSSchedulerScoreWeights{
		Priority:  1.0,
		Load:      1.0,
		Queue:     0.7,
		ErrorRate: 0.8,
		TTFT:      0.5,
		Reset:     reset,
	}
	return &defaultOpenAIAccountScheduler{service: &OpenAIGatewayService{cfg: cfg}}
}

func openAIPlanScores(plan openAIAccountLoadPlan) map[int64]float64 {
	scores := make(map[int64]float64, len(plan.candidates))
	for _, c := range plan.candidates {
		scores[c.account.ID] = c.score
	}
	return scores
}

func TestBuildOpenAIAccountLoadPlan_ResetWeightPrefersSoonestReset(t *testing.T) {
	now := time.Now()
	soon := now.Add(1 * time.Hour)
	later := now.Add(20 * time.Hour)
	filtered := []*Account{
		{ID: 1, Priority: 0, SessionWindowEnd: &later},
		{ID: 2, Priority: 0, SessionWindowEnd: &soon},
	}
	sched := openAIResetTestScheduler(5.0)

	plan := sched.buildOpenAIAccountLoadPlan(OpenAIAccountScheduleRequest{}, filtered, map[int64]*AccountLoadInfo{})
	scores := openAIPlanScores(plan)

	require.Greater(t, scores[2], scores[1])
}

func TestBuildOpenAIAccountLoadPlan_ResetWeightZeroNoEffect(t *testing.T) {
	now := time.Now()
	soon := now.Add(1 * time.Hour)
	later := now.Add(20 * time.Hour)
	filtered := []*Account{
		{ID: 1, Priority: 0, SessionWindowEnd: &later},
		{ID: 2, Priority: 0, SessionWindowEnd: &soon},
	}
	sched := openAIResetTestScheduler(0.0)

	plan := sched.buildOpenAIAccountLoadPlan(OpenAIAccountScheduleRequest{}, filtered, map[int64]*AccountLoadInfo{})
	scores := openAIPlanScores(plan)

	require.Equal(t, scores[1], scores[2])
}

func TestBuildOpenAIAccountLoadPlan_ResetWeightIgnoresNilWindow(t *testing.T) {
	now := time.Now()
	soon := now.Add(2 * time.Hour)
	filtered := []*Account{
		{ID: 1, Priority: 0, SessionWindowEnd: nil},
		{ID: 2, Priority: 0, SessionWindowEnd: &soon},
	}
	sched := openAIResetTestScheduler(5.0)

	plan := sched.buildOpenAIAccountLoadPlan(OpenAIAccountScheduleRequest{}, filtered, map[int64]*AccountLoadInfo{})
	scores := openAIPlanScores(plan)

	require.Greater(t, scores[2], scores[1])
}
