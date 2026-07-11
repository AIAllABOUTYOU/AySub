package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPaymentConfigAPIExposesAndUpdatesSubscriptionUSDToCNYRate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingSubscriptionUSDToCNYRate: "7.15",
	}}
	h := NewPaymentHandler(nil, service.NewPaymentConfigService(nil, repo, nil))

	getRecorder := httptest.NewRecorder()
	getContext, _ := gin.CreateTestContext(getRecorder)
	getContext.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/payment/config", nil)
	h.GetConfig(getContext)

	require.Equal(t, http.StatusOK, getRecorder.Code)
	var getResponse struct {
		Data struct {
			SubscriptionUSDToCNYRate float64 `json:"subscription_usd_to_cny_rate"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(getRecorder.Body.Bytes(), &getResponse))
	require.Equal(t, 7.15, getResponse.Data.SubscriptionUSDToCNYRate)

	updateRecorder := httptest.NewRecorder()
	updateContext, _ := gin.CreateTestContext(updateRecorder)
	updateContext.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/payment/config", bytes.NewBufferString(`{"subscription_usd_to_cny_rate":0}`))
	updateContext.Request.Header.Set("Content-Type", "application/json")
	h.UpdateConfig(updateContext)

	require.Equal(t, http.StatusOK, updateRecorder.Code)
	require.Equal(t, "0", repo.lastUpdates[service.SettingSubscriptionUSDToCNYRate])
}

func TestPaymentConfigAPIRejectsInvalidSubscriptionUSDToCNYRate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, rate := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		repo := &settingHandlerRepoStub{values: map[string]string{}}
		svc := service.NewPaymentConfigService(nil, repo, nil)
		err := svc.UpdatePaymentConfig(context.Background(), service.UpdatePaymentConfigRequest{SubscriptionUSDToCNYRate: &rate})
		require.Error(t, err)
		require.Nil(t, repo.lastUpdates)
	}
}

func TestSystemSettingsDTOExposesSubscriptionUSDToCNYRate(t *testing.T) {
	data := systemSettingsResponseData(dto.SystemSettings{PaymentSubscriptionUSDToCNYRate: 7.15}, nil)
	require.Equal(t, 7.15, data["payment_subscription_usd_to_cny_rate"])
	rate := 7.15
	require.True(t, hasPaymentFields(UpdateSettingsRequest{PaymentSubscriptionUSDToCNYRate: &rate}))
}
