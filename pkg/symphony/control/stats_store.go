package control

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/wibus-wee/synclax/pkg/symphony/orchestrator"
	"github.com/wibus-wee/synclax/pkg/zcore/model"
	"github.com/wibus-wee/synclax/pkg/zgen/querier"

	"github.com/jackc/pgx/v5"
)

type dbStatsStore struct {
	model model.ModelInterface
}

func NewDBStatsStore(m model.ModelInterface) orchestrator.StatsStore {
	if m == nil {
		return nil
	}
	return &dbStatsStore{model: m}
}

func (s *dbStatsStore) Load(ctx context.Context, maxAttempts int) (orchestrator.CodexTotals, map[string]any, []orchestrator.CompletedEntry, error) {
	if s == nil || s.model == nil {
		return orchestrator.CodexTotals{}, nil, nil, errors.New("nil stats store")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	st, err := s.model.GetSymphonyState(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return orchestrator.CodexTotals{}, map[string]any{}, []orchestrator.CompletedEntry{}, nil
		}
		return orchestrator.CodexTotals{}, nil, nil, err
	}

	totals := orchestrator.CodexTotals{
		InputTokens:    int(st.CodexInputTokens),
		OutputTokens:   int(st.CodexOutputTokens),
		TotalTokens:    int(st.CodexTotalTokens),
		SecondsRunning: 0,
	}

	rateLimits := map[string]any{}
	if len(st.RateLimits) > 0 {
		_ = json.Unmarshal(st.RateLimits, &rateLimits)
	}

	limit := maxAttempts
	if limit <= 0 {
		limit = 200
	}

	rawEntries, err := s.model.ListSymphonyCompletedAttempts(ctx, int32(limit))
	if err != nil {
		return totals, rateLimits, nil, err
	}
	entries := make([]orchestrator.CompletedEntry, 0, len(rawEntries))
	for _, row := range rawEntries {
		var entry orchestrator.CompletedEntry
		if err := json.Unmarshal(row.Entry, &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	// Keep chronological order (oldest -> newest) to match in-memory append semantics.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return totals, rateLimits, entries, nil
}

func (s *dbStatsStore) Record(ctx context.Context, totals orchestrator.CodexTotals, rateLimits map[string]any, entry orchestrator.CompletedEntry) error {
	if s == nil || s.model == nil {
		return errors.New("nil stats store")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	rateLimitsJSON := json.RawMessage(`{}`)
	if rateLimits != nil {
		if b, err := json.Marshal(rateLimits); err == nil {
			rateLimitsJSON = b
		}
	}

	// Do not persist uptime in DB; it's derived from process start time.
	params := querier.UpsertSymphonyStateParams{
		CodexInputTokens:  int64(totals.InputTokens),
		CodexOutputTokens: int64(totals.OutputTokens),
		CodexTotalTokens:  int64(totals.TotalTokens),
		RateLimits:        rateLimitsJSON,
	}

	return s.model.RunTransaction(ctx, func(m model.ModelInterface) error {
		if err := m.InsertSymphonyCompletedAttempt(ctx, entryJSON); err != nil {
			return err
		}
		if err := m.UpsertSymphonyState(ctx, params); err != nil {
			return err
		}
		// Keep only the last N rows to avoid unbounded growth.
		_ = m.PruneSymphonyCompletedAttempts(ctx, int32(200))
		return nil
	})
}
