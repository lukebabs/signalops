package api

import (
	"encoding/json"
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
	"strings"
	"time"
)

func registerMarketOpsTaskRoutes(mux *http.ServeMux, repo storage.QueryRepository) {
	mux.HandleFunc("GET /v1/administration/marketops/tasks", func(w http.ResponseWriter, r *http.Request) {
		reader, ok := any(repo).(storage.MarketOpsTaskRepository)
		if !ok {
			writeError(w, http.StatusNotImplemented, "marketops_tasks_unavailable", "marketops task control is unavailable")
			return
		}
		session := time.Time{}
		if raw := strings.TrimSpace(r.URL.Query().Get("session_date")); raw != "" {
			parsed, err := time.Parse("2006-01-02", raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid_query", "session_date must be YYYY-MM-DD")
				return
			}
			session = parsed
		}
		items, err := reader.ListMarketOpsTaskItems(r.Context(), storage.MarketOpsTaskItemFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), WorkflowID: strings.TrimSpace(r.URL.Query().Get("workflow_id")), TaskType: strings.TrimSpace(r.URL.Query().Get("task_type")), Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")), Status: strings.TrimSpace(r.URL.Query().Get("status")), SessionDate: session, Limit: queryLimit(r, 200)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list marketops task items")
			return
		}
		out := make([]map[string]any, 0, len(items))
		for _, x := range items {
			out = append(out, map[string]any{"task_id": x.TaskID, "workflow_id": x.WorkflowID, "tenant_id": x.TenantID, "session_date": x.SessionDate.Format("2006-01-02"), "task_type": x.TaskType, "symbol": x.Symbol, "status": x.Status, "attempt_count": x.AttemptCount, "max_attempts": x.MaxAttempts, "next_attempt_at": x.NextAttemptAt, "provider": x.Provider, "failure_class": x.FailureClass, "provider_status": x.ProviderStatus, "error_message": x.ErrorMessage, "result": taskResultJSON(x.ResultJSON), "completed_at": x.CompletedAt, "created_at": x.CreatedAt, "updated_at": x.UpdatedAt})
		}
		writeJSON(w, http.StatusOK, map[string]any{"tasks": out})
	})
}

func taskResultJSON(raw []byte) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}
