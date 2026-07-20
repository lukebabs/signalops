package state

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestG144FiveSessionChangesSkipIneligibleSurfaceSessions(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	chain := []storage.MarketOpsOptionsChainRecord{}
	for day := 0; day < 7; day++ {
		session := start.AddDate(0, 0, day)
		records := usableSurfaceFixtures(session)
		for index := range records {
			*records[index].ImpliedVolatility += float64(day) * .01
			*records[index].Bid += float64(day) * .1
			*records[index].Ask += float64(day) * .1
			value := int64(100 + day*10)
			records[index].OpenInterest = &value
			if day == 3 && records[index].OptionTicker == "O:AAPL-PUT25" {
				records[index].ImpliedVolatility = nil
				records[index].Bid = nil
				records[index].OpenInterest = nil
			}
		}
		chain = append(chain, records...)
	}
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "g144-eligible-lookback"}, BuildInput{OptionChain: chain})
	if err != nil {
		t.Fatal(err)
	}
	final := observationsForSession(result.Observations, start.AddDate(0, 0, 6))
	dims := `{"option_type":"put","target_delta":0.25,"target_dte":30}`
	assertNumericFeature(t, final, "iv_change_5d", dims, .06, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "premium_change_5d", dims, .6, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "oi_change_5d", dims, 60, storage.MarketOpsQualityUsable)
}
