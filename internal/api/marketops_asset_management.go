package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/storage"
)

const analystWatchlistGroup = "analyst_watchlist"

var analystTickerPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.-]{0,14}$`)

type marketOpsAssetCreateRequest struct {
	Ticker   string `json:"ticker"`
	Company  string `json:"company"`
	Sector   string `json:"sector"`
	Industry string `json:"industry"`
}

type marketOpsAssetOnboardRequest struct {
	Ticker                string `json:"ticker"`
	BackfillEquityHistory bool   `json:"backfill_equity_history"`
	StartDate             string `json:"start_date"`
	EndDate               string `json:"end_date"`
	RequestedBy           string `json:"requested_by"`
}
type marketOpsAssetBackfillCreateRequest struct {
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	RequestedBy string `json:"requested_by"`
}

type marketOpsAssetDisplayNameRequest struct {
	UniverseGroup string `json:"universe_group"`
	DisplayName   string `json:"display_name"`
}

type marketOpsAssetDisplaySectorRequest struct {
	UniverseGroup string `json:"universe_group"`
	DisplaySector string `json:"display_sector"`
}

func registerMarketOpsAssetManagementRoutes(mux *http.ServeMux, repo any) {
	mux.HandleFunc("PATCH /v1/tenants/{tenant_id}/marketops/assets/{symbol}/display-sector", func(w http.ResponseWriter, r *http.Request) {
		writer, ok := repo.(storage.MarketOpsAssetManagementRepository)
		if !ok {
			writeError(w, http.StatusServiceUnavailable, "asset_management_unavailable", "asset sector editing is unavailable")
			return
		}
		var req marketOpsAssetDisplaySectorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid display-sector request")
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		ticker := strings.ToUpper(strings.TrimSpace(r.PathValue("symbol")))
		displaySector := strings.TrimSpace(req.DisplaySector)
		if tenantID == "" || !analystTickerPattern.MatchString(ticker) || (req.UniverseGroup != "top50_megacap" && req.UniverseGroup != analystWatchlistGroup) || displaySector == "" || len([]rune(displaySector)) > 48 {
			writeError(w, http.StatusBadRequest, "invalid_display_sector", "valid tenant, ticker, universe group, and a sector label of at most 48 characters are required")
			return
		}
		asset, err := writer.UpdateMarketOpsAssetDisplaySector(r.Context(), tenantID, req.UniverseGroup, ticker, displaySector)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeError(w, http.StatusNotFound, "asset_not_found", "asset was not found in the selected universe")
				return
			}
			writeError(w, http.StatusInternalServerError, "asset_write_failed", "asset sector could not be updated")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"asset": marketOpsAssetResponses([]storage.MarketOpsAssetRecord{asset})[0]})
	})
	mux.HandleFunc("PATCH /v1/tenants/{tenant_id}/marketops/assets/{symbol}/display-name", func(w http.ResponseWriter, r *http.Request) {
		writer, ok := repo.(storage.MarketOpsAssetManagementRepository)
		if !ok {
			writeError(w, http.StatusServiceUnavailable, "asset_management_unavailable", "asset display-name editing is unavailable")
			return
		}
		var req marketOpsAssetDisplayNameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid display-name request")
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		ticker := strings.ToUpper(strings.TrimSpace(r.PathValue("symbol")))
		displayName := strings.TrimSpace(req.DisplayName)
		if tenantID == "" || !analystTickerPattern.MatchString(ticker) || (req.UniverseGroup != "top50_megacap" && req.UniverseGroup != analystWatchlistGroup) || len([]rune(displayName)) > 120 {
			writeError(w, http.StatusBadRequest, "invalid_display_name", "valid tenant, ticker, universe group, and a display name of at most 120 characters are required")
			return
		}
		asset, err := writer.UpdateMarketOpsAssetDisplayName(r.Context(), tenantID, req.UniverseGroup, ticker, displayName)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeError(w, http.StatusNotFound, "asset_not_found", "asset was not found in the selected universe")
				return
			}
			writeError(w, http.StatusInternalServerError, "asset_write_failed", "asset display name could not be updated")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"asset": marketOpsAssetResponses([]storage.MarketOpsAssetRecord{asset})[0]})
	})
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/assets/validate", func(w http.ResponseWriter, r *http.Request) {
		validated, err := validateWatchlistTicker(r.Context(), r.URL.Query().Get("ticker"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "ticker_validation_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"validation": validated})
	})
	mux.HandleFunc("POST /v1/tenants/{tenant_id}/marketops/assets/onboard", func(w http.ResponseWriter, r *http.Request) {
		assetWriter, assetOK := repo.(storage.MarketOpsAssetManagementRepository)
		jobWriter, jobOK := repo.(storage.MarketOpsAssetBackfillRepository)
		if !assetOK || !jobOK {
			writeError(w, http.StatusServiceUnavailable, "asset_management_unavailable", "asset onboarding is unavailable")
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		var req marketOpsAssetOnboardRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid onboarding request")
			return
		}
		validated, err := validateWatchlistTicker(r.Context(), req.Ticker)
		if err != nil || tenantID == "" {
			writeError(w, http.StatusBadRequest, "ticker_validation_failed", "ticker could not be validated for MarketOps")
			return
		}
		sector := validated.Sector
		if sector == "" {
			sector = validated.Industry
		}
		asset, err := assetWriter.UpsertMarketOpsAsset(r.Context(), storage.MarketOpsAssetRecord{TenantID: tenantID, UniverseGroup: analystWatchlistGroup, Ticker: validated.Ticker, Company: validated.Company, Exchange: validated.Exchange, Sector: sector, SectorKey: normalizeAssetKey(sector), Industry: validated.Industry, IndustryKey: normalizeAssetKey(validated.Industry), MetadataJSON: []byte(`{"origin":"massive_reference"}`)})
		if err != nil {
			writeError(w, http.StatusConflict, "asset_write_failed", err.Error())
			return
		}
		response := map[string]any{"asset": marketOpsAssetResponses([]storage.MarketOpsAssetRecord{asset})[0]}
		if req.BackfillEquityHistory {
			start, end, sessions, windowErr := parseAssetBackfillWindow(req.StartDate, req.EndDate)
			if windowErr != nil {
				writeError(w, http.StatusBadRequest, "invalid_backfill_window", "valid backfill dates are required")
				return
			}
			job, jobErr := jobWriter.CreateMarketOpsAssetBackfillJob(r.Context(), storage.MarketOpsAssetBackfillJobRecord{BackfillJobID: newID("assetbackfill"), TenantID: tenantID, Symbol: validated.Ticker, UniverseGroup: analystWatchlistGroup, StartDate: start, EndDate: end, Status: "queued", RequestedBy: replayActor(r, req.RequestedBy), RequestedSessions: sessions, ResultJSON: []byte(`{"dataset":"equity_eod_prices","options_history":"unavailable"}`)})
			if jobErr != nil {
				writeError(w, http.StatusConflict, "backfill_already_active", "asset was created but an active backfill already exists")
				return
			}
			response["backfill_job"] = marketOpsAssetBackfillJobResponse(job)
		}
		writeJSON(w, http.StatusCreated, response)
	})
	mux.HandleFunc("POST /v1/tenants/{tenant_id}/marketops/assets/watchlist", func(w http.ResponseWriter, r *http.Request) {
		writer, ok := repo.(storage.MarketOpsAssetManagementRepository)
		if !ok {
			writeError(w, http.StatusServiceUnavailable, "asset_management_unavailable", "asset management is unavailable")
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		var req marketOpsAssetCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid asset request")
			return
		}
		ticker := strings.ToUpper(strings.TrimSpace(req.Ticker))
		if tenantID == "" || !analystTickerPattern.MatchString(ticker) {
			writeError(w, http.StatusBadRequest, "invalid_asset", "tenant_id and a valid ticker are required")
			return
		}
		asset, err := writer.UpsertMarketOpsAsset(r.Context(), storage.MarketOpsAssetRecord{TenantID: tenantID, UniverseGroup: analystWatchlistGroup, Ticker: ticker, Company: strings.TrimSpace(req.Company), Sector: strings.TrimSpace(req.Sector), SectorKey: normalizeAssetKey(req.Sector), Industry: strings.TrimSpace(req.Industry), IndustryKey: normalizeAssetKey(req.Industry), MetadataJSON: []byte(`{"origin":"analyst_watchlist"}`)})
		if err != nil {
			writeError(w, http.StatusConflict, "asset_write_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"asset": marketOpsAssetResponses([]storage.MarketOpsAssetRecord{asset})[0]})
	})

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/assets/backfill-jobs", func(w http.ResponseWriter, r *http.Request) {
		reader, ok := repo.(storage.MarketOpsAssetBackfillRepository)
		if !ok {
			writeError(w, http.StatusServiceUnavailable, "asset_backfill_unavailable", "asset backfill is unavailable")
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		jobs, err := reader.ListMarketOpsAssetBackfillJobs(r.Context(), storage.MarketOpsAssetBackfillJobFilter{TenantID: tenantID, Symbol: r.URL.Query().Get("symbol"), Status: r.URL.Query().Get("status"), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list asset backfills")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"backfill_jobs": marketOpsAssetBackfillJobResponses(jobs)})
	})

	mux.HandleFunc("POST /v1/tenants/{tenant_id}/marketops/assets/{symbol}/backfill-jobs", func(w http.ResponseWriter, r *http.Request) {
		writer, ok := repo.(storage.MarketOpsAssetBackfillRepository)
		if !ok {
			writeError(w, http.StatusServiceUnavailable, "asset_backfill_unavailable", "asset backfill is unavailable")
			return
		}
		tenantID, symbol := strings.TrimSpace(r.PathValue("tenant_id")), strings.ToUpper(strings.TrimSpace(r.PathValue("symbol")))
		var req marketOpsAssetBackfillCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid backfill request")
			return
		}
		start, end, sessions, err := parseAssetBackfillWindow(req.StartDate, req.EndDate)
		if tenantID == "" || !analystTickerPattern.MatchString(symbol) || err != nil {
			writeError(w, http.StatusBadRequest, "invalid_backfill_window", "valid tenant, ticker, and weekday date range (1-366 sessions) are required")
			return
		}
		job, err := writer.CreateMarketOpsAssetBackfillJob(r.Context(), storage.MarketOpsAssetBackfillJobRecord{BackfillJobID: newID("assetbackfill"), TenantID: tenantID, Symbol: symbol, UniverseGroup: analystWatchlistGroup, StartDate: start, EndDate: end, Status: "queued", RequestedBy: replayActor(r, req.RequestedBy), RequestedSessions: sessions, ResultJSON: []byte(`{"dataset":"equity_eod_prices","options_history":"unavailable"}`)})
		if err != nil {
			writeError(w, http.StatusConflict, "backfill_already_active", "an active backfill already exists for this asset")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"backfill_job": marketOpsAssetBackfillJobResponse(job)})
	})
}

func parseAssetBackfillWindow(startValue, endValue string) (time.Time, time.Time, int, error) {
	end, err := time.Parse("2006-01-02", strings.TrimSpace(endValue))
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}
	start, err := time.Parse("2006-01-02", strings.TrimSpace(startValue))
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, 0, errors.New("end date precedes start date")
	}
	sessions := 0
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		if day.Weekday() != time.Saturday && day.Weekday() != time.Sunday {
			sessions++
		}
	}
	if sessions < 1 || sessions > 366 {
		return time.Time{}, time.Time{}, 0, http.ErrNotSupported
	}
	return start.UTC(), end.UTC(), sessions, nil
}

func normalizeAssetKey(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), " ", "_"))
}

func marketOpsAssetBackfillJobResponses(records []storage.MarketOpsAssetBackfillJobRecord) []any {
	out := make([]any, 0, len(records))
	for _, x := range records {
		out = append(out, marketOpsAssetBackfillJobResponse(x))
	}
	return out
}
func marketOpsAssetBackfillJobResponse(x storage.MarketOpsAssetBackfillJobRecord) any {
	return map[string]any{"backfill_job_id": x.BackfillJobID, "tenant_id": x.TenantID, "symbol": x.Symbol, "universe_group": x.UniverseGroup, "start_date": x.StartDate.Format("2006-01-02"), "end_date": x.EndDate.Format("2006-01-02"), "status": x.Status, "requested_by": x.RequestedBy, "requested_sessions": x.RequestedSessions, "completed_sessions": x.CompletedSessions, "failed_sessions": x.FailedSessions, "provider_requests": x.ProviderRequests, "error_message": x.ErrorMessage, "result": json.RawMessage(x.ResultJSON), "started_at": x.StartedAt, "completed_at": x.CompletedAt, "created_at": x.CreatedAt, "updated_at": x.UpdatedAt}
}

type marketOpsTickerValidation struct {
	Ticker   string `json:"ticker"`
	Company  string `json:"company"`
	Exchange string `json:"exchange"`
	Sector   string `json:"sector"`
	Industry string `json:"industry"`
}

func validateWatchlistTicker(ctx context.Context, value string) (marketOpsTickerValidation, error) {
	ticker := strings.ToUpper(strings.TrimSpace(value))
	if !analystTickerPattern.MatchString(ticker) {
		return marketOpsTickerValidation{}, errors.New("ticker format is invalid")
	}
	client, err := massive.NewClient(massive.LoadClientConfigFromEnv())
	if err != nil {
		return marketOpsTickerValidation{}, err
	}
	details, err := client.GetTickerDetails(ctx, ticker)
	if err != nil {
		return marketOpsTickerValidation{}, err
	}
	if !details.Active || details.Market != "stocks" || (details.Type != "cs" && details.Type != "common_stock") {
		return marketOpsTickerValidation{}, errors.New("ticker is not an active US common equity")
	}
	sector := details.Sector
	if sector == "" {
		sector = details.Industry
	}
	return marketOpsTickerValidation{Ticker: details.Ticker, Company: details.Name, Exchange: details.Exchange, Sector: sector, Industry: details.Industry}, nil
}
