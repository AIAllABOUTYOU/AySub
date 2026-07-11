package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubCredentialShadowRepo struct {
	AccountRepository
	parent *Account
}

func (s *stubCredentialShadowRepo) GetByID(context.Context, int64) (*Account, error) {
	return s.parent, nil
}

func TestResolveCredentialAccount(t *testing.T) {
	parentID := int64(100)
	parent := &Account{ID: parentID, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive}
	shadow := &Account{ID: 200, ParentAccountID: &parentID, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	repo := &stubCredentialShadowRepo{parent: parent}
	got, err := resolveCredentialAccount(context.Background(), repo, shadow)
	require.NoError(t, err)
	require.Same(t, parent, got)

	repo.parent = &Account{ID: parentID, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	_, err = resolveCredentialAccount(context.Background(), repo, shadow)
	require.Error(t, err)
}
