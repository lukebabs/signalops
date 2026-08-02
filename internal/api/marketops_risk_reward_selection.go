package api

import "fmt"

func riskRewardUsableInputs(payload map[string]any) int {
	if basis, ok := payload["confidence_basis"].(map[string]any); ok {
		if value, ok := numberAny(basis["usable_technical_inputs"]); ok {
			return int(value)
		}
	}
	return 0
}

func betterRiskRewardPayload(candidate, current map[string]any) bool {
	candidateUsable, currentUsable := riskRewardUsableInputs(candidate), riskRewardUsableInputs(current)
	candidateEligible, currentEligible := candidateUsable >= riskRewardMinimumUsableInputs, currentUsable >= riskRewardMinimumUsableInputs
	if candidateEligible != currentEligible {
		return candidateEligible
	}
	if candidateUsable != currentUsable {
		return candidateUsable > currentUsable
	}
	return fmt.Sprint(candidate["_created_at"]) > fmt.Sprint(current["_created_at"])
}
