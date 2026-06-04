package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestPaymentDashboardStatsIncludesOpsMetrics(t *testing.T) {
	ctx := context.Background()
	client := newPaymentStatsTestClient(t)
	now := time.Now().UTC()

	user, err := client.User.Create().
		SetEmail("payment-ops@example.com").
		SetPasswordHash("hash").
		SetUsername("payment-ops-user").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("enabled-alipay").
		SetConfig("{}").
		SetSupportedTypes(payment.TypeAlipay).
		SetEnabled(true).
		SetRefundEnabled(true).
		SetAllowUserRefund(true).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeStripe).
		SetName("disabled-stripe").
		SetConfig("{}").
		SetSupportedTypes(payment.TypeStripe).
		SetEnabled(false).
		Save(ctx)
	require.NoError(t, err)

	paidAt := now.Add(-time.Hour)
	createPaymentStatsOrder(t, ctx, client, user, "completed", OrderStatusCompleted, 120, paidAt, nil, now.Add(time.Hour))
	createPaymentStatsOrder(t, ctx, client, user, "refund-requested", OrderStatusRefundRequested, 30, paidAt, nil, now.Add(time.Hour))
	createPaymentStatsOrder(t, ctx, client, user, "refunding", OrderStatusRefunding, 40, paidAt, nil, now.Add(time.Hour))
	createPaymentStatsOrder(t, ctx, client, user, "refund-failed", OrderStatusRefundFailed, 50, paidAt, nil, now.Add(time.Hour))
	createPaymentStatsOrder(t, ctx, client, user, "refunded", OrderStatusRefunded, 60, paidAt, nil, now.Add(time.Hour))
	createPaymentStatsOrder(t, ctx, client, user, "partial-refunded", OrderStatusPartiallyRefunded, 70, paidAt, nil, now.Add(time.Hour))
	createPaymentStatsOrder(t, ctx, client, user, "paid-stuck", OrderStatusPaid, 80, now.Add(-20*time.Minute), nil, now.Add(time.Hour))
	createPaymentStatsOrder(t, ctx, client, user, "recharging-stuck", OrderStatusRecharging, 90, now.Add(-20*time.Minute), nil, now.Add(time.Hour))
	createPaymentStatsOrder(t, ctx, client, user, "stale-pending", OrderStatusPending, 100, time.Time{}, nil, now.Add(-time.Hour))

	for _, action := range []string{
		"PAYMENT_AMOUNT_MISMATCH",
		"REFUND_GATEWAY_FAILED",
		"FULFILLMENT_FAILED",
	} {
		_, err = client.PaymentAuditLog.Create().
			SetOrderID("ops-test").
			SetAction(action).
			SetOperator("test").
			SetDetail("{}").
			SetCreatedAt(now.Add(-time.Hour)).
			Save(ctx)
		require.NoError(t, err)
	}
	_, err = client.PaymentAuditLog.Create().
		SetOrderID("old-ops-test").
		SetAction("FULFILLMENT_FAILED").
		SetOperator("test").
		SetDetail("{}").
		SetCreatedAt(now.AddDate(0, 0, -60)).
		Save(ctx)
	require.NoError(t, err)

	stats, err := (&PaymentService{entClient: client}).GetDashboardStats(ctx, 30)
	require.NoError(t, err)
	require.NotNil(t, stats)

	require.Equal(t, 30, stats.Ops.WindowDays)
	require.Equal(t, 2, stats.Ops.CallbackFailures)
	require.Equal(t, 1, stats.Ops.OrderInconsistencies)
	require.Equal(t, 1, stats.Ops.ProviderUnavailable)
	require.Equal(t, 1, stats.Ops.RefundRequested)
	require.Equal(t, 1, stats.Ops.Refunding)
	require.Equal(t, 1, stats.Ops.RefundFailed)
	require.Equal(t, 2, stats.Ops.Refunded)
	require.Equal(t, 1, stats.Ops.FulfillmentFailed)
	require.Equal(t, 2, stats.Ops.PaidNotCompleted)
	require.Equal(t, 1, stats.Ops.StalePending)
	require.Equal(t, 1, stats.Ops.EnabledProviderInstances)
	require.Equal(t, 1, stats.Ops.DisabledProviderInstances)
	require.Equal(t, 1, stats.Ops.RefundEnabledProviderInstances)
	require.Equal(t, 1, stats.Ops.UserRefundEnabledProviderInstances)
}

func createPaymentStatsOrder(
	t *testing.T,
	ctx context.Context,
	client *dbent.Client,
	user *dbent.User,
	suffix string,
	status string,
	amount float64,
	paidAt time.Time,
	completedAt *time.Time,
	expiresAt time.Time,
) {
	t.Helper()

	create := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(amount).
		SetPayAmount(amount).
		SetFeeRate(0).
		SetRechargeCode("PAYMENT-OPS-" + strings.ToUpper(strings.ReplaceAll(suffix, "-", "_"))).
		SetOutTradeNo("sub2_payment_ops_" + suffix).
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-" + suffix).
		SetOrderType(payment.OrderTypeBalance).
		SetProviderInstanceID("1").
		SetProviderKey(payment.TypeAlipay).
		SetStatus(status).
		SetExpiresAt(expiresAt).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com")
	if !paidAt.IsZero() {
		create.SetPaidAt(paidAt)
	}
	if completedAt != nil {
		create.SetCompletedAt(*completedAt)
	}
	_, err := create.Save(ctx)
	require.NoError(t, err)
}

func newPaymentStatsTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	dbName := fmt.Sprintf(
		"file:%s?mode=memory&cache=shared&_fk=1",
		strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()),
	)
	db, err := sql.Open("sqlite", dbName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}
