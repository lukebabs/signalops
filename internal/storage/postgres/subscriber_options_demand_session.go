package postgres

import (
	"context"
	"fmt"
)

// AssumeSubscriberOptionsDemandRole makes the only role transition permitted
// by the S6 planner. The login remains NOINHERIT and cannot select an
// arbitrary database role through this API.
func (r *Repository) AssumeSubscriberOptionsDemandRole(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, "SET ROLE signalops_subscriber_options_demand"); err != nil {
		return fmt.Errorf("assume subscriber options demand role: %w", err)
	}
	return nil
}
