package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

var (
	ErrCheckinDisabled       = infraerrors.Forbidden("CHECKIN_DISABLED", "daily check-in is disabled")
	ErrCheckinAlreadyClaimed = infraerrors.Conflict("CHECKIN_ALREADY_CLAIMED", "daily check-in already claimed")
)

type UserCheckin struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	CheckinDate  string    `json:"checkin_date"`
	RewardAmount float64   `json:"reward_amount"`
	BalanceAfter float64   `json:"balance_after"`
	Source       string    `json:"source"`
	CreatedAt    time.Time `json:"created_at"`
}

type CheckinStatus struct {
	Enabled         bool         `json:"enabled"`
	RewardAmount    float64      `json:"reward_amount"`
	CheckedIn       bool         `json:"checked_in"`
	CheckedInToday  bool         `json:"checked_in_today"`
	CheckinDate     string       `json:"checkin_date"`
	NextAvailableAt time.Time    `json:"next_available_at"`
	NextCheckinAt   time.Time    `json:"next_checkin_at"`
	LastCheckinAt   *time.Time   `json:"last_checkin_at,omitempty"`
	StreakDays      int          `json:"streak_days"`
	Balance         *float64     `json:"balance,omitempty"`
	Message         string       `json:"message,omitempty"`
	AwardedAmount   *float64     `json:"awarded_amount,omitempty"`
	NewBalance      *float64     `json:"new_balance,omitempty"`
	Record          *UserCheckin `json:"record,omitempty"`
}

type CheckinRepository interface {
	GetByUserAndDate(ctx context.Context, userID int64, checkinDate string) (*UserCheckin, error)
	CountCurrentStreak(ctx context.Context, userID int64, startDate string) (int, error)
	Claim(ctx context.Context, userID int64, checkinDate string, rewardAmount float64) (*UserCheckin, error)
}

type CheckinService struct {
	repo               CheckinRepository
	settingService     *SettingService
	authInvalidator    APIKeyAuthCacheInvalidator
	billingInvalidator BillingCache
}

func NewCheckinService(repo CheckinRepository, settingService *SettingService, authInvalidator APIKeyAuthCacheInvalidator, billingInvalidator BillingCache) *CheckinService {
	return &CheckinService{
		repo:               repo,
		settingService:     settingService,
		authInvalidator:    authInvalidator,
		billingInvalidator: billingInvalidator,
	}
}

func (s *CheckinService) Status(ctx context.Context, userID int64, userTZ string) (*CheckinStatus, error) {
	settings, err := s.settingService.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}
	checkinDate, nextAvailableAt := checkinDateWindow(userTZ)
	status := &CheckinStatus{
		Enabled:         settings.CheckinEnabled,
		RewardAmount:    settings.CheckinRewardAmount,
		CheckinDate:     checkinDate,
		NextAvailableAt: nextAvailableAt,
		NextCheckinAt:   nextAvailableAt,
	}
	if !settings.CheckinEnabled {
		return status, nil
	}
	record, err := s.repo.GetByUserAndDate(ctx, userID, checkinDate)
	if err != nil {
		return nil, err
	}
	if record != nil {
		status.CheckedIn = true
		status.CheckedInToday = true
		status.LastCheckinAt = &record.CreatedAt
		status.Balance = &record.BalanceAfter
		status.Record = record
	}
	streakStartDate := checkinDate
	if record == nil {
		streakStartDate = previousCheckinDate(checkinDate)
	}
	streak, err := s.repo.CountCurrentStreak(ctx, userID, streakStartDate)
	if err != nil {
		return nil, err
	}
	status.StreakDays = streak
	return status, nil
}

func (s *CheckinService) Claim(ctx context.Context, userID int64, userTZ string) (*CheckinStatus, error) {
	settings, err := s.settingService.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.CheckinEnabled {
		return nil, ErrCheckinDisabled
	}
	checkinDate, nextAvailableAt := checkinDateWindow(userTZ)
	record, err := s.repo.Claim(ctx, userID, checkinDate, settings.CheckinRewardAmount)
	if err != nil {
		return nil, err
	}
	s.invalidateUserBalance(ctx, userID)
	streak, err := s.repo.CountCurrentStreak(ctx, userID, checkinDate)
	if err != nil {
		return nil, err
	}
	awarded := settings.CheckinRewardAmount
	newBalance := record.BalanceAfter
	return &CheckinStatus{
		Enabled:         true,
		RewardAmount:    settings.CheckinRewardAmount,
		CheckedIn:       true,
		CheckedInToday:  true,
		CheckinDate:     checkinDate,
		NextAvailableAt: nextAvailableAt,
		NextCheckinAt:   nextAvailableAt,
		LastCheckinAt:   &record.CreatedAt,
		StreakDays:      streak,
		Balance:         &record.BalanceAfter,
		Message:         "签到成功",
		AwardedAmount:   &awarded,
		NewBalance:      &newBalance,
		Record:          record,
	}, nil
}

func (s *CheckinService) invalidateUserBalance(ctx context.Context, userID int64) {
	if s.authInvalidator != nil {
		s.authInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingInvalidator == nil {
		return
	}
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.billingInvalidator.InvalidateUserBalance(cacheCtx, userID)
	}()
}

func checkinDateWindow(userTZ string) (string, time.Time) {
	now := timezone.NowInUserLocation(userTZ)
	checkinDate := now.Format("2006-01-02")
	next := timezone.StartOfDayInUserLocation(now.AddDate(0, 0, 1), userTZ)
	return checkinDate, next
}

func previousCheckinDate(checkinDate string) string {
	t, err := time.Parse("2006-01-02", checkinDate)
	if err != nil {
		return checkinDate
	}
	return t.AddDate(0, 0, -1).Format("2006-01-02")
}
