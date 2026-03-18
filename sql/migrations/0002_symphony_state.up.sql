BEGIN;

CREATE TABLE IF NOT EXISTS symphony_state (
    id                 INTEGER PRIMARY KEY,
    codex_input_tokens  BIGINT NOT NULL DEFAULT 0,
    codex_output_tokens BIGINT NOT NULL DEFAULT 0,
    codex_total_tokens  BIGINT NOT NULL DEFAULT 0,
    rate_limits         JSONB  NOT NULL DEFAULT '{}'::jsonb,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO symphony_state (id, codex_input_tokens, codex_output_tokens, codex_total_tokens, rate_limits)
VALUES (1, 0, 0, 0, '{}'::jsonb)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS symphony_completed_attempts (
    id         BIGSERIAL PRIMARY KEY,
    entry      JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS symphony_completed_attempts_created_at_idx
    ON symphony_completed_attempts (created_at DESC);

COMMIT;

