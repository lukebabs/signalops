package storage

import "context"

// SubscriberSubscriptionAdministrationRepository is deliberately distinct from
// read-side subscription resolution. Only a platform subscription administrator
// may invoke these operations; browser subscribers and tenant administrators
// cannot self-upgrade a product or create a tenant contract.
type SubscriberSubscriptionAdministrationRepository interface {
	UpsertSubscriberSubjectSubscription(context.Context, SubscriberSubjectSubscriptionMutation) error
	UpsertSubscriberTenantSubscription(context.Context, SubscriberTenantSubscriptionMutation) error
	UpsertSubscriberSubscriptionSeat(context.Context, SubscriberSubscriptionSeatMutation) error
}

type SubscriberSubjectSubscriptionMutation struct {
	TenantID      string
	Subject       string
	ProductKey    string
	Status        string
	ActorSubject  string
	CorrelationID string
}

type SubscriberTenantSubscriptionMutation struct {
	TenantID      string
	ProductKey    string
	Status        string
	ActorSubject  string
	CorrelationID string
}

type SubscriberSubscriptionSeatMutation struct {
	TenantID      string
	Subject       string
	SeatRole      string
	Status        string
	ActorSubject  string
	CorrelationID string
}
