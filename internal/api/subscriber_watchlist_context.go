package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberWatchlistContext struct {
	TenantID        string
	Subject         string
	SelectionMode   string
	ListID          string
	ListName        string
	SelectionSource string
	Lists           []storage.SubscriberWatchlistRecord
	Items           []storage.SubscriberWatchlistItemRecord
	Tickers         map[string]struct{}
}

type subscriberWatchlistContextRequest struct {
	SelectionMode string          `json:"selection_mode"`
	ListID        string          `json:"list_id"`
	Provenance    json.RawMessage `json:"provenance"`
}

func subscriberWatchlistContextEnabled(cfg RouterConfig, tenantID string) bool {
	_, enabled := cfg.SubscriberListsPilotTenants[tenantID]
	return cfg.SubscriberListsEnabled && cfg.SubscriberWatchlistRepository != nil && enabled
}

func resolveSubscriberWatchlistContext(r *http.Request, cfg RouterConfig, tenantID, subject string) (subscriberWatchlistContext, error) {
	context := subscriberWatchlistContext{TenantID: tenantID, Subject: subject, Tickers: map[string]struct{}{}}
	if !subscriberWatchlistContextEnabled(cfg, tenantID) {
		return context, nil
	}
	lists, err := cfg.SubscriberWatchlistRepository.ListSubscriberWatchlists(r.Context(), tenantID, subject)
	if err != nil {
		return context, err
	}
	context.Lists = lists
	if len(lists) == 0 {
		return context, storage.ErrNotFound
	}
	preference, preferenceErr := cfg.SubscriberWatchlistRepository.GetSubscriberWatchlistContextPreference(r.Context(), tenantID, subject)
	if preferenceErr != nil && !errors.Is(preferenceErr, storage.ErrNotFound) {
		return context, preferenceErr
	}
	context.SelectionSource = "saved"
	if errors.Is(preferenceErr, storage.ErrNotFound) {
		preference.SelectionMode = storage.SubscriberWatchlistContextModeList
		var oldestPrivate *storage.SubscriberWatchlistRecord
		for index := range lists {
			list := &lists[index]
			if list.ListKind == storage.SubscriberWatchlistKindPrivate && (oldestPrivate == nil || list.CreatedAt.Before(oldestPrivate.CreatedAt)) {
				oldestPrivate = list
			}
		}
		if oldestPrivate != nil {
			preference.ListID = oldestPrivate.ListID
		}
		if preference.ListID == "" {
			for _, list := range lists {
				if list.ListKind == storage.SubscriberWatchlistKindTenantDefault {
					preference.ListID = list.ListID
					break
				}
			}
			context.SelectionSource = "tenant_default"
		} else {
			context.SelectionSource = "oldest_private"
		}
	}
	if preference.SelectionMode == storage.SubscriberWatchlistContextModeAll {
		context.SelectionMode = storage.SubscriberWatchlistContextModeAll
		context.SelectionSource = firstNonEmpty(context.SelectionSource, "saved")
	} else {
		for _, list := range lists {
			if list.ListID == preference.ListID {
				context.SelectionMode, context.ListID, context.ListName = storage.SubscriberWatchlistContextModeList, list.ListID, list.ListName
				break
			}
		}
		if context.ListID == "" {
			return subscriberWatchlistContext{}, storage.ErrNotFound
		}
	}
	selected := lists
	if context.SelectionMode == storage.SubscriberWatchlistContextModeList {
		selected = nil
		for _, list := range lists {
			if list.ListID == context.ListID {
				selected = append(selected, list)
			}
		}
	}
	seen := map[string]struct{}{}
	for _, list := range selected {
		items, err := cfg.SubscriberWatchlistRepository.ListSubscriberWatchlistItems(r.Context(), tenantID, subject, list.ListID)
		if err != nil {
			return subscriberWatchlistContext{}, err
		}
		for _, item := range items {
			if _, duplicate := seen[item.GlobalAssetID]; duplicate {
				continue
			}
			seen[item.GlobalAssetID] = struct{}{}
			context.Items = append(context.Items, item)
			if ticker := strings.ToUpper(strings.TrimSpace(item.Ticker)); ticker != "" {
				context.Tickers[ticker] = struct{}{}
			}
		}
	}
	sort.Slice(context.Items, func(i, j int) bool { return context.Items[i].Ticker < context.Items[j].Ticker })
	return context, nil
}

func requireSubscriberWatchlistContext(w http.ResponseWriter, r *http.Request, cfg RouterConfig, tenantID string) (subscriberWatchlistContext, bool) {
	if !subscriberWatchlistContextEnabled(cfg, tenantID) {
		return subscriberWatchlistContext{}, true
	}
	subject, ok := requireRequestSubject(w, r, "")
	if !ok || strings.TrimSpace(subject) == "" {
		if ok {
			writeError(w, http.StatusForbidden, "subject_required", "authenticated subject identity is required")
		}
		return subscriberWatchlistContext{}, false
	}
	context, err := resolveSubscriberWatchlistContext(r, cfg, tenantID, subject)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "subscriber_watchlist_context_not_found", "no authorized watchlist context is available")
		return subscriberWatchlistContext{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "subscriber_watchlist_context_unavailable", "watchlist context could not be resolved")
		return subscriberWatchlistContext{}, false
	}
	return context, true
}

func registerSubscriberWatchlistContextRoutes(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/subscriber/watchlist-context", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		if !subscriberWatchlistContextEnabled(cfg, tenantID) {
			writeError(w, http.StatusNotFound, "subscriber_lists_not_enabled", "subscriber lists are not enabled for this tenant")
			return
		}
		subject, ok := requireRequestSubject(w, r, "")
		if !ok {
			return
		}
		context, err := resolveSubscriberWatchlistContext(r, cfg, tenantID, subject)
		if err != nil {
			writeQueryError(w, err, "subscriber_watchlist_context_not_found", "watchlist context was not found")
			return
		}
		writeJSON(w, http.StatusOK, subscriberWatchlistContextResponse(context))
	})
	mux.HandleFunc("PUT /v1/tenants/{tenant_id}/marketops/subscriber/watchlist-context", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		if !subscriberWatchlistContextEnabled(cfg, tenantID) {
			writeError(w, http.StatusNotFound, "subscriber_lists_not_enabled", "subscriber lists are not enabled for this tenant")
			return
		}
		subject, ok := requireRequestSubject(w, r, "")
		if !ok {
			return
		}
		var request subscriberWatchlistContextRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "request body must be a valid watchlist context object")
			return
		}
		request.SelectionMode, request.ListID = strings.TrimSpace(request.SelectionMode), strings.TrimSpace(request.ListID)
		if request.SelectionMode != storage.SubscriberWatchlistContextModeAll && request.SelectionMode != storage.SubscriberWatchlistContextModeList {
			writeError(w, http.StatusBadRequest, "invalid_watchlist_context", "selection_mode must be list or all")
			return
		}
		if request.SelectionMode == storage.SubscriberWatchlistContextModeAll {
			request.ListID = ""
		}
		lists, err := cfg.SubscriberWatchlistRepository.ListSubscriberWatchlists(r.Context(), tenantID, subject)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "subscriber_watchlist_context_unavailable", "authorized lists could not be read")
			return
		}
		if request.SelectionMode == storage.SubscriberWatchlistContextModeList {
			authorized := false
			for _, list := range lists {
				if list.ListID == request.ListID {
					authorized = true
					break
				}
			}
			if !authorized {
				writeError(w, http.StatusNotFound, "subscriber_list_not_found", "watchlist was not found")
				return
			}
		}
		_, err = cfg.SubscriberWatchlistRepository.SetSubscriberWatchlistContextPreference(r.Context(), storage.SubscriberWatchlistContextPreference{TenantID: tenantID, Subject: subject, SelectionMode: request.SelectionMode, ListID: request.ListID, ProvenanceJSON: request.Provenance})
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_watchlist_context", "watchlist context could not be saved")
			return
		}
		context, err := resolveSubscriberWatchlistContext(r, cfg, tenantID, subject)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "subscriber_watchlist_context_unavailable", "watchlist context could not be resolved")
			return
		}
		writeJSON(w, http.StatusOK, subscriberWatchlistContextResponse(context))
	})
}

func subscriberWatchlistContextResponse(context subscriberWatchlistContext) map[string]any {
	return map[string]any{
		"selection_mode": context.SelectionMode, "list_id": context.ListID, "list_name": context.ListName,
		"selection_source": context.SelectionSource, "lists": subscriberWatchlistResponses(context.Lists),
		"items": subscriberWatchlistItemResponses(context.Items), "member_count": len(context.Items),
	}
}
