-- 优化账号分组调度索引
-- 添加复合索引以加速 account_groups 表的查询

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_account_groups_group_priority_account
    ON account_groups (group_id, priority, account_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_account_groups_account_priority_group
    ON account_groups (account_id, priority, group_id);
