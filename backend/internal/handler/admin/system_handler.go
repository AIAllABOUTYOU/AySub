package admin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/sysutil"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SystemHandler handles system-related operations
type SystemHandler struct {
	updateSvc systemUpdateService
	lockSvc   *service.SystemOperationLockService
}

type systemUpdateService interface {
	CheckUpdate(ctx context.Context, force bool) (*service.UpdateInfo, error)
	PerformUpdate(ctx context.Context) error
	Rollback() error
	ListRollbackVersions(ctx context.Context) ([]service.RollbackVersion, error)
	RollbackToVersion(ctx context.Context, version string) error
}

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler(updateSvc systemUpdateService, lockSvc *service.SystemOperationLockService) *SystemHandler {
	return &SystemHandler{
		updateSvc: updateSvc,
		lockSvc:   lockSvc,
	}
}

// GetVersion returns the current version
// GET /api/v1/admin/system/version
func (h *SystemHandler) GetVersion(c *gin.Context) {
	info, _ := h.updateSvc.CheckUpdate(c.Request.Context(), false)
	response.Success(c, gin.H{
		"version": info.CurrentVersion,
	})
}

// CheckUpdates checks for available updates
// GET /api/v1/admin/system/check-updates
func (h *SystemHandler) CheckUpdates(c *gin.Context) {
	force := c.Query("force") == "true"
	info, err := h.updateSvc.CheckUpdate(c.Request.Context(), force)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, info)
}

// GetRollbackVersions returns up to three stable releases older than the current version.
// GET /api/v1/admin/system/rollback-versions
func (h *SystemHandler) GetRollbackVersions(c *gin.Context) {
	versions, err := h.updateSvc.ListRollbackVersions(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"versions": versions})
}

// PerformUpdate downloads and applies the update
// POST /api/v1/admin/system/update
func (h *SystemHandler) PerformUpdate(c *gin.Context) {
	operationID := buildSystemOperationID(c, "update")
	payload := gin.H{"operation_id": operationID}
	executeAdminIdempotentJSON(c, "admin.system.update", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		var releaseReason string
		succeeded := false
		defer func() {
			release(releaseReason, succeeded)
		}()

		if err := h.updateSvc.PerformUpdate(ctx); err != nil {
			if errors.Is(err, service.ErrNoUpdateAvailable) {
				info, checkErr := h.updateSvc.CheckUpdate(ctx, false)
				if checkErr != nil {
					releaseReason = "SYSTEM_UPDATE_FAILED"
					return nil, checkErr
				}
				succeeded = true
				return gin.H{
					"message":            "Already up to date",
					"already_up_to_date": true,
					"current_version":    info.CurrentVersion,
					"latest_version":     info.LatestVersion,
					"operation_id":       lock.OperationID(),
				}, nil
			}
			releaseReason = "SYSTEM_UPDATE_FAILED"
			return nil, err
		}
		succeeded = true

		return gin.H{
			"message":      "Update completed. Please restart the service.",
			"need_restart": true,
			"operation_id": lock.OperationID(),
		}, nil
	})
}

// Rollback restores the previous version
// POST /api/v1/admin/system/rollback
func (h *SystemHandler) Rollback(c *gin.Context) {
	var req struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	version := strings.TrimPrefix(strings.TrimSpace(req.Version), "v")
	operationID := buildSystemOperationID(c, "rollback", version)
	payload := gin.H{"operation_id": operationID, "version": version}
	executeAdminIdempotentJSON(c, "admin.system.rollback", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		var releaseReason string
		succeeded := false
		defer func() {
			release(releaseReason, succeeded)
		}()

		var rollbackErr error
		if version == "" {
			rollbackErr = h.updateSvc.Rollback()
		} else {
			rollbackErr = h.updateSvc.RollbackToVersion(ctx, version)
		}
		if rollbackErr != nil {
			releaseReason = "SYSTEM_ROLLBACK_FAILED"
			return nil, rollbackErr
		}
		succeeded = true

		return gin.H{
			"message":      "Rollback completed. Please restart the service.",
			"need_restart": true,
			"operation_id": lock.OperationID(),
		}, nil
	})
}

// RestartService restarts the systemd service
// POST /api/v1/admin/system/restart
func (h *SystemHandler) RestartService(c *gin.Context) {
	operationID := buildSystemOperationID(c, "restart")
	payload := gin.H{"operation_id": operationID}
	executeAdminIdempotentJSON(c, "admin.system.restart", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		succeeded := false
		defer func() {
			release("", succeeded)
		}()

		// Schedule service restart in background after sending response
		// This ensures the client receives the success response before the service restarts
		go func() {
			// Wait a moment to ensure the response is sent
			time.Sleep(500 * time.Millisecond)
			sysutil.RestartServiceAsync()
		}()
		succeeded = true
		return gin.H{
			"message":      "Service restart initiated",
			"operation_id": lock.OperationID(),
		}, nil
	})
}

func (h *SystemHandler) acquireSystemLock(
	ctx context.Context,
	operationID string,
) (*service.SystemOperationLock, func(string, bool), error) {
	if h.lockSvc == nil {
		return nil, nil, service.ErrIdempotencyStoreUnavail
	}
	lock, err := h.lockSvc.Acquire(ctx, operationID)
	if err != nil {
		return nil, nil, err
	}
	release := func(reason string, succeeded bool) {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = h.lockSvc.Release(releaseCtx, lock, succeeded, reason)
	}
	return lock, release, nil
}

func buildSystemOperationID(c *gin.Context, operation string, targets ...string) string {
	operationScope := operation
	if len(targets) > 0 {
		target := strings.TrimSpace(strings.Join(targets, "|"))
		if target != "" {
			targetHash := service.HashIdempotencyKey(target)
			if len(targetHash) > 8 {
				targetHash = targetHash[:8]
			}
			operationScope += "-" + targetHash
		}
	}
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" {
		return "sysop-" + operationScope + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	actorScope := "admin:0"
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		actorScope = "admin:" + strconv.FormatInt(subject.UserID, 10)
	}
	seed := operationScope + "|" + actorScope + "|" + c.FullPath() + "|" + key
	hash := service.HashIdempotencyKey(seed)
	if len(hash) > 24 {
		hash = hash[:24]
	}
	return "sysop-" + hash
}
