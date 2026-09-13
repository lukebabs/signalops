package api

import "testing"

func TestHistoricalAssuranceDataSelectionIsImmutable(t *testing.T) {
	selection := historicalAssuranceDataSelection()
	if selection["usage_context"] != "historical_assurance" {
		t.Fatalf("usage_context = %v", selection["usage_context"])
	}
	if selection["selected_observation_role"] != "initial_tenant_local_capture" {
		t.Fatalf("selected_observation_role = %v", selection["selected_observation_role"])
	}
	if selection["restatement"] != "disabled" {
		t.Fatalf("restatement = %v", selection["restatement"])
	}
}
