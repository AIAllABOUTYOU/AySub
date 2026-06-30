-- Migration: Add user daily check-in reward records.
-- Feature is disabled by default through settings and enforced by backend APIs.

CREATE TABLE IF NOT EXISTS user_checkins (
	id BIGSERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	checkin_date DATE NOT NULL,
	reward_amount NUMERIC(20,8) NOT NULL DEFAULT 0,
	balance_after NUMERIC(20,8) NOT NULL DEFAULT 0,
	source VARCHAR(32) NOT NULL DEFAULT 'daily_checkin',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_checkins_user_date
	ON user_checkins(user_id, checkin_date);

CREATE INDEX IF NOT EXISTS idx_user_checkins_user_created_at
	ON user_checkins(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_checkins_created_at
	ON user_checkins(created_at DESC);

COMMENT ON TABLE user_checkins IS 'Daily check-in reward records';
COMMENT ON COLUMN user_checkins.checkin_date IS 'User-local check-in date';
COMMENT ON COLUMN user_checkins.reward_amount IS 'Balance reward granted by this check-in';
COMMENT ON COLUMN user_checkins.balance_after IS 'User balance after reward grant';

INSERT INTO settings (key, value)
VALUES
	('checkin_enabled', 'false'),
	('checkin_reward_amount', '0'),
	('checkin_reward_mode', 'fixed'),
	('checkin_reward_min_amount', '0'),
	('checkin_reward_max_amount', '0')
ON CONFLICT (key) DO NOTHING;
