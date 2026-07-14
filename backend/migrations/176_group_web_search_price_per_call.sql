ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS web_search_price_per_call DECIMAL(20,8);
