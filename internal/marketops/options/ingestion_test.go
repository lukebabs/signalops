package options

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
)

func TestChainRecordFromMassiveSnapshot(t *testing.T) {
	underlying := 200.0
	oi := int64(1000)
	iv := 0.42
	record, err := ChainRecordFromMassiveSnapshot("tenant-local", "src-massive", "run-1", massive.OptionContractDailyRecord{
		ProviderContractID: "O:NVDA260116C00100000",
		OptionTicker:       "O:NVDA260116C00100000",
		UnderlyingSymbol:   "nvda",
		ContractType:       "CALL",
		ExpirationDate:     time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC),
		StrikePrice:        100,
		ObservationDate:    time.Date(2026, 7, 17, 14, 30, 0, 0, time.UTC),
		UnderlyingClose:    &underlying,
		OpenInterest:       &oi,
		ImpliedVolatility:  &iv,
		Raw:                map[string]any{"provider": "massive"},
	})
	if err != nil {
		t.Fatalf("convert snapshot: %v", err)
	}
	if record.TenantID != "tenant-local" || record.Symbol != "NVDA" || record.ContractType != "call" {
		t.Fatalf("record = %+v", record)
	}
	if record.TradeDate.Format("2006-01-02") != "2026-07-17" || record.ExpirationDate.Format("2006-01-02") != "2026-01-16" {
		t.Fatalf("dates = %s/%s", record.TradeDate, record.ExpirationDate)
	}
	if record.Moneyness == nil || *record.Moneyness != 0.5 {
		t.Fatalf("moneyness = %v", record.Moneyness)
	}
	if record.OpenInterest == nil || *record.OpenInterest != 1000 || record.ImpliedVolatility == nil || *record.ImpliedVolatility != 0.42 {
		t.Fatalf("oi/iv = %v/%v", record.OpenInterest, record.ImpliedVolatility)
	}
	if record.PayloadHash == "" || string(record.RawPayloadJSON) == "{}" {
		t.Fatalf("raw/hash = %q/%s", record.PayloadHash, string(record.RawPayloadJSON))
	}
}

func TestChainRecordFromMassiveSnapshotRejectsInvalidRecord(t *testing.T) {
	_, err := ChainRecordFromMassiveSnapshot("tenant-local", "src", "run", massive.OptionContractDailyRecord{UnderlyingSymbol: "NVDA", OptionTicker: "O:BAD", ContractType: "straddle"})
	if err == nil {
		t.Fatal("expected invalid contract type error")
	}
}
