package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) ListUndeliveredSignalAssertionEvents(ctx context.Context, limit int) ([]storage.SignalAssertionEventRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT event_id,assertion_id,event_type,COALESCE(previous_state,''),COALESCE(new_state,''),reason_code,details,occurred_at,transition_sequence,COALESCE(evaluation_id,''),evaluation_mode,COALESCE(evaluation_run_id,''),idempotency_key,published_at FROM signal_assertion_events WHERE published_at IS NULL ORDER BY occurred_at,event_id LIMIT $1`, clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list signal assurance outbox: %w", err)
	}
	defer rows.Close()
	out := []storage.SignalAssertionEventRecord{}
	for rows.Next() {
		var x storage.SignalAssertionEventRecord
		if err := rows.Scan(&x.EventID, &x.AssertionID, &x.EventType, &x.PreviousState, &x.NewState, &x.ReasonCode, &x.DetailsJSON, &x.OccurredAt, &x.TransitionSequence, &x.EvaluationID, &x.EvaluationMode, &x.EvaluationRunID, &x.IdempotencyKey, &x.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) MarkSignalAssertionEventPublished(ctx context.Context, eventID string, publishedAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE signal_assertion_events SET published_at=$1 WHERE event_id=$2 AND published_at IS NULL`, publishedAt.UTC(), eventID)
	if err != nil {
		return fmt.Errorf("mark signal assurance outbox published: %w", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return storage.ErrNotFound
	}
	return nil
}
