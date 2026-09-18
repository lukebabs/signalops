package api

import (
	"context"
	"encoding/json"
	"github.com/lukebabs/signalops/internal/marketops/taskmanager"
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
	"strings"
	"time"
)

func registerMarketOpsTaskRoutes(mux *http.ServeMux, repo storage.QueryRepository) {
	mux.HandleFunc("GET /v1/administration/marketops/task-manager", func(w http.ResponseWriter, r *http.Request) {
		if !requireTenantAdministrator(w, r) {
			return
		}
		reader, ok := any(repo).(storage.MarketOpsScheduledJobStatusRepository)
		if !ok {
			writeError(w, http.StatusNotImplemented, "task_manager_unavailable", "marketops task manager is unavailable")
			return
		}
		rows, err := reader.ListMarketOpsScheduledJobStatuses(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to load task manager statuses")
			return
		}
		inputs := make([]taskmanager.Input, 0, len(rows))
		for _, x := range rows {
			inputs = append(inputs, taskmanager.Input{JobID: x.JobID, Status: x.Status, Reason: x.Reason, UpdatedAt: x.UpdatedAt, ExitCode: x.ExitCode})
		}
		writeJSON(w, http.StatusOK, map[string]any{"generated_at": time.Now().UTC().Format(time.RFC3339), "tasks": taskmanager.Evaluate(time.Now().UTC(), inputs)})
	})
	mux.HandleFunc("POST /v1/administration/marketops/task-manager/{job_id}/retry", func(w http.ResponseWriter, r *http.Request) {
		if !requireTenantAdministrator(w, r) {
			return
		}
		reader, ok := any(repo).(storage.MarketOpsScheduledJobStatusRepository)
		if !ok {
			writeError(w, http.StatusNotImplemented, "task_manager_unavailable", "marketops task manager is unavailable")
			return
		}
		rows, err := reader.ListMarketOpsScheduledJobStatuses(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to load task manager statuses")
			return
		}
		inputs := make([]taskmanager.Input, 0, len(rows))
		for _, x := range rows {
			inputs = append(inputs, taskmanager.Input{JobID: x.JobID, Status: x.Status, Reason: x.Reason, UpdatedAt: x.UpdatedAt, ExitCode: x.ExitCode})
		}
		var target *taskmanager.Decision
		for _, d := range taskmanager.Evaluate(time.Now().UTC(), inputs) {
			if d.JobID == r.PathValue("job_id") {
				target = &d
				break
			}
		}
		if target == nil {
			writeError(w, http.StatusNotFound, "task_not_found", "task is not managed")
			return
		}
		if target.State == "blocked" {
			writeError(w, http.StatusConflict, "task_retry_blocked", target.Reason)
			return
		}
		if !target.RetryEligible {
			writeError(w, http.StatusConflict, "task_retry_not_eligible", "task is not currently eligible for retry")
			return
		}
		result, err := triggerScheduledJobRunNow(context.WithoutCancel(r.Context()), target.JobID, time.Now().UTC())
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "task_retry_start_failed", "run": result})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"retry": result})
	})
	mux.HandleFunc("GET /v1/administration/marketops/tasks", func(w http.ResponseWriter, r *http.Request) {
		reader, ok := any(repo).(storage.MarketOpsTaskRepository)
		tenantID, tenantOK := requireRequestTenant(w, r, strings.TrimSpace(r.URL.Query().Get("tenant_id")))
		if !tenantOK {
			return
		}
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
		items, err := reader.ListMarketOpsTaskItems(r.Context(), storage.MarketOpsTaskItemFilter{TenantID: tenantID, WorkflowID: strings.TrimSpace(r.URL.Query().Get("workflow_id")), TaskType: strings.TrimSpace(r.URL.Query().Get("task_type")), Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")), Status: strings.TrimSpace(r.URL.Query().Get("status")), SessionDate: session, Limit: queryLimit(r, 200)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list marketops task items")
			return
		}
		if tenantID == "tenant-local" {
			if global, ok := any(repo).(storage.SubscriberGlobalAnnualFinancialTaskRepository); ok {
				if rows, e := global.ListSubscriberGlobalAnnualFinancialTasks(r.Context(), queryLimit(r, 200)); e == nil {
					items = append(items, rows...)
				}
			}
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
