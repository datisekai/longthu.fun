-- 0002_add_autodetect.down.sql — Rollback Auto-Detect columns

ALTER TABLE `groups` DROP COLUMN auto_detect_credentials_json;

DROP INDEX idx_groups_autodetect ON `groups`;

ALTER TABLE payments DROP COLUMN host_user_id;

DROP INDEX idx_payments_player_host ON payments;
DROP INDEX idx_payments_host_status ON payments;
