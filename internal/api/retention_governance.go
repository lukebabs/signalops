package api

import (
	"encoding/json"
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
	"time"
)

func registerRetentionGovernanceRoutes(mux *http.ServeMux, repo storage.QueryRepository) {
	reader, ok := any(repo).(storage.RetentionGovernanceRepository)
	if !ok {
		return
	}
	mux.HandleFunc("GET /v1/administration/storage/governance", func(w http.ResponseWriter, r *http.Request) {
		policies, err := reader.ListRetentionPolicies(r.Context(), "tenant-local")
		if err != nil {
			writeError(w, 500, "query_failed", "failed to load retention policies")
			return
		}
		runs, err := reader.ListRetentionRuns(r.Context(), "tenant-local", 100)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to load retention runs")
			return
		}
		latest := map[string]storage.RetentionRunRecord{}
		for _, run := range runs {
			if _, ok := latest[run.PolicyID]; !ok {
				latest[run.PolicyID] = run
			}
		}
		out := make([]any, 0, len(policies))
		for _, p := range policies {
			row := map[string]any{"policy_id": p.PolicyID, "app_id": p.AppID, "domain": p.Domain, "data_class": p.DataClass, "retention_days": p.RetentionDays, "mode": p.Mode, "preservation_rule": p.PreservationRule, "description": p.Description, "updated_at": p.UpdatedAt}
			if run, ok := latest[p.PolicyID]; ok {
				var detail any = map[string]any{}
				_ = json.Unmarshal(run.DetailJSON, &detail)
				row["last_run"] = map[string]any{"run_id": run.RunID, "mode": run.Mode, "status": run.Status, "candidate_rows": run.CandidateRows, "affected_rows": run.AffectedRows, "oldest_candidate_at": run.OldestCandidateAt, "newest_candidate_at": run.NewestCandidateAt, "started_at": run.StartedAt, "completed_at": run.CompletedAt, "detail": detail}
			}
			out = append(out, row)
		}
		writeJSON(w, 200, map[string]any{"generated_at": time.Now().UTC(), "policies": out, "runs": retentionRunResponses(runs)})
	})
}
func retentionRunResponses(items []storage.RetentionRunRecord) []any {
	out := make([]any, 0, len(items))
	for _, x := range items {
		var detail any = map[string]any{}
		_ = json.Unmarshal(x.DetailJSON, &detail)
		out = append(out, map[string]any{"run_id": x.RunID, "policy_id": x.PolicyID, "mode": x.Mode, "status": x.Status, "candidate_rows": x.CandidateRows, "affected_rows": x.AffectedRows, "oldest_candidate_at": x.OldestCandidateAt, "newest_candidate_at": x.NewestCandidateAt, "started_at": x.StartedAt, "completed_at": x.CompletedAt, "detail": detail})
	}
	return out
}
