//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type updateServiceCacheStub struct {
	data string
}

func (s *updateServiceCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *updateServiceCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type updateServiceGitHubClientStub struct {
	release        *GitHubRelease
	recentReleases []*GitHubRelease
	recentErr      error
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(context.Context, string) (*GitHubRelease, error) {
	return s.release, nil
}

func (s *updateServiceGitHubClientStub) FetchRecentReleases(context.Context, string, int) ([]*GitHubRelease, error) {
	return s.recentReleases, s.recentErr
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	panic("DownloadFile should not be called when no update is available")
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	panic("FetchChecksumFile should not be called when no update is available")
}

func TestUpdateServicePerformUpdateNoUpdateReturnsSentinel(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{
				TagName: "v0.1.132",
				Name:    "v0.1.132",
			},
		},
		"0.1.132",
		"release",
	)

	err := svc.PerformUpdate(context.Background())

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoUpdateAvailable))
	require.ErrorIs(t, err, ErrNoUpdateAvailable)
}

func TestUpdateServiceListRollbackVersionsFiltersSortsAndLimits(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentReleases: []*GitHubRelease{
			{TagName: "v0.1.140", PublishedAt: "newer"},
			{TagName: "v0.1.139", PublishedAt: "current"},
			{TagName: "v0.1.138", PublishedAt: "third", HTMLURL: "https://example/138"},
			{TagName: "0.1.137", PublishedAt: "second"},
			{TagName: "v0.1.136", PublishedAt: "first"},
			{TagName: "v0.1.135", PublishedAt: "limited"},
			{TagName: "v0.1.138", PublishedAt: "duplicate"},
			{TagName: "v0.1.134", Draft: true},
			{TagName: "v0.1.133", Prerelease: true},
			nil,
		}},
		"v0.1.139",
		"release",
	)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Equal(t, []RollbackVersion{
		{Version: "0.1.138", PublishedAt: "third", HTMLURL: "https://example/138"},
		{Version: "0.1.137", PublishedAt: "second"},
		{Version: "0.1.136", PublishedAt: "first"},
	}, versions)
}

func TestUpdateServiceRollbackToVersionRejectsUnlistedVersion(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentReleases: []*GitHubRelease{{TagName: "v0.1.138"}}},
		"0.1.139",
		"release",
	)

	err := svc.RollbackToVersion(context.Background(), "v0.1.137")

	require.ErrorIs(t, err, ErrRollbackVersionNotAllowed)
}
