package control

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/wibus-wee/synclax/pkg/symphony/orchestrator"
	"github.com/wibus-wee/synclax/pkg/zcore/model"
	"github.com/wibus-wee/synclax/pkg/zgen/querier"

	"github.com/jackc/pgx/v5"
)

type dbStatsStore struct {
	model      model.ModelInterface
	workflowID string
}

func NewDBStatsStore(m model.ModelInterface, workflowID string) orchestrator.StatsStore {
	if m == nil {
		return nil
	}
	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		workflowID = "default"
	}
	return &dbStatsStore{model: m, workflowID: workflowID}
}

func (s *dbStatsStore) Load(ctx context.Context, maxAttempts int) (orchestrator.AgentTotals, map[string]any, []orchestrator.CompletedEntry, error) {
	if s == nil || s.model == nil {
		return orchestrator.AgentTotals{}, nil, nil, errors.New("nil stats store")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	st, err := s.model.GetSymphonyState(ctx, s.workflowID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return orchestrator.AgentTotals{}, map[string]any{}, []orchestrator.CompletedEntry{}, nil
		}
		return orchestrator.AgentTotals{}, nil, nil, err
	}

	totals := orchestrator.AgentTotals{
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

	rawEntries, err := s.model.ListSymphonyCompletedAttempts(ctx, querier.ListSymphonyCompletedAttemptsParams{
		WorkflowID: s.workflowID,
		Limit:      int32(limit),
	})
	if err != nil {
		return totals, rateLimits, nil, err
	}
	entries := make([]orchestrator.CompletedEntry, 0, len(rawEntries))
	for _, raw := range rawEntries {
		var entry orchestrator.CompletedEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
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

func (s *dbStatsStore) Record(ctx context.Context, totals orchestrator.AgentTotals, rateLimits map[string]any, entry orchestrator.CompletedEntry) error {
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
		WorkflowID:        s.workflowID,
		CodexInputTokens:  int64(totals.InputTokens),
		CodexOutputTokens: int64(totals.OutputTokens),
		CodexTotalTokens:  int64(totals.TotalTokens),
		RateLimits:        rateLimitsJSON,
	}

	return s.model.RunTransaction(ctx, func(m model.ModelInterface) error {
		if err := m.InsertSymphonyCompletedAttempt(ctx, querier.InsertSymphonyCompletedAttemptParams{
			WorkflowID: s.workflowID,
			Entry:      entryJSON,
		}); err != nil {
			return err
		}
		if err := m.UpsertSymphonyState(ctx, params); err != nil {
			return err
		}
		// Keep only the last N rows to avoid unbounded growth.
		_ = m.PruneSymphonyCompletedAttempts(ctx, querier.PruneSymphonyCompletedAttemptsParams{
			WorkflowID: s.workflowID,
			Keep:       int32(200),
		})
		return nil
	})
}
