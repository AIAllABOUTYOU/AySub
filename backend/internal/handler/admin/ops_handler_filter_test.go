package admin

import (
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseAdminOpsErrorFilter_FullUsageContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("GET", "/admin/ops/errors?time_range=24h&view=all&platform=openai&group_id=4&account_id=5&user_id=6&api_key_id=7&model=gpt-5.6&request_type=ws_v2&phase=upstream&category=rate_limit&status_codes=429,503&resolved=yes&sort_by=status_code&sort_order=asc", nil)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	filter, err := parseAdminOpsErrorFilter(c, "1h")
	require.NoError(t, err)
	require.Equal(t, "all", filter.View)
	require.Equal(t, "openai", filter.Platform)
	require.Equal(t, int64(4), *filter.GroupID)
	require.Equal(t, int64(5), *filter.AccountID)
	require.Equal(t, int64(6), *filter.UserID)
	require.Equal(t, int64(7), *filter.APIKeyID)
	require.Equal(t, "gpt-5.6", filter.Model)
	require.Equal(t, int16(service.RequestTypeWSV2), *filter.RequestType)
	require.Equal(t, "upstream", filter.Phase)
	require.Equal(t, []string{"rate_limit_error"}, filter.ErrorTypesAny)
	require.Equal(t, []int{429, 503}, filter.StatusCodes)
	require.True(t, *filter.Resolved)
	require.Equal(t, "status_code", filter.SortBy)
	require.Equal(t, "asc", filter.SortOrder)
	require.NotNil(t, filter.StartTime)
	require.NotNil(t, filter.EndTime)
}

func TestParseAdminOpsErrorFilter_RejectsInvalidValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []string{
		"?user_id=0",
		"?api_key_id=abc",
		"?request_type=socket",
		"?stream=maybe",
		"?status_codes=429,nope",
	}
	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/admin/ops/errors"+query, nil)
			_, err := parseAdminOpsErrorFilter(c, "1h")
			require.Error(t, err)
		})
	}
}
