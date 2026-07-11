package admin

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestEnrichShadowParentInfo(t *testing.T) {
	parentID := int64(10)
	items := []AccountWithConcurrency{{Account: &dto.Account{ID: 20, ParentAccountID: &parentID}}}
	parents := map[int64]*service.Account{parentID: {
		ID: parentID, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth,
		Credentials: map[string]any{"email": "owner@example.com", "plan_type": "plus", "chatgpt_account_id": "acct-1"},
		Extra:       map[string]any{"privacy_mode": "false"},
	}}
	enrichShadowParentInfo(items, parents)
	require.Equal(t, "owner@example.com", items[0].ParentEmail)
	require.Equal(t, "plus", items[0].ParentPlanType)
	require.Equal(t, "false", items[0].ParentPrivacyMode)
	require.Equal(t, "acct-1", items[0].ParentChatGPTAccountID)
}
