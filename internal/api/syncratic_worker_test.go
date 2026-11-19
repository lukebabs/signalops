package api

import (
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestSyncraticWorkerAutoAskIsLimitedToDailyNarratives(t *testing.T) {
	if !syncraticWorkerShouldAutoAsk(storage.SyncraticContextWindowRecord{ContextStrategy: "marketops_daily_overview_v1"}) {
		t.Fatal("daily overview context should be eligible for automatic Ask enrichment")
	}
	if !syncraticWorkerShouldAutoAsk(storage.SyncraticContextWindowRecord{ContextStrategy: "marketops_sri_daily_v1"}) {
		t.Fatal("SRI daily context should be eligible for automatic Ask enrichment")
	}
	if syncraticWorkerShouldAutoAsk(storage.SyncraticContextWindowRecord{ContextStrategy: "market_state_session_v2"}) {
		t.Fatal("per-asset market-state contexts must remain operator-triggered, not automatic worker Ask calls")
	}
}
