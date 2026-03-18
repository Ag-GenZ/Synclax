-- name: GetSymphonyState :one
SELECT
    id,
    codex_input_tokens,
    codex_output_tokens,
    codex_total_tokens,
    rate_limits,
    updated_at
FROM symphony_state
WHERE id = 1;

-- name: UpsertSymphonyState :exec
INSERT INTO symphony_state (
    id,
    codex_input_tokens,
    codex_output_tokens,
    codex_total_tokens,
    rate_limits,
    updated_at
)
VALUES (1, $1, $2, $3, $4, now())
ON CONFLICT (id) DO UPDATE SET
    codex_input_tokens = EXCLUDED.codex_input_tokens,
    codex_output_tokens = EXCLUDED.codex_output_tokens,
    codex_total_tokens = EXCLUDED.codex_total_tokens,
    rate_limits = EXCLUDED.rate_limits,
    updated_at = now();

-- name: InsertSymphonyCompletedAttempt :exec
INSERT INTO symphony_completed_attempts (entry)
VALUES ($1);

-- name: PruneSymphonyCompletedAttempts :exec
DELETE FROM symphony_completed_attempts
WHERE id < (
    SELECT id
    FROM symphony_completed_attempts
    ORDER BY id DESC
    OFFSET GREATEST($1 - 1, 0)
    LIMIT 1
);

-- name: ListSymphonyCompletedAttempts :many
SELECT entry
FROM symphony_completed_attempts
ORDER BY id DESC
LIMIT $1;
