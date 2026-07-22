package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type marketOpsAlgorithmAdjudicationReader interface {
	ListMarketOpsAlgorithmAdjudications(context.Context, storage.MarketOpsAlgorithmAdjudicationFilter) ([]storage.MarketOpsAlgorithmAdjudicationRecord, error)
}

func registerMarketOpsAlgorithmAdjudicationRoutes(mux *http.ServeMux, repo storage.QueryRepository) {
	mux.HandleFunc("GET /v1/marketops/algorithm-adjudications", func(w http.ResponseWriter, r *http.Request) {
		reader, ok := any(repo).(marketOpsAlgorithmAdjudicationReader)
		if !ok {
			writeError(w, http.StatusNotImplemented, "adjudications_unavailable", "algorithm adjudications are unavailable")
			return
		}
		records, err := reader.ListMarketOpsAlgorithmAdjudications(r.Context(), storage.MarketOpsAlgorithmAdjudicationFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")), HypothesisEvaluationID: strings.TrimSpace(r.URL.Query().Get("hypothesis_evaluation_id")), CorrelationID: strings.TrimSpace(r.URL.Query().Get("correlation_id")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm adjudications")
			return
		}
		out := make([]map[string]any, 0, len(records))
		for _, x := range records {
			out = append(out, map[string]any{"adjudication_id": x.AdjudicationID, "tenant_id": x.TenantID, "hypothesis_evaluation_id": x.HypothesisEvaluationID, "algorithm_result_id": x.AlgorithmResultID, "hypothesis_key": x.HypothesisKey, "hypothesis_version": x.HypothesisVersion, "symbol": x.Symbol, "session_date": x.SessionDate.UTC().Format("2006-01-02"), "verdict": x.Verdict, "confidence": x.Confidence, "explanation": json.RawMessage(jsonOrDefault(x.ExplanationJSON, "{}")), "correlation_id": x.CorrelationID, "adjudicator_version": x.AdjudicatorVersion, "created_at": x.CreatedAt.UTC().Format(time.RFC3339)})
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_adjudications": out})
	})
}
