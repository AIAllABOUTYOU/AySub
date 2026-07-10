-- Add independent Grok video rates and auditable per-second usage metadata.
-- Video price columns are USD/second and intentionally do not inherit image prices.

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS video_rate_independent BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS video_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS video_price_480p DECIMAL(20,8),
    ADD COLUMN IF NOT EXISTS video_price_720p DECIMAL(20,8),
    ADD COLUMN IF NOT EXISTS video_price_1080p DECIMAL(20,8);

COMMENT ON COLUMN groups.video_rate_independent IS '视频生成是否使用独立倍率；false 表示共享分组有效倍率';
COMMENT ON COLUMN groups.video_rate_multiplier IS '视频生成独立倍率，仅 video_rate_independent=true 时生效';
COMMENT ON COLUMN groups.video_price_480p IS '480p 视频生成每秒单价 (USD/s)，Grok 平台使用';
COMMENT ON COLUMN groups.video_price_720p IS '720p 视频生成每秒单价 (USD/s)，Grok 平台使用';
COMMENT ON COLUMN groups.video_price_1080p IS '1080p 视频生成每秒单价 (USD/s)，Grok 平台使用';

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS video_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS video_resolution VARCHAR(10),
    ADD COLUMN IF NOT EXISTS video_duration_seconds INTEGER;

COMMENT ON COLUMN usage_logs.video_count IS '视频生成数量；>0 表示本行是视频生成用量';
COMMENT ON COLUMN usage_logs.video_resolution IS '计费用视频分辨率 480p/720p/1080p';
COMMENT ON COLUMN usage_logs.video_duration_seconds IS '提交时请求的视频时长（秒），按秒计费的乘数';

-- Video rows may keep image_count as a legacy media-unit counter. Exempt them
-- even when a channel intentionally bills the request in token mode.
ALTER TABLE usage_logs
    DROP CONSTRAINT IF EXISTS usage_logs_image_billing_size_check;

ALTER TABLE usage_logs
    ADD CONSTRAINT usage_logs_image_billing_size_check
    CHECK (
        image_count <= 0
        OR billing_mode = 'video'
        OR COALESCE(video_count, 0) > 0
        OR (
            image_size IS NOT NULL
            AND image_size IN ('1K', '2K', '4K', 'mixed')
        )
    ) NOT VALID;
