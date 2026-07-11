package admin

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func enrichShadowParentInfo(items []AccountWithConcurrency, parents map[int64]*service.Account) {
	for i := range items {
		account := items[i].Account
		if account == nil || account.ParentAccountID == nil {
			continue
		}
		parent := parents[*account.ParentAccountID]
		if parent == nil {
			continue
		}
		account.ParentEmail = parent.GetCredential("email")
		account.ParentPlanType = parent.GetCredential("plan_type")
		account.ParentSubscriptionExpiresAt = parent.GetCredential("subscription_expires_at")
		account.ParentChatGPTAccountID = parent.GetCredential("chatgpt_account_id")
		account.ParentPrivacyMode = parent.GetExtraString("privacy_mode")
	}
}

func (h *AccountHandler) enrichShadowParents(ctx context.Context, items []AccountWithConcurrency) {
	seen := make(map[int64]struct{})
	for i := range items {
		if account := items[i].Account; account != nil && account.ParentAccountID != nil {
			seen[*account.ParentAccountID] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	parents, err := h.adminService.GetAccountsByIDs(ctx, ids)
	if err != nil {
		return
	}
	byID := make(map[int64]*service.Account, len(parents))
	for _, parent := range parents {
		if parent != nil {
			byID[parent.ID] = parent
		}
	}
	enrichShadowParentInfo(items, byID)
}
