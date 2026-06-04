package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type totpRecoveryCodeRepository struct {
	db *sql.DB
}

func NewTotpRecoveryCodeRepository(db *sql.DB) service.TotpRecoveryCodeRepository {
	return &totpRecoveryCodeRepository{db: db}
}

func (r *totpRecoveryCodeRepository) Replace(ctx context.Context, userID int64, codeHashes []string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("totp recovery code repository is not configured")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_totp_recovery_codes WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, hash := range codeHashes {
		if hash == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_totp_recovery_codes (user_id, code_hash)
			VALUES ($1, $2)
			ON CONFLICT (user_id, code_hash) DO NOTHING
		`, userID, hash); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *totpRecoveryCodeRepository) CountAvailable(ctx context.Context, userID int64) (int, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("totp recovery code repository is not configured")
	}
	var count int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM user_totp_recovery_codes
		WHERE user_id = $1 AND used_at IS NULL
	`, userID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *totpRecoveryCodeRepository) MarkUsed(ctx context.Context, userID int64, codeHash string) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("totp recovery code repository is not configured")
	}
	var id int64
	err := r.db.QueryRowContext(ctx, `
		UPDATE user_totp_recovery_codes
		SET used_at = NOW()
		WHERE id = (
			SELECT id
			FROM user_totp_recovery_codes
			WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL
			ORDER BY id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id
	`, userID, codeHash).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
