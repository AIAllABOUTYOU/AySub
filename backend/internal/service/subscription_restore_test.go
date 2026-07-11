package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type restoreSubscriptionRepoStub struct {
	userSubRepoNoop
	sub          *UserSubscription
	existsActive bool
	restoredWith string
}

func (r *restoreSubscriptionRepoStub) GetByIDIncludeDeleted(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	copy := *r.sub
	return &copy, nil
}

func (r *restoreSubscriptionRepoStub) ExistsActiveByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	return r.existsActive, nil
}

func (r *restoreSubscriptionRepoStub) Restore(_ context.Context, id int64, status string) (*UserSubscription, error) {
	r.restoredWith = status
	copy := *r.sub
	copy.ID = id
	copy.Status = status
	copy.DeletedAt = nil
	return &copy, nil
}

func TestRestoreSubscription(t *testing.T) {
	deletedAt := time.Now().Add(-time.Hour)

	t.Run("restores expired active record as expired", func(t *testing.T) {
		repo := &restoreSubscriptionRepoStub{sub: &UserSubscription{
			ID: 1, UserID: 10, GroupID: 20, Status: SubscriptionStatusActive,
			ExpiresAt: time.Now().Add(-time.Minute), DeletedAt: &deletedAt,
		}}
		svc := NewSubscriptionService(nil, repo, nil, nil, nil)

		restored, err := svc.RestoreSubscription(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, SubscriptionStatusExpired, repo.restoredWith)
		require.Equal(t, SubscriptionStatusExpired, restored.Status)
		require.Nil(t, restored.DeletedAt)
	})

	t.Run("rejects non revoked record", func(t *testing.T) {
		repo := &restoreSubscriptionRepoStub{sub: &UserSubscription{ID: 2, UserID: 10, GroupID: 20, Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(time.Hour)}}
		svc := NewSubscriptionService(nil, repo, nil, nil, nil)
		_, err := svc.RestoreSubscription(context.Background(), 2)
		require.ErrorIs(t, err, ErrSubscriptionNotRevoked)
	})

	t.Run("rejects active replacement conflict", func(t *testing.T) {
		repo := &restoreSubscriptionRepoStub{
			sub:          &UserSubscription{ID: 3, UserID: 10, GroupID: 20, Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(time.Hour), DeletedAt: &deletedAt},
			existsActive: true,
		}
		svc := NewSubscriptionService(nil, repo, nil, nil, nil)
		_, err := svc.RestoreSubscription(context.Background(), 3)
		require.ErrorIs(t, err, ErrSubscriptionRestoreConflict)
	})
}
