package massive

import (
	"os"
	"strings"
	"testing"
)

func TestTopMegacapCompaniesParsesSeedFile(t *testing.T) {
	companies, err := TopMegacapCompanies()
	if err != nil {
		t.Fatalf("parse megacap companies: %v", err)
	}
	if len(companies) != 50 {
		t.Fatalf("company count = %d, want 50", len(companies))
	}

	first := companies[0]
	if first.Rank != 1 || first.Ticker != "NVDA" || first.Company != "NVIDIA" {
		t.Fatalf("first company = %+v", first)
	}
	if first.Sector != "Technology" || first.Industry != "Semiconductors" {
		t.Fatalf("first classification = %q / %q", first.Sector, first.Industry)
	}
	if first.TickerKey != "nvda" || first.CompanyKey != "nvidia" || first.SectorKey != "technology" {
		t.Fatalf("first keys = %+v", first)
	}

	last := companies[len(companies)-1]
	if last.Rank != 50 || last.Ticker != "PANW" || last.Company != "Palo Alto Networks" {
		t.Fatalf("last company = %+v", last)
	}
	if last.Sector != "Technology" || last.Industry != "Cybersecurity" {
		t.Fatalf("last classification = %q / %q", last.Sector, last.Industry)
	}
}

func TestParseMegacapCompaniesNormalizesExchangeSuffixes(t *testing.T) {
	companies, err := TopMegacapCompanies()
	if err != nil {
		t.Fatalf("parse megacap companies: %v", err)
	}

	byTicker := map[string]MegacapCompanySeed{}
	for _, company := range companies {
		byTicker[company.Ticker] = company
	}

	if byTicker["BRK.B"].TickerKey != "brk_b" {
		t.Fatalf("BRK.B ticker key = %q", byTicker["BRK.B"].TickerKey)
	}
	for _, removed := range []string{"2222.SR", "005930.KS", "000660.KS", "601939.SS", "RO.SW"} {
		if _, ok := byTicker[removed]; ok {
			t.Fatalf("removed ticker %s remains in canonical universe", removed)
		}
	}
	for _, replacement := range []string{"PM", "RY", "BABA", "NVS", "PANW"} {
		if _, ok := byTicker[replacement]; !ok {
			t.Fatalf("replacement ticker %s missing from canonical universe", replacement)
		}
	}
}

func TestNormalizedMegacapSeedMatchesParser(t *testing.T) {
	companies, err := TopMegacapCompanies()
	if err != nil {
		t.Fatalf("parse megacap companies: %v", err)
	}
	got, err := os.ReadFile("top50megacap.normalized.csv")
	if err != nil {
		t.Fatalf("read normalized seed: %v", err)
	}
	if want := MegacapSeedCSV(companies); string(got) != want {
		t.Fatal("normalized seed does not match parsed canonical source")
	}
}

func TestMegacapSeedCSV(t *testing.T) {
	companies, err := TopMegacapCompanies()
	if err != nil {
		t.Fatalf("parse megacap companies: %v", err)
	}

	csv := MegacapSeedCSV(companies[:2])
	if !strings.HasPrefix(csv, "rank,ticker,ticker_key,company,company_key,sector,sector_key,industry,industry_key\n") {
		t.Fatalf("csv header missing: %q", csv)
	}
	if !strings.Contains(csv, "1,NVDA,nvda,NVIDIA,nvidia,Technology,technology,Semiconductors,semiconductors") {
		t.Fatalf("csv missing NVDA row: %q", csv)
	}
}
