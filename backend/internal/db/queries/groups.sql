-- name: GetGroupByIDForHost :one
-- Host-scoped lookup. Returning sql.ErrNoRows for both "missing" and
-- "belongs to another host" preserves tenant isolation.
SELECT id, host_user_id, name, slug, default_bank_account_id, telegram_chat_id,
       privacy_mode, auto_detect_enabled, auto_detect_bank_account_id,
       archived_at, created_at, updated_at
FROM `groups`
WHERE id = ? AND host_user_id = ?
LIMIT 1;
