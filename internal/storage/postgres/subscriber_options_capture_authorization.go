package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

// RequestSubscriberOptionsCaptureAuthorization records a pending review only.
// It has no provider client and cannot create a capture authorization.
func (r *Repository) RequestSubscriberOptionsCaptureAuthorization(ctx context.Context, request storage.SubscriberOptionsCaptureAuthorizationRequest) (storage.SubscriberOptionsCaptureAuthorizationRequest, error) {
	request.AuthorizationRequestID, request.CapturePlanID, request.RequestReason, request.RequestedBy, request.CorrelationID = strings.TrimSpace(request.AuthorizationRequestID), strings.TrimSpace(request.CapturePlanID), strings.TrimSpace(request.RequestReason), strings.TrimSpace(request.RequestedBy), strings.TrimSpace(request.CorrelationID)
	if request.AuthorizationRequestID == "" {
		request.AuthorizationRequestID = newSubscriberID("suboptcaptureapproval")
	}
	if request.CapturePlanID == "" || request.RequestedBy != "subscriber-options-capture" || request.CorrelationID == "" || len(request.RequestReason) == 0 || len(request.RequestReason) > 500 {
		return request, errors.New("invalid Options capture authorization request")
	}
	if request.RequestedAt.IsZero() {
		request.RequestedAt = time.Now().UTC()
	}
	provenance, err := json.Marshal(map[string]any{"schema_version": storage.SubscriberOptionsCaptureAuthorizationRequestVersion, "provider_execution_enabled": false, "request_state": "pending_approval", "provider_request_budget": 1})
	if err != nil {
		return request, fmt.Errorf("encode Options capture approval request: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO subscriber_options_capture_authorization_requests (authorization_request_id,capture_plan_id,request_version,requested_worker_identity,requested_provider,requested_provider_budget,request_state,requested_by,request_reason,correlation_id,request_provenance,requested_at) VALUES ($1,$2,$3,'subscriber-options-capture','massive',1,'pending_approval',$4,$5,$6,$7::jsonb,$8)`, request.AuthorizationRequestID, request.CapturePlanID, storage.SubscriberOptionsCaptureAuthorizationRequestVersion, request.RequestedBy, request.RequestReason, request.CorrelationID, string(provenance), request.RequestedAt.UTC())
	if err != nil {
		return request, fmt.Errorf("insert Options capture authorization request: %w", err)
	}
	return request, nil
}
