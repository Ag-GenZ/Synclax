-- name: GetSymphonyState :one
SELECT
    id,
    workflow_id,
    codex_input_tokens,
    codex_output_tokens,
    codex_total_tokens,
    rate_limits,
    updated_at
FROM symphony_state
WHERE workflow_id = $1;

-- name: UpsertSymphonyState :exec
INSERT INTO symphony_state (
    workflow_id,
    codex_input_tokens,
    codex_output_tokens,
    codex_total_tokens,
    rate_limits,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (workflow_id) DO UPDATE SET
    codex_input_tokens = EXCLUDED.codex_input_tokens,
    codex_output_tokens = EXCLUDED.codex_output_tokens,
    codex_total_tokens = EXCLUDED.codex_total_tokens,
    rate_limits = EXCLUDED.rate_limits,
    updated_at = now();

-- name: InsertSymphonyCompletedAttempt :exec
INSERT INTO symphony_completed_attempts (workflow_id, entry)
VALUES ($1, $2);

-- name: PruneSymphonyCompletedAttempts :exec
DELETE FROM symphony_completed_attempts sca
WHERE sca.workflow_id = $1
  AND sca.id < (
    SELECT sca2.id
    FROM symphony_completed_attempts sca2
    WHERE sca2.workflow_id = $1
    ORDER BY sca2.id DESC
    OFFSET GREATEST((sqlc.arg(keep))::int - 1, 0)
    LIMIT 1
);

-- name: ListSymphonyCompletedAttempts :many
SELECT entry
FROM symphony_completed_attempts
WHERE workflow_id = $1
ORDER BY id DESC
LIMIT $2;
