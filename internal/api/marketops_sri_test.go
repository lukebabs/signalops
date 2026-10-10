package api

import (
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestSRISnapshotResponseExposesPrimaryETF(t *testing.T) {
	got := sriSnapshotResponse(storage.MarketOpsSRISnapshotRecord{InputProvenanceJSON: []byte(`{"primary_etf":"xlk"}`)})
	if got["primary_etf"] != "XLK" {
		t.Fatalf("primary_etf = %v, want XLK", got["primary_etf"])
	}
}
