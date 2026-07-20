package postgres

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestValidateMarketOpsOpportunityDisposition(t *testing.T) {
	valid := storage.MarketOpsOpportunityDispositionRecord{
		DispositionID: "moppdisp-1", TenantID: "tenant-1", OpportunityID: "mopp-1",
		Disposition: storage.MarketOpsOpportunityDispositionWatch, Actor: "analyst-1",
		MetadataJSON: []byte(`{"source":"api"}`), CreatedAt: time.Now().UTC(),
	}
	if err := validateMarketOpsOpportunityDisposition(valid); err != nil {
		t.Fatalf("valid disposition rejected: %v", err)
	}
	for name, mutate := range map[string]func(*storage.MarketOpsOpportunityDispositionRecord){
		"missing_actor":       func(record *storage.MarketOpsOpportunityDispositionRecord) { record.Actor = "" },
		"invalid_disposition": func(record *storage.MarketOpsOpportunityDispositionRecord) { record.Disposition = "approved" },
		"invalid_metadata":    func(record *storage.MarketOpsOpportunityDispositionRecord) { record.MetadataJSON = []byte(`[]`) },
		"missing_created_at":  func(record *storage.MarketOpsOpportunityDispositionRecord) { record.CreatedAt = time.Time{} },
	} {
		t.Run(name, func(t *testing.T) {
			record := valid
			mutate(&record)
			if validateMarketOpsOpportunityDisposition(record) == nil {
				t.Fatalf("invalid disposition accepted: %+v", record)
			}
		})
	}
}
