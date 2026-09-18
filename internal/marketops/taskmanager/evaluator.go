package taskmanager

import "time"

type Dependency struct {
	JobID            string        `json:"job_id"`
	RequiredStatuses []string      `json:"required_statuses"`
	MaxAge           time.Duration `json:"max_age"`
}
type Contract struct {
	JobID, Label string
	Dependencies []Dependency
	MaxAttempts  int
	Backoff      time.Duration
}
type Input struct {
	JobID, Status, Reason string
	UpdatedAt             time.Time
	ExitCode              *int
}
type Decision struct {
	JobID, Label, State, Reason string
	Dependencies                []string
	RetryEligible               bool
	Attempt                     int
	NextAttemptAt               *time.Time
}

var contracts = []Contract{
	{"marketops-warm-eod", "Warm EOD baseline", nil, 3, 15 * time.Minute},
	{"marketops-intraday", "Intraday monitor", nil, 3, 5 * time.Minute},
	{"marketops-risk-reward", "Risk/Reward", []Dependency{{"marketops-daily-postclose", []string{"succeeded"}, 24 * time.Hour}}, 3, 15 * time.Minute},
	{"marketops-sri-refresh", "SRI refresh", []Dependency{{"marketops-daily-postclose", []string{"succeeded"}, 24 * time.Hour}}, 3, 15 * time.Minute},
	{"marketops-sri-holdings-refresh", "SRI holdings", []Dependency{{"marketops-sri-refresh", []string{"succeeded"}, 24 * time.Hour}}, 3, 15 * time.Minute},
	{"marketops-daily-postclose", "Daily post-close", nil, 3, 15 * time.Minute},
	{"marketops-saf-benchmark", "Signal Assurance benchmark", []Dependency{{"marketops-daily-postclose", []string{"succeeded"}, 24 * time.Hour}, {"marketops-risk-reward", []string{"succeeded"}, 24 * time.Hour}}, 3, 30 * time.Minute},
	{"marketops-saf-evaluation", "Signal Assurance", []Dependency{{"marketops-saf-benchmark", []string{"succeeded"}, 24 * time.Hour}, {"marketops-risk-reward", []string{"succeeded"}, 24 * time.Hour}}, 3, 30 * time.Minute},
	{"marketops-postclose-recovery", "Post-close recovery", nil, 3, 15 * time.Minute},
}

func Contracts() []Contract { return append([]Contract(nil), contracts...) }
func Evaluate(now time.Time, inputs []Input) []Decision {
	by := map[string]Input{}
	for _, x := range inputs {
		by[x.JobID] = x
	}
	out := make([]Decision, 0, len(contracts))
	for _, c := range contracts {
		x := by[c.JobID]
		d := Decision{JobID: c.JobID, Label: c.Label, State: "ready", Attempt: 0}
		if x.Status == "running" {
			d.State = "running"
		} else if x.Status == "succeeded" {
			d.State = "succeeded"
		} else if x.Status == "skipped" {
			d.State = "deferred"
			d.Reason = x.Reason
		} else if x.Status == "failed" {
			d.State = "retryable"
			d.Reason = x.Reason
			d.RetryEligible = true
		} else if x.Status != "" {
			d.State = "ready"
		}
		for _, dep := range c.Dependencies {
			upstream := by[dep.JobID]
			d.Dependencies = append(d.Dependencies, dep.JobID)
			if upstream.Status == "" {
				d.State = "blocked"
				d.Reason = "dependency has no recorded status"
				d.RetryEligible = false
			} else {
				ok := false
				for _, s := range dep.RequiredStatuses {
					if upstream.Status == s {
						ok = true
					}
				}
				if !ok {
					d.State = "blocked"
					d.Reason = "dependency " + dep.JobID + " is " + upstream.Status
					d.RetryEligible = false
				} else if dep.MaxAge > 0 && !upstream.UpdatedAt.IsZero() && now.Sub(upstream.UpdatedAt) > dep.MaxAge {
					d.State = "blocked"
					d.Reason = "dependency " + dep.JobID + " is stale"
					d.RetryEligible = false
				}
			}
		}
		out = append(out, d)
	}
	return out
}
