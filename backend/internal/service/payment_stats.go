package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/paymentproviderinstance"
)

// --- Dashboard & Analytics ---

func (s *PaymentService) GetDashboardStats(ctx context.Context, days int) (*DashboardStats, error) {
	if days <= 0 {
		days = 30
	}
	now := time.Now()
	since := now.AddDate(0, 0, -days)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	paidStatuses := []string{OrderStatusCompleted, OrderStatusPaid, OrderStatusRecharging}

	orders, err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.StatusIn(paidStatuses...),
			paymentorder.PaidAtGTE(since),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	st := &DashboardStats{}
	computeBasicStats(st, orders, todayStart)

	st.PendingOrders, err = s.entClient.PaymentOrder.Query().
		Where(paymentorder.StatusEQ(OrderStatusPending)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	st.DailySeries = buildDailySeries(orders, since, days)
	st.PaymentMethods = buildMethodDistribution(orders)
	st.TopUsers = buildTopUsers(orders)
	st.Ops, err = s.buildPaymentOpsStats(ctx, since, now, days)
	if err != nil {
		return nil, err
	}

	return st, nil
}

func (s *PaymentService) buildPaymentOpsStats(ctx context.Context, since, now time.Time, days int) (PaymentOpsStats, error) {
	stats := PaymentOpsStats{WindowDays: days}
	stalePendingBefore := now.Add(-time.Duration(defaultOrderTimeoutMin+paymentGraceMinutes) * time.Minute)
	paidNotCompletedBefore := now.Add(-time.Duration(paymentGraceMinutes) * time.Minute)

	var err error
	stats.CallbackFailures, err = s.countPaymentAuditActions(ctx, since,
		"PAYMENT_PROVIDER_MISMATCH",
		"PAYMENT_PROVIDER_METADATA_MISMATCH",
		"PAYMENT_INVALID_AMOUNT",
		"PAYMENT_AMOUNT_MISMATCH",
		"PAYMENT_AFTER_EXPIRY",
		"REFUND_PROVIDER_METADATA_MISMATCH",
		"REFUND_NO_TRADE_NO",
		"REFUND_GATEWAY_FAILED",
		"REFUND_FAILED",
		"REFUND_ROLLBACK_FAILED",
	)
	if err != nil {
		return stats, err
	}

	stats.OrderInconsistencies, err = s.countPaymentAuditActions(ctx, since,
		"PAYMENT_PROVIDER_MISMATCH",
		"PAYMENT_PROVIDER_METADATA_MISMATCH",
		"PAYMENT_INVALID_AMOUNT",
		"PAYMENT_AMOUNT_MISMATCH",
		"PAYMENT_AFTER_EXPIRY",
		"REFUND_PROVIDER_METADATA_MISMATCH",
	)
	if err != nil {
		return stats, err
	}

	stats.ProviderUnavailable, err = s.entClient.PaymentProviderInstance.Query().
		Where(paymentproviderinstance.EnabledEQ(false)).
		Count(ctx)
	if err != nil {
		return stats, err
	}

	stats.RefundRequested, err = s.countPaymentOrdersByStatus(ctx, OrderStatusRefundRequested)
	if err != nil {
		return stats, err
	}
	stats.Refunding, err = s.countPaymentOrdersByStatus(ctx, OrderStatusRefunding)
	if err != nil {
		return stats, err
	}
	stats.RefundFailed, err = s.countPaymentOrdersByStatus(ctx, OrderStatusRefundFailed)
	if err != nil {
		return stats, err
	}
	stats.Refunded, err = s.entClient.PaymentOrder.Query().
		Where(paymentorder.StatusIn(OrderStatusRefunded, OrderStatusPartiallyRefunded)).
		Count(ctx)
	if err != nil {
		return stats, err
	}

	stats.FulfillmentFailed, err = s.countPaymentAuditActions(ctx, since, "FULFILLMENT_FAILED")
	if err != nil {
		return stats, err
	}
	stats.PaidNotCompleted, err = s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.StatusIn(OrderStatusPaid, OrderStatusRecharging),
			paymentorder.PaidAtLT(paidNotCompletedBefore),
			paymentorder.CompletedAtIsNil(),
		).
		Count(ctx)
	if err != nil {
		return stats, err
	}
	stats.StalePending, err = s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.StatusEQ(OrderStatusPending),
			paymentorder.ExpiresAtLT(stalePendingBefore),
		).
		Count(ctx)
	if err != nil {
		return stats, err
	}

	stats.EnabledProviderInstances, err = s.entClient.PaymentProviderInstance.Query().
		Where(paymentproviderinstance.EnabledEQ(true)).
		Count(ctx)
	if err != nil {
		return stats, err
	}
	stats.DisabledProviderInstances = stats.ProviderUnavailable
	stats.RefundEnabledProviderInstances, err = s.entClient.PaymentProviderInstance.Query().
		Where(paymentproviderinstance.RefundEnabledEQ(true)).
		Count(ctx)
	if err != nil {
		return stats, err
	}
	stats.UserRefundEnabledProviderInstances, err = s.entClient.PaymentProviderInstance.Query().
		Where(paymentproviderinstance.AllowUserRefundEQ(true)).
		Count(ctx)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func (s *PaymentService) countPaymentAuditActions(ctx context.Context, since time.Time, actions ...string) (int, error) {
	if len(actions) == 0 {
		return 0, nil
	}
	return s.entClient.PaymentAuditLog.Query().
		Where(
			paymentauditlog.ActionIn(actions...),
			paymentauditlog.CreatedAtGTE(since),
		).
		Count(ctx)
}

func (s *PaymentService) countPaymentOrdersByStatus(ctx context.Context, status string) (int, error) {
	return s.entClient.PaymentOrder.Query().
		Where(paymentorder.StatusEQ(status)).
		Count(ctx)
}

func computeBasicStats(st *DashboardStats, orders []*dbent.PaymentOrder, todayStart time.Time) {
	var totalAmount, todayAmount float64
	var todayCount int
	for _, o := range orders {
		totalAmount += o.PayAmount
		if o.PaidAt != nil && !o.PaidAt.Before(todayStart) {
			todayAmount += o.PayAmount
			todayCount++
		}
	}
	st.TotalAmount = math.Round(totalAmount*100) / 100
	st.TodayAmount = math.Round(todayAmount*100) / 100
	st.TotalCount = len(orders)
	st.TodayCount = todayCount
	if st.TotalCount > 0 {
		st.AvgAmount = math.Round(totalAmount/float64(st.TotalCount)*100) / 100
	}
}

func buildDailySeries(orders []*dbent.PaymentOrder, since time.Time, days int) []DailyStats {
	dailyMap := make(map[string]*DailyStats)
	for _, o := range orders {
		if o.PaidAt == nil {
			continue
		}
		date := o.PaidAt.Format("2006-01-02")
		ds, ok := dailyMap[date]
		if !ok {
			ds = &DailyStats{Date: date}
			dailyMap[date] = ds
		}
		ds.Amount += o.PayAmount
		ds.Count++
	}
	series := make([]DailyStats, 0, days)
	for i := 0; i < days; i++ {
		date := since.AddDate(0, 0, i+1).Format("2006-01-02")
		if ds, ok := dailyMap[date]; ok {
			ds.Amount = math.Round(ds.Amount*100) / 100
			series = append(series, *ds)
		} else {
			series = append(series, DailyStats{Date: date})
		}
	}
	return series
}

func buildMethodDistribution(orders []*dbent.PaymentOrder) []PaymentMethodStat {
	methodMap := make(map[string]*PaymentMethodStat)
	for _, o := range orders {
		ms, ok := methodMap[o.PaymentType]
		if !ok {
			ms = &PaymentMethodStat{Type: o.PaymentType}
			methodMap[o.PaymentType] = ms
		}
		ms.Amount += o.PayAmount
		ms.Count++
	}
	methods := make([]PaymentMethodStat, 0, len(methodMap))
	for _, ms := range methodMap {
		ms.Amount = math.Round(ms.Amount*100) / 100
		methods = append(methods, *ms)
	}
	return methods
}

func buildTopUsers(orders []*dbent.PaymentOrder) []TopUserStat {
	userMap := make(map[int64]*TopUserStat)
	for _, o := range orders {
		us, ok := userMap[o.UserID]
		if !ok {
			us = &TopUserStat{UserID: o.UserID, Email: o.UserEmail}
			userMap[o.UserID] = us
		}
		us.Amount += o.PayAmount
	}
	userList := make([]*TopUserStat, 0, len(userMap))
	for _, us := range userMap {
		us.Amount = math.Round(us.Amount*100) / 100
		userList = append(userList, us)
	}
	sort.Slice(userList, func(i, j int) bool {
		return userList[i].Amount > userList[j].Amount
	})
	limit := topUsersLimit
	if len(userList) < limit {
		limit = len(userList)
	}
	result := make([]TopUserStat, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, *userList[i])
	}
	return result
}

// --- Audit Logs ---

func (s *PaymentService) writeAuditLog(ctx context.Context, oid int64, action, op string, detail map[string]any) {
	dj, _ := json.Marshal(detail)
	_, err := s.entClient.PaymentAuditLog.Create().SetOrderID(strconv.FormatInt(oid, 10)).SetAction(action).SetDetail(string(dj)).SetOperator(op).Save(ctx)
	if err != nil {
		slog.Error("audit log failed", "orderID", oid, "action", action, "error", err)
	}
}

func (s *PaymentService) GetOrderAuditLogs(ctx context.Context, oid int64) ([]*dbent.PaymentAuditLog, error) {
	return s.entClient.PaymentAuditLog.Query().Where(paymentauditlog.OrderIDEQ(strconv.FormatInt(oid, 10))).Order(paymentauditlog.ByCreatedAt()).All(ctx)
}
