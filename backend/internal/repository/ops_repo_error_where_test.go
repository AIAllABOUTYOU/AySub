package repository

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestBuildOpsErrorLogsWhere_QueryUsesQualifiedColumns(t *testing.T) {
	filter := &service.OpsErrorLogFilter{
		Query: "ACCESS_DENIED",
	}

	where, args := buildOpsErrorLogsWhere(filter)
	if where == "" {
		t.Fatalf("where should not be empty")
	}
	if len(args) != 1 {
		t.Fatalf("args len = %d, want 1", len(args))
	}
	if !strings.Contains(where, "e.request_id ILIKE $") {
		t.Fatalf("where should include qualified request_id condition: %s", where)
	}
	if !strings.Contains(where, "e.client_request_id ILIKE $") {
		t.Fatalf("where should include qualified client_request_id condition: %s", where)
	}
	if !strings.Contains(where, "e.error_message ILIKE $") {
		t.Fatalf("where should include qualified error_message condition: %s", where)
	}
}

func TestBuildOpsErrorLogsWhere_AdminUsageFilters(t *testing.T) {
	userID := int64(11)
	apiKeyID := int64(12)
	accountID := int64(13)
	groupID := int64(14)
	requestType := int16(service.RequestTypeWSV2)
	filter := &service.OpsErrorLogFilter{
		UserID:         &userID,
		APIKeyID:       &apiKeyID,
		AccountID:      &accountID,
		GroupID:        &groupID,
		Model:          "gpt-5.6",
		RequestType:    &requestType,
		ErrorPhasesAny: []string{"upstream", "network"},
		ErrorTypesAny:  []string{"rate_limit_error"},
		View:           "all",
	}

	where, args := buildOpsErrorLogsWhere(filter)
	for _, clause := range []string{
		"e.group_id = $",
		"e.account_id = $",
		"e.user_id = $",
		"e.api_key_id = $",
		"COALESCE(NULLIF(TRIM(e.requested_model), ''), e.model, '') = $",
		"e.request_type = $",
		"e.error_phase = ANY($",
		"e.error_type = ANY($",
	} {
		if !strings.Contains(where, clause) {
			t.Fatalf("where missing %q: %s", clause, where)
		}
	}
	if len(args) != 8 {
		t.Fatalf("args len = %d, want 8: %#v", len(args), args)
	}
	if !reflect.DeepEqual(args[:6], []any{groupID, accountID, userID, apiKeyID, "gpt-5.6", requestType}) {
		t.Fatalf("unexpected scalar args: %#v", args[:6])
	}
}

func TestBuildOpsErrorLogsWhere_UpstreamPhaseNeedsExplicitRecoveredOptIn(t *testing.T) {
	requestWhere, _ := buildOpsErrorLogsWhere(&service.OpsErrorLogFilter{Phase: "upstream", View: "all"})
	if !strings.Contains(requestWhere, "COALESCE(e.status_code, 0) >= 400") {
		t.Fatalf("request error filter must keep client status guard: %s", requestWhere)
	}

	upstreamWhere, _ := buildOpsErrorLogsWhere(&service.OpsErrorLogFilter{
		Phase:                    "upstream",
		IncludeRecoveredUpstream: true,
		View:                     "all",
	})
	if strings.Contains(upstreamWhere, "COALESCE(e.status_code, 0) >= 400") {
		t.Fatalf("dedicated upstream filter should include recovered rows: %s", upstreamWhere)
	}
}

func TestOpsErrorLogsOrderBy_WhitelistAndFallback(t *testing.T) {
	tests := []struct {
		name   string
		filter *service.OpsErrorLogFilter
		want   string
	}{
		{name: "default", filter: nil, want: "e.created_at DESC, e.id DESC"},
		{name: "model", filter: &service.OpsErrorLogFilter{SortBy: "model", SortOrder: "asc"}, want: "COALESCE(NULLIF(TRIM(e.requested_model), ''), e.model) ASC, e.id ASC"},
		{name: "status", filter: &service.OpsErrorLogFilter{SortBy: "status_code", SortOrder: "desc"}, want: "COALESCE(e.upstream_status_code, e.status_code, 0) DESC, e.id DESC"},
		{name: "unknown", filter: &service.OpsErrorLogFilter{SortBy: "DROP TABLE", SortOrder: "sideways"}, want: "e.created_at DESC, e.id DESC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := opsErrorLogsOrderBy(tt.filter); got != tt.want {
				t.Fatalf("order = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildOpsErrorLogsWhere_UserQueryUsesExistsSubquery(t *testing.T) {
	filter := &service.OpsErrorLogFilter{
		UserQuery: "admin@",
	}

	where, args := buildOpsErrorLogsWhere(filter)
	if where == "" {
		t.Fatalf("where should not be empty")
	}
	if len(args) != 1 {
		t.Fatalf("args len = %d, want 1", len(args))
	}
	if !strings.Contains(where, "EXISTS (SELECT 1 FROM users u WHERE u.id = e.user_id AND u.email ILIKE $") {
		t.Fatalf("where should include EXISTS user email condition: %s", where)
	}
}
