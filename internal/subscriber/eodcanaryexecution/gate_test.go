package eodcanaryexecution

import "testing"

func TestFreezeCreatesExactlyTwoDeterministicRequestSlots(t *testing.T) {
	members, err := Freeze(ExpectedWorkerIdentity, []Member{
		{GlobalAssetID: "asset-nvda", Ticker: "nvda", Priority: 2},
		{GlobalAssetID: "asset-aapl", Ticker: "aapl", Priority: 1},
	}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 || members[0].Ticker != "AAPL" || members[0].RequestOrdinal != 1 || members[1].Ticker != "NVDA" || members[1].RequestOrdinal != 2 {
		t.Fatalf("unexpected frozen member slots: %#v", members)
	}
}

func TestFreezeFailsClosed(t *testing.T) {
	if _, err := Freeze("browser-session", []Member{{GlobalAssetID: "a", Ticker: "A", Priority: 1}}, 1); err == nil {
		t.Fatal("expected wrong identity to fail")
	}
	if _, err := Freeze(ExpectedWorkerIdentity, []Member{{GlobalAssetID: "a", Ticker: "A", Priority: 1}}, 2); err == nil {
		t.Fatal("expected request budget mismatch to fail")
	}
	if _, err := Freeze(ExpectedWorkerIdentity, []Member{{GlobalAssetID: "a", Ticker: "A", Priority: 1}, {GlobalAssetID: "b", Ticker: "A", Priority: 2}}, 2); err == nil {
		t.Fatal("expected duplicate ticker to fail")
	}
}
