BEGIN;

DROP INDEX IF EXISTS symphony_completed_attempts_workflow_created_at_idx;

ALTER TABLE symphony_completed_attempts
    DROP COLUMN IF EXISTS workflow_id;

ALTER TABLE symphony_state
    DROP CONSTRAINT IF EXISTS symphony_state_workflow_id_key;

ALTER TABLE symphony_state
    DROP COLUMN IF EXISTS workflow_id;

COMMIT;

