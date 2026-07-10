//go:build unit

package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type systemHandlerUpdateServiceStub struct {
	performErr       error
	updateInfo       *service.UpdateInfo
	checkErr         error
	checkForces      []bool
	performCall      int
	rollbackVersions []service.RollbackVersion
	rollbackListErr  error
	rollbackErr      error
	rollbackCall     int
	rollbackToErr    error
	rollbackTargets  []string
}

func (s *systemHandlerUpdateServiceStub) CheckUpdate(_ context.Context, force bool) (*service.UpdateInfo, error) {
	s.checkForces = append(s.checkForces, force)
	return s.updateInfo, s.checkErr
}

func (s *systemHandlerUpdateServiceStub) PerformUpdate(context.Context) error {
	s.performCall++
	return s.performErr
}

func (s *systemHandlerUpdateServiceStub) Rollback() error {
	s.rollbackCall++
	return s.rollbackErr
}

func (s *systemHandlerUpdateServiceStub) ListRollbackVersions(context.Context) ([]service.RollbackVersion, error) {
	return s.rollbackVersions, s.rollbackListErr
}

func (s *systemHandlerUpdateServiceStub) RollbackToVersion(_ context.Context, version string) error {
	s.rollbackTargets = append(s.rollbackTargets, version)
	return s.rollbackToErr
}

type systemUpdateResponseEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Message         string `json:"message"`
		AlreadyUpToDate bool   `json:"already_up_to_date"`
		CurrentVersion  string `json:"current_version"`
		LatestVersion   string `json:"latest_version"`
		OperationID     string `json:"operation_id"`
	} `json:"data"`
}

type systemUpdateErrorEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func newSystemHandlerTestRouter(t *testing.T, updateSvc *systemHandlerUpdateServiceStub, repo *memoryIdempotencyRepoStub) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	service.SetDefaultIdempotencyCoordinator(nil)
	t.Cleanup(func() {
		service.SetDefaultIdempotencyCoordinator(nil)
	})

	lockSvc := service.NewSystemOperationLockService(repo, service.IdempotencyConfig{
		ProcessingTimeout:  time.Second,
		SystemOperationTTL: time.Minute,
	})
	handler := NewSystemHandler(updateSvc, lockSvc)

	router := gin.New()
	router.GET("/api/v1/admin/system/rollback-versions", handler.GetRollbackVersions)
	router.POST("/api/v1/admin/system/update", handler.PerformUpdate)
	router.POST("/api/v1/admin/system/rollback", handler.Rollback)
	return router
}

func requireSystemLockStatus(t *testing.T, repo *memoryIdempotencyRepoStub, wantStatus string) {
	t.Helper()
	repo.mu.Lock()
	defer repo.mu.Unlock()

	for _, record := range repo.data {
		if record.Status == wantStatus {
			return
		}
	}
	t.Fatalf("system lock status %q not found in records: %#v", wantStatus, repo.data)
}

func TestSystemHandlerPerformUpdateAlreadyUpToDateReturnsOK(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{
		performErr: service.ErrNoUpdateAvailable,
		updateInfo: &service.UpdateInfo{
			CurrentVersion: "0.1.132",
			LatestVersion:  "0.1.132",
			HasUpdate:      false,
		},
	}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouter(t, updateSvc, repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", nil)
	req.Header.Set("Idempotency-Key", "already-up-to-date")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, updateSvc.performCall)
	require.Equal(t, []bool{false}, updateSvc.checkForces)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusSucceeded)

	var body systemUpdateResponseEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Equal(t, "success", body.Message)
	require.Equal(t, "Already up to date", body.Data.Message)
	require.True(t, body.Data.AlreadyUpToDate)
	require.Equal(t, "0.1.132", body.Data.CurrentVersion)
	require.Equal(t, "0.1.132", body.Data.LatestVersion)
	require.NotEmpty(t, body.Data.OperationID)
}

func TestSystemHandlerPerformUpdateFailureStillReturnsInternalError(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{
		performErr: errors.New("download failed"),
	}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouter(t, updateSvc, repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", nil)
	req.Header.Set("Idempotency-Key", "real-failure")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, 1, updateSvc.performCall)
	require.Empty(t, updateSvc.checkForces)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusFailedRetryable)

	var body systemUpdateErrorEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, http.StatusInternalServerError, body.Code)
	require.Equal(t, "internal error", body.Message)
}

func TestSystemHandlerGetRollbackVersions(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{rollbackVersions: []service.RollbackVersion{
		{Version: "0.1.138", PublishedAt: "2026-07-01T00:00:00Z"},
	}}
	router := newSystemHandlerTestRouter(t, updateSvc, newMemoryIdempotencyRepoStub())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/system/rollback-versions", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"version":"0.1.138"`)
}

func TestSystemHandlerRollbackWithoutBodyUsesLocalBackup(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouter(t, updateSvc, repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/rollback", nil)
	req.Header.Set("Idempotency-Key", "local-backup")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, updateSvc.rollbackCall)
	require.Empty(t, updateSvc.rollbackTargets)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusSucceeded)
}

func TestSystemHandlerRollbackToVersionUsesOnlineRelease(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouter(t, updateSvc, repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/rollback", strings.NewReader(`{"version":"v0.1.138"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "online-version")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Zero(t, updateSvc.rollbackCall)
	require.Equal(t, []string{"0.1.138"}, updateSvc.rollbackTargets)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusSucceeded)
}

func TestSystemHandlerRollbackRejectsMalformedBody(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{}
	router := newSystemHandlerTestRouter(t, updateSvc, newMemoryIdempotencyRepoStub())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/rollback", strings.NewReader(`{"version":`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, updateSvc.rollbackCall)
	require.Empty(t, updateSvc.rollbackTargets)
}

func TestBuildSystemOperationIDIncludesRollbackTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctxA, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctxA.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/rollback", nil)
	ctxA.Request.Header.Set("Idempotency-Key", "same-key")

	ctxB, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctxB.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/rollback", nil)
	ctxB.Request.Header.Set("Idempotency-Key", "same-key")

	require.NotEqual(t,
		buildSystemOperationID(ctxA, "rollback", "0.1.138"),
		buildSystemOperationID(ctxB, "rollback", "0.1.137"),
	)
}
