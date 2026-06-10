package service

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type AccountSelectionStrategy string

const (
	AccountSelectionStrategyLoadAware  AccountSelectionStrategy = "load-aware"
	AccountSelectionStrategyRoundRobin AccountSelectionStrategy = "round-robin"
	AccountSelectionStrategyFillFirst  AccountSelectionStrategy = "fill-first"
)

func normalizeAccountSelectionStrategy(value string) AccountSelectionStrategy {
	switch AccountSelectionStrategy(strings.ToLower(strings.TrimSpace(value))) {
	case AccountSelectionStrategyRoundRobin:
		return AccountSelectionStrategyRoundRobin
	case AccountSelectionStrategyFillFirst:
		return AccountSelectionStrategyFillFirst
	default:
		return AccountSelectionStrategyLoadAware
	}
}

type accountSelectionCursorStore struct {
	mu      sync.Mutex
	cursors map[string]int
}

func newAccountSelectionCursorStore() *accountSelectionCursorStore {
	return &accountSelectionCursorStore{cursors: make(map[string]int)}
}

func (s *accountSelectionCursorStore) order(key string, accounts []*Account) []*Account {
	ordered := sortAccountsByPriorityAndID(accounts)
	if len(ordered) <= 1 {
		return ordered
	}
	if s == nil {
		return ordered
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cursors == nil {
		s.cursors = make(map[string]int)
	}
	idx := s.cursors[key] % len(ordered)
	s.cursors[key] = (idx + 1) % len(ordered)
	if idx == 0 {
		return ordered
	}
	rotated := make([]*Account, 0, len(ordered))
	rotated = append(rotated, ordered[idx:]...)
	rotated = append(rotated, ordered[:idx]...)
	return rotated
}

func sortAccountsByPriorityAndID(accounts []*Account) []*Account {
	ordered := append([]*Account(nil), accounts...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		return a.ID < b.ID
	})
	return ordered
}

func accountSelectionKey(parts ...any) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(fmt.Sprint(part)))
	}
	return strings.Join(out, "|")
}
