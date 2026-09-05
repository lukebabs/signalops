package storage

import (
	"context"
	"time"
)

const SubscriberOptionsCaptureAuthorizationRequestVersion = "s6-options-capture-approval-request-v1"

type SubscriberOptionsCaptureAuthorizationRequest struct {
	AuthorizationRequestID string
	CapturePlanID          string
	RequestReason          string
	RequestedBy            string
	CorrelationID          string
	RequestedAt            time.Time
}

type SubscriberOptionsCaptureAuthorizationRepository interface {
	RequestSubscriberOptionsCaptureAuthorization(context.Context, SubscriberOptionsCaptureAuthorizationRequest) (SubscriberOptionsCaptureAuthorizationRequest, error)
}
