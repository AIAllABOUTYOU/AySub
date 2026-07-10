package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newAdminUserRoleTestRouter(svc *stubAdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 77})
		c.Next()
	})
	handler := NewUserHandler(svc, nil, nil, nil)
	router.POST("/users", handler.Create)
	router.PUT("/users/:id", handler.Update)
	return router
}

func TestUserHandlerCreatePassesRoleAndActor(t *testing.T) {
	svc := newStubAdminService()
	router := newAdminUserRoleTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{
		"email":"new-admin@example.com",
		"password":"strong-pass",
		"role":"admin"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, svc.createdUsers, 1)
	require.Equal(t, service.RoleAdmin, svc.createdUsers[0].Role)
	require.Equal(t, int64(77), svc.createdUsers[0].ActorAdminID)
}

func TestUserHandlerCreateOmitsRoleForServiceDefault(t *testing.T) {
	svc := newStubAdminService()
	router := newAdminUserRoleTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{
		"email":"new-user@example.com",
		"password":"strong-pass"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, svc.createdUsers, 1)
	require.Empty(t, svc.createdUsers[0].Role)
}

func TestUserHandlerRejectsInvalidRole(t *testing.T) {
	svc := newStubAdminService()
	router := newAdminUserRoleTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{
		"email":"bad-role@example.com",
		"password":"strong-pass",
		"role":"owner"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Empty(t, svc.createdUsers)
}

func TestUserHandlerUpdatePassesRoleAndActor(t *testing.T) {
	svc := newStubAdminService()
	router := newAdminUserRoleTestRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/users/42", strings.NewReader(`{"role":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{42}, svc.updatedUserIDs)
	require.Len(t, svc.updatedUsers, 1)
	require.Equal(t, service.RoleAdmin, svc.updatedUsers[0].Role)
	require.Equal(t, int64(77), svc.updatedUsers[0].ActorAdminID)
}
