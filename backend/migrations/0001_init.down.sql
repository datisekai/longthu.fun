-- 0001_init.down.sql — Drop all 12 tables in reverse FK dependency order.
-- IF EXISTS for idempotency (re-running down after a partial up).

SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS telegram_messages;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS payment_intents;
DROP TABLE IF EXISTS session_charges;
DROP TABLE IF EXISTS session_participants;
DROP TABLE IF EXISTS session_cost_items;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS players;
DROP TABLE IF EXISTS `groups`;
DROP TABLE IF EXISTS bank_accounts;
DROP TABLE IF EXISTS users;

SET FOREIGN_KEY_CHECKS = 1;
