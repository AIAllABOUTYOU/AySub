package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type sparkShadowAccountRepoStub struct {
	AccountRepository
	accounts map[int64]*Account
	shadows  map[int64][]*Account
	nextID   int64
	deleted  []int64
}

func (s *sparkShadowAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	return s.accounts[id], nil
}

func (s *sparkShadowAccountRepoStub) ListShadowsByParent(_ context.Context, parentID int64) ([]*Account, error) {
	return s.shadows[parentID], nil
}

func (s *sparkShadowAccountRepoStub) Create(_ context.Context, account *Account) error {
	s.nextID++
	account.ID = s.nextID
	s.accounts[account.ID] = account
	if account.ParentAccountID != nil {
		s.shadows[*account.ParentAccountID] = append(s.shadows[*account.ParentAccountID], account)
	}
	return nil
}

func (s *sparkShadowAccountRepoStub) BindGroups(_ context.Context, accountID int64, groupIDs []int64) error {
	s.accounts[accountID].GroupIDs = append([]int64(nil), groupIDs...)
	return nil
}

func (s *sparkShadowAccountRepoStub) Delete(_ context.Context, id int64) error {
	s.deleted = append(s.deleted, id)
	delete(s.accounts, id)
	return nil
}

func TestAdminServiceCreateAndDeleteSparkShadow(t *testing.T) {
	parent := &Account{ID: 10, Name: "parent", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Concurrency: 4, Priority: 30, GroupIDs: []int64{7}}
	repo := &sparkShadowAccountRepoStub{accounts: map[int64]*Account{10: parent}, shadows: make(map[int64][]*Account), nextID: 100}
	svc := &adminServiceImpl{accountRepo: repo}

	shadow, err := svc.CreateShadow(context.Background(), parent.ID, ShadowOptions{})
	require.NoError(t, err)
	require.Equal(t, "parent (Spark)", shadow.Name)
	require.Equal(t, QuotaDimensionSpark, shadow.QuotaDimension)
	require.Equal(t, parent.Concurrency, shadow.Concurrency)
	require.Equal(t, parent.Priority, shadow.Priority)
	require.Equal(t, parent.GroupIDs, shadow.GroupIDs)
	require.Empty(t, shadow.GetOpenAIAccessToken())

	_, err = svc.CreateShadow(context.Background(), parent.ID, ShadowOptions{})
	require.Error(t, err)

	err = svc.DeleteAccount(context.Background(), parent.ID)
	require.NoError(t, err)
	require.Equal(t, []int64{shadow.ID, parent.ID}, repo.deleted)
}

func TestAdminServiceCreateShadowRejectsShadowParent(t *testing.T) {
	parentID := int64(10)
	shadowParent := &Account{ID: 11, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ParentAccountID: &parentID}
	repo := &sparkShadowAccountRepoStub{accounts: map[int64]*Account{11: shadowParent}, shadows: make(map[int64][]*Account)}
	_, err := (&adminServiceImpl{accountRepo: repo}).CreateShadow(context.Background(), 11, ShadowOptions{})
	require.Error(t, err)
}
