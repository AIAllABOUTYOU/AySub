//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type atomicRoleUserRepoStub struct {
	*rpmUserRepoStub
	stored     User
	roleErr    error
	guardCalls int
}

func newAtomicRoleUserRepoStub(user User) *atomicRoleUserRepoStub {
	base := &userRepoStub{user: &user}
	return &atomicRoleUserRepoStub{
		rpmUserRepoStub: &rpmUserRepoStub{userRepoStub: base},
		stored:          user,
	}
}

func (s *atomicRoleUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	if s.stored.ID != id {
		return nil, ErrUserNotFound
	}
	clone := s.stored
	return &clone, nil
}

func (s *atomicRoleUserRepoStub) UpdateUserWithAdminRoleGuard(_ context.Context, user *User) (string, error) {
	s.guardCalls++
	oldRole := s.stored.Role
	if s.roleErr != nil {
		return oldRole, s.roleErr
	}
	clone := *user
	s.stored = clone
	s.lastUpdated = &clone
	s.userRepoStub.user = &clone
	return oldRole, nil
}

type roleAuditWriterStub struct {
	inputs []SecurityAuditCreateInput
	err    error
}

func (s *roleAuditWriterStub) CreateAuditLog(_ context.Context, input SecurityAuditCreateInput) (*SecurityAuditLog, error) {
	s.inputs = append(s.inputs, input)
	if s.err != nil {
		return nil, s.err
	}
	return &SecurityAuditLog{}, nil
}

func TestAdminServiceCreateUserDefaultsRoleToUser(t *testing.T) {
	repo := &userRepoStub{nextID: 31}
	svc := &adminServiceImpl{userRepo: repo}

	user, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:    "plain@test.com",
		Password: "strong-pass",
	})

	require.NoError(t, err)
	require.Equal(t, RoleUser, user.Role)
}

func TestAdminServiceCreateUserWithAdminRolePersistsAudit(t *testing.T) {
	repo := &userRepoStub{nextID: 30}
	audit := &roleAuditWriterStub{}
	svc := &adminServiceImpl{userRepo: repo, securityAuditWriter: audit}

	user, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:        "admin@test.com",
		Password:     "strong-pass",
		Role:         RoleAdmin,
		ActorAdminID: 7,
	})

	require.NoError(t, err)
	require.Equal(t, RoleAdmin, user.Role)
	require.Len(t, audit.inputs, 1)
	require.Equal(t, "admin.user.role.create", audit.inputs[0].Action)
	require.Equal(t, SecurityAuditResultSuccess, audit.inputs[0].Result)
	require.Equal(t, int64(7), *audit.inputs[0].ActorID)
	require.Equal(t, user.ID, *audit.inputs[0].SubjectID)
	require.Equal(t, "", audit.inputs[0].DiffSummary["old_role"])
	require.Equal(t, RoleAdmin, audit.inputs[0].DiffSummary["new_role"])
}

func TestAdminServiceCreateUserRejectsInvalidRole(t *testing.T) {
	repo := &userRepoStub{nextID: 32}
	svc := &adminServiceImpl{userRepo: repo}

	_, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:    "bad@test.com",
		Password: "strong-pass",
		Role:     "owner",
	})

	require.ErrorIs(t, err, ErrInvalidUserRole)
	require.Empty(t, repo.created)
}

func TestAdminServiceUpdateUserPromotesAndInvalidatesAuthCache(t *testing.T) {
	repo := newAtomicRoleUserRepoStub(User{ID: 42, Email: "u@example.com", Role: RoleUser, Status: StatusActive})
	invalidator := &authCacheInvalidatorStub{}
	audit := &roleAuditWriterStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       &redeemRepoStub{},
		authCacheInvalidator: invalidator,
		securityAuditWriter:  audit,
	}

	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{Role: RoleAdmin, ActorAdminID: 7})

	require.NoError(t, err)
	require.Equal(t, RoleAdmin, updated.Role)
	require.Equal(t, 1, repo.guardCalls)
	require.Equal(t, []int64{42}, invalidator.userIDs)
	require.Len(t, audit.inputs, 1)
	require.Equal(t, RoleUser, audit.inputs[0].DiffSummary["old_role"])
	require.Equal(t, RoleAdmin, audit.inputs[0].DiffSummary["new_role"])
	require.Equal(t, SecurityAuditResultSuccess, audit.inputs[0].Result)
}

func TestAdminServiceUpdateUserOmittedRoleKeepsExisting(t *testing.T) {
	base := &userRepoStub{user: &User{ID: 42, Email: "a@example.com", Role: RoleAdmin, Status: StatusActive}}
	repo := &rpmUserRepoStub{userRepoStub: base}
	svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: &redeemRepoStub{}}
	newName := "renamed"

	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{Username: &newName})

	require.NoError(t, err)
	require.Equal(t, RoleAdmin, updated.Role)
	require.Equal(t, RoleAdmin, repo.lastUpdated.Role)
}

func TestAdminServiceUpdateUserDemotesAdminWhenAnotherAdminExists(t *testing.T) {
	repo := newAtomicRoleUserRepoStub(User{ID: 42, Email: "a@example.com", Role: RoleAdmin, Status: StatusActive})
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       &redeemRepoStub{},
		authCacheInvalidator: invalidator,
	}

	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{Role: RoleUser, ActorAdminID: 99})

	require.NoError(t, err)
	require.Equal(t, RoleUser, updated.Role)
	require.Equal(t, []int64{42}, invalidator.userIDs)
}

func TestAdminServiceUpdateUserRejectsSelfDemotion(t *testing.T) {
	repo := newAtomicRoleUserRepoStub(User{ID: 42, Email: "a@example.com", Role: RoleAdmin, Status: StatusActive})
	audit := &roleAuditWriterStub{}
	svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: &redeemRepoStub{}, securityAuditWriter: audit}

	_, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{Role: RoleUser, ActorAdminID: 42})

	require.ErrorIs(t, err, ErrCannotDemoteSelf)
	require.Zero(t, repo.guardCalls)
	require.Len(t, audit.inputs, 1)
	require.Equal(t, SecurityAuditResultDenied, audit.inputs[0].Result)
}

func TestAdminServiceUpdateUserRejectsLastActiveAdminDemotion(t *testing.T) {
	repo := newAtomicRoleUserRepoStub(User{ID: 42, Email: "a@example.com", Role: RoleAdmin, Status: StatusActive})
	repo.roleErr = ErrLastActiveAdmin
	audit := &roleAuditWriterStub{}
	svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: &redeemRepoStub{}, securityAuditWriter: audit}

	_, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{Role: RoleUser, ActorAdminID: 99})

	require.ErrorIs(t, err, ErrLastActiveAdmin)
	require.Nil(t, repo.lastUpdated)
	require.Len(t, audit.inputs, 1)
	require.Equal(t, SecurityAuditResultDenied, audit.inputs[0].Result)
	require.Equal(t, RoleAdmin, audit.inputs[0].DiffSummary["old_role"])
	require.Equal(t, RoleUser, audit.inputs[0].DiffSummary["new_role"])
}

func TestAdminServiceUpdateUserRejectsInvalidRole(t *testing.T) {
	repo := newAtomicRoleUserRepoStub(User{ID: 42, Email: "u@example.com", Role: RoleUser, Status: StatusActive})
	audit := &roleAuditWriterStub{}
	svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: &redeemRepoStub{}, securityAuditWriter: audit}

	_, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{Role: "owner", ActorAdminID: 7})

	require.ErrorIs(t, err, ErrInvalidUserRole)
	require.Zero(t, repo.guardCalls)
	require.Len(t, audit.inputs, 1)
	require.Equal(t, SecurityAuditResultDenied, audit.inputs[0].Result)
}
