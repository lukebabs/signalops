package main

var cohortStageOrder = []string{"preflight", "state_materialization", "hypothesis_evaluation", "opportunity_build", "outcome_materialization", "hypothesis_proposal_generation"}

func parseStages(value string) []string {
	requested := parseList(value, false)
	selected := map[string]bool{}
	for _, stage := range requested {
		selected[stage] = true
	}
	out := []string{}
	for _, stage := range cohortStageOrder {
		if selected[stage] {
			out = append(out, stage)
			delete(selected, stage)
		}
	}
	for _, stage := range requested {
		if selected[stage] {
			out = append(out, stage)
			delete(selected, stage)
		}
	}
	return out
}
