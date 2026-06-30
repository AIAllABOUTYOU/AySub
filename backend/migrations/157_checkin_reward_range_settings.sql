INSERT INTO settings (key, value)
VALUES
	('checkin_reward_mode', 'fixed'),
	('checkin_reward_min_amount', '0'),
	('checkin_reward_max_amount', '0')
ON CONFLICT (key) DO NOTHING;
