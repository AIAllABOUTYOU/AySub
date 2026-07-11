//go:build unit

package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestGetCheckoutInfoExposesSubscriptionUSDToCNYRate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := sql.Open("sqlite", "file:payment_handler_checkout_cny_rate?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	repo := &checkoutPaymentSettingRepoStub{values: map[string]string{
		service.SettingSubscriptionUSDToCNYRate: "7.15",
	}}
	h := NewPaymentHandler(nil, service.NewPaymentConfigService(client, repo, nil), nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/checkout-info", nil)

	h.GetCheckoutInfo(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Data struct {
			SubscriptionUSDToCNYRate float64 `json:"subscription_usd_to_cny_rate"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 7.15, resp.Data.SubscriptionUSDToCNYRate)
}

type checkoutPaymentSettingRepoStub struct {
	values map[string]string
}

func (s *checkoutPaymentSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, nil
}
func (s *checkoutPaymentSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}
func (s *checkoutPaymentSettingRepoStub) Set(context.Context, string, string) error { return nil }
func (s *checkoutPaymentSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}
func (s *checkoutPaymentSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (s *checkoutPaymentSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}
func (s *checkoutPaymentSettingRepoStub) Delete(context.Context, string) error { return nil }
