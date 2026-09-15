package storage

import "context"

type SubscriberCatalogMembershipResult struct {
	Membership      SubscriberWatchlistMembershipRecord
	ActivationState string
}
type SubscriberCatalogMembershipRepository interface {
	AddSubscriberPrivateCatalogMembership(context.Context, SubscriberWatchlistMembershipRequest) (SubscriberCatalogMembershipResult, error)
}
