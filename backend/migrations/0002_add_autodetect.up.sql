-- 0002_add_autodetect.sql — Auto-Detect (Story 6.2)
--
-- Adds columns for storing payOS credentials per group.
-- Credentials are stored encrypted (AES-256-GCM) via app-level encryption.
-- The encryption key comes from SECRETS_MASTER_KEY env var.

-- ---------------------------------------------------------------------------
-- Add auto-detect columns to groups table
-- ---------------------------------------------------------------------------
ALTER TABLE `groups`
  ADD COLUMN auto_detect_credentials_json TEXT NULL COMMENT 'Encrypted payOS credentials: {clientId, apiKey, checksumKey}' AFTER auto_detect_bank_account_id;

-- ---------------------------------------------------------------------------
-- Index for fast lookup of groups with auto-detect enabled
-- ---------------------------------------------------------------------------
CREATE INDEX idx_groups_autodetect ON `groups` (host_user_id, auto_detect_enabled);

-- ---------------------------------------------------------------------------
-- Index for payment matching (host via group -> player)
-- ---------------------------------------------------------------------------
CREATE INDEX idx_payments_player_host ON payments (player_id, status);

-- ---------------------------------------------------------------------------
-- Add host_user_id to payments for faster dashboard queries
-- (denormalized for dashboard aggregation; derived from player -> group -> host_user_id)
-- ---------------------------------------------------------------------------
ALTER TABLE payments
  ADD COLUMN host_user_id BIGINT UNSIGNED NULL COMMENT 'Denormalized for dashboard queries';

CREATE INDEX idx_payments_host_status ON payments (host_user_id, status);
