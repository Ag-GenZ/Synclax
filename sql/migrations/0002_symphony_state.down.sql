BEGIN;

DROP INDEX IF EXISTS symphony_completed_attempts_created_at_idx;
DROP TABLE IF EXISTS symphony_completed_attempts;
DROP TABLE IF EXISTS symphony_state;

COMMIT;

