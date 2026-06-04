package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type checkinRepository struct {
	db *sql.DB
}

func NewCheckinRepository(_ *dbent.Client, sqlDB *sql.DB) service.CheckinRepository {
	return &checkinRepository{db: sqlDB}
}

func (r *checkinRepository) GetByUserAndDate(ctx context.Context, userID int64, checkinDate string) (*service.UserCheckin, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	var record service.UserCheckin
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, checkin_date::text, reward_amount, balance_after, source, created_at
		FROM user_checkins
		WHERE user_id = $1 AND checkin_date = $2::date
	`, userID, checkinDate).Scan(
		&record.ID,
		&record.UserID,
		&record.CheckinDate,
		&record.RewardAmount,
		&record.BalanceAfter,
		&record.Source,
		&record.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *checkinRepository) CountCurrentStreak(ctx context.Context, userID int64, startDate string) (int, error) {
	if r == nil || r.db == nil {
		return 0, nil
	}
	var streak int
	err := r.db.QueryRowContext(ctx, `
		WITH ordered AS (
			SELECT checkin_date,
			       ROW_NUMBER() OVER (ORDER BY checkin_date DESC) AS rn
			FROM user_checkins
			WHERE user_id = $1
			  AND checkin_date <= $2::date
		)
		SELECT COUNT(*)
		FROM ordered
		WHERE checkin_date = ($2::date - ((rn - 1) * INTERVAL '1 day'))::date
	`, userID, startDate).Scan(&streak)
	if err != nil {
		return 0, err
	}
	return streak, nil
}

func (r *checkinRepository) Claim(ctx context.Context, userID int64, checkinDate string, rewardAmount float64) (*service.UserCheckin, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("checkin repository is not configured")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var id int64
	var createdAt time.Time
	err = tx.QueryRowContext(ctx, `
		INSERT INTO user_checkins (user_id, checkin_date, reward_amount, balance_after, source)
		VALUES ($1, $2::date, $3, 0, 'daily_checkin')
		ON CONFLICT (user_id, checkin_date) DO NOTHING
		RETURNING id, created_at
	`, userID, checkinDate, rewardAmount).Scan(&id, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrCheckinAlreadyClaimed
	}
	if err != nil {
		return nil, err
	}

	var balanceAfter float64
	err = tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING balance
	`, rewardAmount, userID).Scan(&balanceAfter)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE user_checkins
		SET balance_after = $1
		WHERE id = $2
	`, balanceAfter, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &service.UserCheckin{
		ID:           id,
		UserID:       userID,
		CheckinDate:  checkinDate,
		RewardAmount: rewardAmount,
		BalanceAfter: balanceAfter,
		Source:       "daily_checkin",
		CreatedAt:    createdAt,
	}, nil
}
