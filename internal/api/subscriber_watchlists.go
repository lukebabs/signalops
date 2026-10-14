package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberWatchlistRequest struct {
	ListName      string          `json:"list_name"`
	ListKind      string          `json:"list_kind"`
	GlobalAssetID string          `json:"global_asset_id"`
	CorrelationID string          `json:"correlation_id"`
	Provenance    json.RawMessage `json:"provenance"`
}

func registerSubscriberWatchlistRoutes(mux *http.ServeMux, cfg RouterConfig) {
	requireScope := func(w http.ResponseWriter, r *http.Request) (string, string, bool) {
		tenantID, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return "", "", false
		}
		if _, enabled := cfg.SubscriberListsPilotTenants[tenantID]; !enabled {
			writeError(w, http.StatusNotFound, "subscriber_lists_not_enabled", "subscriber lists are not enabled for this tenant")
			return "", "", false
		}
		subject, ok := requireRequestSubject(w, r, "")
		if !ok || strings.TrimSpace(subject) == "" {
			if ok {
				writeError(w, http.StatusForbidden, "subject_required", "authenticated subject identity is required")
			}
			return "", "", false
		}
		return tenantID, subject, true
	}
	repo := func(w http.ResponseWriter) (storage.SubscriberWatchlistRepository, bool) {
		if cfg.SubscriberWatchlistRepository == nil {
			writeError(w, http.StatusServiceUnavailable, "subscriber_lists_unavailable", "subscriber list storage is unavailable")
			return nil, false
		}
		return cfg.SubscriberWatchlistRepository, true
	}

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/subscriber/lists", func(w http.ResponseWriter, r *http.Request) {
		tenantID, subject, ok := requireScope(w, r)
		if !ok {
			return
		}
		store, ok := repo(w)
		if !ok {
			return
		}
		records, err := store.ListSubscriberWatchlists(r.Context(), tenantID, subject)
		if err != nil {
			writeQueryError(w, err, "subscriber_list_not_found", "subscriber list was not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"lists": subscriberWatchlistResponses(records)})
	})

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/subscriber/lists/{list_id}/memberships", func(w http.ResponseWriter, r *http.Request) {
		tenantID, subject, ok := requireScope(w, r)
		if !ok {
			return
		}
		store, ok := repo(w)
		if !ok {
			return
		}
		records, err := store.ListSubscriberWatchlistMemberships(r.Context(), tenantID, subject, r.PathValue("list_id"))
		if err != nil {
			writeQueryError(w, err, "subscriber_list_not_found", "subscriber list was not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"memberships": subscriberWatchlistMembershipResponses(records)})
	})

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/subscriber/lists/{list_id}/items", func(w http.ResponseWriter, r *http.Request) {
		tenantID, subject, ok := requireScope(w, r)
		if !ok {
			return
		}
		store, ok := repo(w)
		if !ok {
			return
		}
		items, err := store.ListSubscriberWatchlistItems(r.Context(), tenantID, subject, r.PathValue("list_id"))
		if err != nil {
			writeQueryError(w, err, "subscriber_list_not_found", "subscriber list was not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": subscriberWatchlistItemResponses(items)})
	})

	mux.HandleFunc("POST /v1/tenants/{tenant_id}/marketops/subscriber/private-lists", func(w http.ResponseWriter, r *http.Request) {
		tenantID, subject, ok := requireScope(w, r)
		if !ok {
			return
		}
		request, ok := readSubscriberWatchlistRequest(w, r)
		if !ok {
			return
		}
		store, ok := repo(w)
		if !ok {
			return
		}
		record, err := store.CreateSubscriberPrivateWatchlist(r.Context(), storage.SubscriberWatchlistCreateRequest{
			TenantID: tenantID, ListName: request.ListName, ActorSubject: subject, CorrelationID: request.CorrelationID, ProvenanceJSON: request.Provenance,
		})
		if errors.Is(err, storage.ErrConflict) {
			writeError(w, http.StatusConflict, "subscriber_list_conflict", "subscriber list already exists")
			return
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_subscriber_list", "a valid private list name is required")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"list": subscriberWatchlistResponse(record)})
	})

	mux.HandleFunc("POST /v1/tenants/{tenant_id}/marketops/subscriber/tenant-default-list", func(w http.ResponseWriter, r *http.Request) {
		tenantID, subject, ok := requireScope(w, r)
		if !ok || !requireTenantAdministrator(w, r) {
			return
		}
		request, ok := readSubscriberWatchlistRequest(w, r)
		if !ok {
			return
		}
		store, ok := repo(w)
		if !ok {
			return
		}
		record, err := store.CreateSubscriberTenantDefaultWatchlist(r.Context(), storage.SubscriberWatchlistCreateRequest{
			TenantID: tenantID, ListName: request.ListName, ActorSubject: subject, CorrelationID: request.CorrelationID, ProvenanceJSON: request.Provenance,
		})
		if errors.Is(err, storage.ErrConflict) {
			writeError(w, http.StatusConflict, "subscriber_list_conflict", "tenant default list already exists")
			return
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_subscriber_list", "a valid tenant default list name is required")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"list": subscriberWatchlistResponse(record)})
	})

	mux.HandleFunc("POST /v1/tenants/{tenant_id}/marketops/subscriber/lists/{list_id}/memberships", func(w http.ResponseWriter, r *http.Request) {
		tenantID, subject, ok := requireScope(w, r)
		if !ok {
			return
		}
		request, ok := readSubscriberWatchlistRequest(w, r)
		if !ok {
			return
		}
		store, ok := repo(w)
		if !ok {
			return
		}
		input := storage.SubscriberWatchlistMembershipRequest{TenantID: tenantID, ListID: r.PathValue("list_id"), GlobalAssetID: request.GlobalAssetID, ActorSubject: subject, CorrelationID: request.CorrelationID, ProvenanceJSON: request.Provenance}
		var record storage.SubscriberWatchlistMembershipRecord
		var err error
		switch strings.TrimSpace(request.ListKind) {
		case storage.SubscriberWatchlistKindPrivate:
			record, err = store.AddSubscriberPrivateWatchlistMembership(r.Context(), input)
		case storage.SubscriberWatchlistKindTenantDefault:
			if !requireTenantAdministrator(w, r) {
				return
			}
			record, err = store.AddSubscriberTenantDefaultWatchlistMembership(r.Context(), input)
		default:
			writeError(w, http.StatusBadRequest, "invalid_subscriber_list_kind", "list_kind must be private or tenant_default")
			return
		}
		if err != nil {
			writeSubscriberWatchlistMutationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"membership": subscriberWatchlistMembershipResponse(record)})
	})
	mux.HandleFunc("DELETE /v1/tenants/{tenant_id}/marketops/subscriber/lists/{list_id}/memberships/{global_asset_id}", func(w http.ResponseWriter, r *http.Request) {
		tenantID, subject, ok := requireScope(w, r)
		if !ok {
			return
		}
		store, ok := repo(w)
		if !ok {
			return
		}
		input := storage.SubscriberWatchlistMembershipRequest{TenantID: tenantID, ListID: r.PathValue("list_id"), GlobalAssetID: r.PathValue("global_asset_id"), ActorSubject: subject, CorrelationID: strings.TrimSpace(r.URL.Query().Get("correlation_id"))}
		var err error
		switch strings.TrimSpace(r.URL.Query().Get("list_kind")) {
		case storage.SubscriberWatchlistKindPrivate:
			err = store.RemoveSubscriberPrivateWatchlistMembership(r.Context(), input)
		case storage.SubscriberWatchlistKindTenantDefault:
			if !requireTenantAdministrator(w, r) {
				return
			}
			err = store.RemoveSubscriberTenantDefaultWatchlistMembership(r.Context(), input)
		default:
			writeError(w, http.StatusBadRequest, "invalid_subscriber_list_kind", "list_kind must be private or tenant_default")
			return
		}
		if err != nil {
			writeSubscriberWatchlistMutationError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func readSubscriberWatchlistRequest(w http.ResponseWriter, r *http.Request) (subscriberWatchlistRequest, bool) {
	var request subscriberWatchlistRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be a valid subscriber list object")
		return request, false
	}
	return request, true
}

func writeSubscriberWatchlistMutationError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "subscriber_list_not_found", "subscriber list was not found")
		return
	}
	if errors.Is(err, storage.ErrConflict) {
		writeError(w, http.StatusConflict, "subscriber_list_conflict", "subscriber list already exists")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid_subscriber_membership", "a valid authorized list and global asset id are required")
}

func subscriberWatchlistResponses(records []storage.SubscriberWatchlistRecord) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, subscriberWatchlistResponse(record))
	}
	return out
}

func subscriberWatchlistResponse(record storage.SubscriberWatchlistRecord) map[string]any {
	return map[string]any{"list_id": record.ListID, "tenant_id": record.TenantID, "list_kind": record.ListKind, "owner_subject": record.OwnerSubject, "list_name": record.ListName, "created_at": record.CreatedAt, "updated_at": record.UpdatedAt}
}

func subscriberWatchlistItemResponses(records []storage.SubscriberWatchlistItemRecord) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, map[string]any{"tenant_id": record.TenantID, "list_id": record.ListID, "list_kind": record.ListKind, "list_name": record.ListName, "global_asset_id": record.GlobalAssetID, "ticker": record.Ticker, "company_name": record.CompanyName, "asset_type": record.AssetType, "exchange": record.Exchange, "sector": record.Sector, "eligibility_status": record.EligibilityStatus, "coverage_state": record.CoverageState, "coverage_mode": record.CoverageMode, "added_at": record.AddedAt})
	}
	return out
}

func subscriberWatchlistMembershipResponses(records []storage.SubscriberWatchlistMembershipRecord) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, subscriberWatchlistMembershipResponse(record))
	}
	return out
}

func subscriberWatchlistMembershipResponse(record storage.SubscriberWatchlistMembershipRecord) map[string]any {
	return map[string]any{"tenant_id": record.TenantID, "list_id": record.ListID, "global_asset_id": record.GlobalAssetID, "added_by_subject": record.AddedBySubject, "added_at": record.AddedAt, "updated_at": record.UpdatedAt}
}
