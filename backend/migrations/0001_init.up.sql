-- 0001_init.up.sql — Longthu.fun initial schema (Story 1.2)
--
-- Source of truth: _bmad-output/planning-artifacts/architecture.md
--   §Project Structure & Boundaries → "Complete data schema (12 tables)"
--
-- Conventions (architecture §Implementation Patterns → Naming patterns):
--   * snake_case plural tables, snake_case columns
--   * id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY
--   * timestamps DATETIME(3); created_at default CURRENT_TIMESTAMP(3); updated_at on update
--   * FK columns: <singular>_id; FK constraint names: fk_<table>_<referenced>
--   * Index names: idx_<table>_<columns>; unique names: uk_<table>_<columns>
--   * Money: BIGINT NOT NULL (smallest VND unit, no decimal)
--   * Enums via VARCHAR + CHECK constraint (MySQL 8 enforces CHECK)
--
-- Charset/collation enforced server-wide via docker-compose (--character-set-server=utf8mb4
-- --collation-server=utf8mb4_0900_ai_ci), but every CREATE TABLE re-declares it
-- explicitly so the migration is portable.

SET FOREIGN_KEY_CHECKS = 1;

-- ---------------------------------------------------------------------------
-- 1. users — host accounts
-- ---------------------------------------------------------------------------
CREATE TABLE users (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    email           VARCHAR(254) NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    display_name    VARCHAR(120) NOT NULL,
    tier            VARCHAR(16)  NOT NULL DEFAULT 'free',
    tier_changed_at DATETIME(3)  NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    CONSTRAINT uk_users_email UNIQUE (email),
    CONSTRAINT ck_users_tier CHECK (tier IN ('free', 'pro', 'pro_plus')),
    INDEX idx_users_tier (tier)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- ---------------------------------------------------------------------------
-- 2. bank_accounts — host's recipient accounts
--
-- Partial-unique enforcement of "at most one default per user" uses a
-- generated column trick: when is_default = 1 the generated column equals
-- user_id, otherwise NULL. A UNIQUE index on the generated column allows
-- many rows with default_flag = NULL (= non-default) but only one non-NULL
-- value (= the one default) per user.
-- ---------------------------------------------------------------------------
CREATE TABLE bank_accounts (
    id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id              BIGINT UNSIGNED NOT NULL,
    bank_name            VARCHAR(80)  NOT NULL,
    bank_code            VARCHAR(20)  NOT NULL,
    account_number       VARCHAR(40)  NOT NULL,
    account_holder_name  VARCHAR(120) NOT NULL,
    is_default           TINYINT(1)   NOT NULL DEFAULT 0,
    default_flag         BIGINT UNSIGNED GENERATED ALWAYS AS (IF(is_default = 1, user_id, NULL)) STORED,
    created_at           DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at           DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    CONSTRAINT fk_bank_accounts_users FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT uk_bank_accounts_default_per_user UNIQUE (default_flag),
    INDEX idx_bank_accounts_user (user_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- ---------------------------------------------------------------------------
-- 3. groups — recurring badminton circles
-- ---------------------------------------------------------------------------
CREATE TABLE `groups` (
    id                            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    host_user_id                  BIGINT UNSIGNED NOT NULL,
    name                          VARCHAR(120) NOT NULL,
    slug                          VARCHAR(64)  NULL,
    default_bank_account_id       BIGINT UNSIGNED NULL,
    telegram_chat_id              BIGINT NULL,
    privacy_mode                  VARCHAR(20)  NOT NULL DEFAULT 'public',
    auto_detect_enabled           TINYINT(1)   NOT NULL DEFAULT 0,
    auto_detect_bank_account_id   BIGINT UNSIGNED NULL,
    archived_at                   DATETIME(3)  NULL,
    created_at                    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at                    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    CONSTRAINT fk_groups_users FOREIGN KEY (host_user_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT fk_groups_bank_default FOREIGN KEY (default_bank_account_id) REFERENCES bank_accounts (id) ON DELETE SET NULL ON UPDATE RESTRICT,
    CONSTRAINT fk_groups_bank_autodetect FOREIGN KEY (auto_detect_bank_account_id) REFERENCES bank_accounts (id) ON DELETE SET NULL ON UPDATE RESTRICT,
    CONSTRAINT uk_groups_slug UNIQUE (slug),
    CONSTRAINT ck_groups_privacy_mode CHECK (privacy_mode IN ('public', 'private_leaning')),
    INDEX idx_groups_host (host_user_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- ---------------------------------------------------------------------------
-- 4. players — group members
-- ---------------------------------------------------------------------------
CREATE TABLE players (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    group_id         BIGINT UNSIGNED NOT NULL,
    display_name     VARCHAR(120) NOT NULL,
    public_code      VARCHAR(16)  NOT NULL,
    telegram_user_id BIGINT       NULL,
    is_active        TINYINT(1)   NOT NULL DEFAULT 1,
    created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    CONSTRAINT fk_players_groups FOREIGN KEY (group_id) REFERENCES `groups` (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT uk_players_group_display_name UNIQUE (group_id, display_name),
    CONSTRAINT uk_players_public_code UNIQUE (public_code),
    INDEX idx_players_active (group_id, is_active)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- ---------------------------------------------------------------------------
-- 5. sessions — badminton meetings; status: draft | finalized | archived
--
-- date is stored as DATE in local Asia/Ho_Chi_Minh (per architecture
-- §Operational decisions → Date / time storage).
-- ---------------------------------------------------------------------------
CREATE TABLE sessions (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    group_id            BIGINT UNSIGNED NOT NULL,
    `date`              DATE         NOT NULL,
    title               VARCHAR(120) NULL,
    location            VARCHAR(120) NULL,
    start_time          TIME         NULL,
    end_time            TIME         NULL,
    total_cost          BIGINT       NOT NULL DEFAULT 0,
    share_code          VARCHAR(16)  NULL,
    status              VARCHAR(20)  NOT NULL DEFAULT 'draft',
    created_by_user_id  BIGINT UNSIGNED NOT NULL,
    finalized_at        DATETIME(3)  NULL,
    created_at          DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    CONSTRAINT fk_sessions_groups FOREIGN KEY (group_id) REFERENCES `groups` (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT fk_sessions_users FOREIGN KEY (created_by_user_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT uk_sessions_share_code UNIQUE (share_code),
    CONSTRAINT ck_sessions_status CHECK (status IN ('draft', 'finalized', 'archived')),
    INDEX idx_sessions_group_status (group_id, status)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- ---------------------------------------------------------------------------
-- 6. session_cost_items — line items per session
-- ---------------------------------------------------------------------------
CREATE TABLE session_cost_items (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    session_id            BIGINT UNSIGNED NOT NULL,
    type                  VARCHAR(20)  NOT NULL,
    label                 VARCHAR(80)  NOT NULL,
    amount                BIGINT       NOT NULL,
    notes                 TEXT         NULL,
    is_included_in_split  TINYINT(1)   NOT NULL DEFAULT 1,
    created_at            DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    CONSTRAINT fk_session_cost_items_sessions FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    CONSTRAINT ck_session_cost_items_type CHECK (type IN ('court', 'shuttle', 'water', 'other', 'discount')),
    INDEX idx_session_cost_items_session (session_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- ---------------------------------------------------------------------------
-- 7. session_participants — players in a session with weight
-- ---------------------------------------------------------------------------
CREATE TABLE session_participants (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    session_id  BIGINT UNSIGNED NOT NULL,
    player_id   BIGINT UNSIGNED NOT NULL,
    weight      DECIMAL(4, 2)   NOT NULL DEFAULT 1.00,
    notes       VARCHAR(120)    NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    CONSTRAINT fk_session_participants_sessions FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    CONSTRAINT fk_session_participants_players FOREIGN KEY (player_id) REFERENCES players (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT uk_session_participants UNIQUE (session_id, player_id),
    INDEX idx_session_participants_player (player_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- ---------------------------------------------------------------------------
-- 8. session_charges — per-player obligations; status:
--    unpaid | pending_confirmation | suspected | paid | waived
-- ---------------------------------------------------------------------------
CREATE TABLE session_charges (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    session_id  BIGINT UNSIGNED NOT NULL,
    player_id   BIGINT UNSIGNED NOT NULL,
    amount      BIGINT          NOT NULL,
    status      VARCHAR(24)     NOT NULL DEFAULT 'unpaid',
    paid_at     DATETIME(3)     NULL,
    paid_via    VARCHAR(12)     NULL,
    description VARCHAR(255)    NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    CONSTRAINT fk_session_charges_sessions FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT fk_session_charges_players FOREIGN KEY (player_id) REFERENCES players (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT uk_session_charges UNIQUE (session_id, player_id),
    CONSTRAINT ck_session_charges_status CHECK (status IN ('unpaid', 'pending_confirmation', 'suspected', 'paid', 'waived')),
    CONSTRAINT ck_session_charges_paid_via CHECK (paid_via IS NULL OR paid_via IN ('auto', 'manual', 'linked')),
    INDEX idx_session_charges_player_status (player_id, status),
    INDEX idx_session_charges_session_status (session_id, status)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- ---------------------------------------------------------------------------
-- 9. payment_intents — host-anchored payment requests
--
-- code format: LT{6-char Crockford-base32} = 8 chars total, per addendum §C
-- (2026-05-24 update — defensive shortening for payOS 9-char description
-- limit on non-linked accounts). Column allows up to 16 chars for headroom.
--
-- covers_charge_ids_json: JSON array of session_charge IDs this intent covers.
-- ---------------------------------------------------------------------------
CREATE TABLE payment_intents (
    id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    player_id            BIGINT UNSIGNED NOT NULL,
    group_id             BIGINT UNSIGNED NOT NULL,
    session_id           BIGINT UNSIGNED NULL,
    amount               BIGINT          NOT NULL,
    code                 VARCHAR(16)     NOT NULL,
    status               VARCHAR(16)     NOT NULL DEFAULT 'pending',
    provider             VARCHAR(16)     NOT NULL DEFAULT 'payos',
    provider_payment_id  VARCHAR(120)    NULL,
    covers_charge_ids_json JSON          NOT NULL,
    expires_at           DATETIME(3)     NULL,
    created_at           DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at           DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    CONSTRAINT fk_payment_intents_players FOREIGN KEY (player_id) REFERENCES players (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT fk_payment_intents_groups FOREIGN KEY (group_id) REFERENCES `groups` (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT fk_payment_intents_sessions FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT uk_payment_intents_code UNIQUE (code),
    CONSTRAINT ck_payment_intents_status CHECK (status IN ('pending', 'matched', 'expired', 'cancelled')),
    CONSTRAINT ck_payment_intents_provider CHECK (provider IN ('payos', 'manual', 'import')),
    INDEX idx_payment_intents_player_status (player_id, status),
    INDEX idx_payment_intents_status_expires (status, expires_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- ---------------------------------------------------------------------------
-- 10. payments — incoming bank transfers (matched / suspected / unmatched)
--
-- uk_payments_provider_tx is the WEBHOOK IDEMPOTENCY ANCHOR for FR-20.
-- ---------------------------------------------------------------------------
CREATE TABLE payments (
    id                       BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    player_id                BIGINT UNSIGNED NULL,
    amount                   BIGINT          NOT NULL,
    bank_description         VARCHAR(255)    NOT NULL,
    matched_intent_id        BIGINT UNSIGNED NULL,
    status                   VARCHAR(16)     NOT NULL,
    provider                 VARCHAR(16)     NOT NULL DEFAULT 'payos',
    provider_transaction_id  VARCHAR(120)    NOT NULL,
    received_at              DATETIME(3)     NOT NULL,
    raw_payload_json         JSON            NOT NULL,
    created_at               DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at               DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    CONSTRAINT fk_payments_players FOREIGN KEY (player_id) REFERENCES players (id) ON DELETE SET NULL ON UPDATE RESTRICT,
    CONSTRAINT fk_payments_matched_intent FOREIGN KEY (matched_intent_id) REFERENCES payment_intents (id) ON DELETE SET NULL ON UPDATE RESTRICT,
    CONSTRAINT uk_payments_provider_tx UNIQUE (provider, provider_transaction_id),
    CONSTRAINT ck_payments_status CHECK (status IN ('matched', 'suspected', 'unmatched')),
    CONSTRAINT ck_payments_provider CHECK (provider IN ('payos', 'manual_import')),
    INDEX idx_payments_status (status)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- ---------------------------------------------------------------------------
-- 11. telegram_messages — bot-sent messages tracked for future edits
-- ---------------------------------------------------------------------------
CREATE TABLE telegram_messages (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    session_id  BIGINT UNSIGNED NOT NULL,
    chat_id     BIGINT          NOT NULL,
    message_id  BIGINT          NOT NULL,
    type        VARCHAR(20)     NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    CONSTRAINT fk_telegram_messages_sessions FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    CONSTRAINT ck_telegram_messages_type CHECK (type IN ('session_bill', 'reminder')),
    INDEX idx_telegram_messages_session (session_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- ---------------------------------------------------------------------------
-- 12. audit_log — append-only payment-state transitions
--
-- host_user_id NULL for system_webhook / system_cron events.
-- entity_id is intentionally a plain BIGINT (no FK) because the same column
-- references different tables based on entity_type.
-- ---------------------------------------------------------------------------
CREATE TABLE audit_log (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    host_user_id  BIGINT UNSIGNED NULL,
    actor_type    VARCHAR(24)     NOT NULL,
    event_type    VARCHAR(48)     NOT NULL,
    entity_type   VARCHAR(32)     NOT NULL,
    entity_id     BIGINT          NOT NULL,
    before_state  JSON            NULL,
    after_state   JSON            NULL,
    context_json  JSON            NULL,
    occurred_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    CONSTRAINT fk_audit_log_users FOREIGN KEY (host_user_id) REFERENCES users (id) ON DELETE SET NULL ON UPDATE RESTRICT,
    CONSTRAINT ck_audit_log_actor_type CHECK (actor_type IN ('host', 'player_link', 'system', 'system_webhook', 'system_cron', 'admin')),
    INDEX idx_audit_host_event (host_user_id, event_type),
    INDEX idx_audit_entity (entity_type, entity_id),
    INDEX idx_audit_occurred (occurred_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;
