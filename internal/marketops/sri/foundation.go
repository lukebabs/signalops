package sri

import (
	"github.com/lukebabs/signalops/internal/storage"
	"sort"
	"time"
)

const AlgorithmVersion = "sri.foundation.v1"
const RegistryVersion = "sri.registry.v1"

type PricePoint struct {
	Session time.Time
	Close   float64
	EventID string
}
type ScoredSegment struct {
	Segment                                                              storage.MarketOpsSRISegmentRecord
	PrimaryETF                                                           string
	Session                                                              time.Time
	State, QualityState                                                  string
	Composite, RelativeStrength, Momentum, Acceleration, EvidenceQuality *float64
	Rank                                                                 *int
	Flags                                                                []string
	Components                                                           map[string]float64
	Provenance                                                           map[string]any
}

func FoundationRegistry(tenant string) ([]storage.MarketOpsSRISegmentRecord, []storage.MarketOpsSRIETFRecord) {
	defs := []struct{ key, name, typ, parent, primary, secondary string }{
		{"benchmark_market", "Market", "benchmark", "", "SPY", ""}, {"benchmark_growth", "Growth", "benchmark", "", "QQQ", ""}, {"benchmark_equal_weight", "Equal Weight Market", "benchmark", "", "RSP", ""},
		{"sector_materials", "Materials", "sector", "", "XLB", ""}, {"sector_communication", "Communication Services", "sector", "", "XLC", ""}, {"sector_energy", "Energy", "sector", "", "XLE", ""}, {"sector_financials", "Financials", "sector", "", "XLF", ""}, {"sector_industrials", "Industrials", "sector", "", "XLI", ""}, {"sector_technology", "Technology", "sector", "", "XLK", ""}, {"sector_staples", "Consumer Staples", "sector", "", "XLP", ""}, {"sector_real_estate", "Real Estate", "sector", "", "XLRE", ""}, {"sector_utilities", "Utilities", "sector", "", "XLU", ""}, {"sector_healthcare", "Healthcare", "sector", "", "XLV", ""}, {"sector_discretionary", "Consumer Discretionary", "sector", "", "XLY", ""},
		{"industry_software", "Software", "industry", "sector_technology", "IGV", "SKYY"}, {"industry_semiconductors", "Semiconductors", "industry", "sector_technology", "SMH", "SOXX"}, {"industry_biotech", "Biotechnology", "industry", "sector_healthcare", "IBB", "XBI"}, {"industry_regional_banks", "Regional Banks", "industry", "sector_financials", "KRE", "KBE"}, {"industry_oil_services", "Oil Services", "industry", "sector_energy", "OIH", "XOP"},
	}
	segs := make([]storage.MarketOpsSRISegmentRecord, 0, len(defs))
	etfs := []storage.MarketOpsSRIETFRecord{}
	for _, d := range defs {
		id := "sri_" + d.key
		segs = append(segs, storage.MarketOpsSRISegmentRecord{TenantID: tenant, SegmentID: id, SegmentKey: d.key, Name: d.name, SegmentType: d.typ, ParentSegmentKey: d.parent, Active: true, RegistryVersion: RegistryVersion, MetadataJSON: []byte("{}")})
		role := "primary"
		if d.typ == "benchmark" {
			role = "context"
		}
		etfs = append(etfs, storage.MarketOpsSRIETFRecord{TenantID: tenant, ETFSymbol: d.primary, SegmentID: id, Role: role, BenchmarkPriority: 1, Active: true, RegistryVersion: RegistryVersion, ConfigJSON: []byte("{}")})
		if d.secondary != "" {
			etfs = append(etfs, storage.MarketOpsSRIETFRecord{TenantID: tenant, ETFSymbol: d.secondary, SegmentID: id, Role: "secondary", BenchmarkPriority: 2, Active: true, RegistryVersion: RegistryVersion, ConfigJSON: []byte("{}")})
		}
	}
	return segs, etfs
}
func Score(segments []storage.MarketOpsSRISegmentRecord, registry []storage.MarketOpsSRIETFRecord, prices map[string][]PricePoint) []ScoredSegment {
	primary := map[string]string{}
	for _, x := range registry {
		if x.Role == "primary" || x.Role == "context" {
			if _, ok := primary[x.SegmentID]; !ok {
				primary[x.SegmentID] = x.ETFSymbol
			}
		}
	}
	spy := prices["SPY"]
	qqq := prices["QQQ"]
	rsp := prices["RSP"]
	raw := []ScoredSegment{}
	for _, segment := range segments {
		if segment.SegmentType == "benchmark" {
			continue
		}
		symbol := primary[segment.SegmentID]
		series := prices[symbol]
		if len(series) < 61 || len(spy) < 61 || len(qqq) < 61 || len(rsp) < 61 {
			raw = append(raw, ScoredSegment{Segment: segment, PrimaryETF: symbol, QualityState: "partial", State: storage.MarketOpsSRIStateNeutral, Flags: []string{"INSUFFICIENT_PRICE_HISTORY"}})
			continue
		}
		r5 := ret(series, 5)
		r20 := ret(series, 20)
		r60 := ret(series, 60)
		b5 := (ret(spy, 5) + ret(qqq, 5) + ret(rsp, 5)) / 3
		b20 := (ret(spy, 20) + ret(qqq, 20) + ret(rsp, 20)) / 3
		b60 := (ret(spy, 60) + ret(qqq, 60) + ret(rsp, 60)) / 3
		raw = append(raw, ScoredSegment{Segment: segment, PrimaryETF: symbol, Session: series[len(series)-1].Session, QualityState: "usable", State: storage.MarketOpsSRIStateNeutral, Components: map[string]float64{"rs_5d": r5 - b5, "rs_20d": r20 - b20, "rs_60d": r60 - b60, "return_5d": r5, "return_20d": r20, "return_60d": r60, "acceleration": r5 - retAt(series, 5, 20)}, Provenance: map[string]any{"price_basis": "normalized_equity_eod", "primary_etf": symbol}})
	}
	valid := []int{}
	for i := range raw {
		if raw[i].QualityState == "usable" {
			valid = append(valid, i)
		}
	}
	if len(valid) == 0 {
		return raw
	}
	rs5 := percentiles(raw, valid, "rs_5d")
	rs20 := percentiles(raw, valid, "rs_20d")
	rs60 := percentiles(raw, valid, "rs_60d")
	m5 := percentiles(raw, valid, "return_5d")
	m20 := percentiles(raw, valid, "return_20d")
	m60 := percentiles(raw, valid, "return_60d")
	acc := percentiles(raw, valid, "acceleration")
	for _, i := range valid {
		v := &raw[i]
		rs := .10*rs5[i] + .40*rs20[i] + .50*rs60[i]
		mo := .15*m5[i] + .35*m20[i] + .50*m60[i]
		ac := acc[i]
		co := .55*rs + .30*mo + .15*ac
		v.RelativeStrength = &rs
		v.Momentum = &mo
		v.Acceleration = &ac
		v.Composite = &co
		q := 1.0
		v.EvidenceQuality = &q
		switch {
		case co >= 75:
			v.State = storage.MarketOpsSRIStateLeading
		case co >= 60 && ac >= 55:
			v.State = storage.MarketOpsSRIStateImproving
		case co < 25:
			v.State = storage.MarketOpsSRIStateLagging
		case co < 40 && ac < 45:
			v.State = storage.MarketOpsSRIStateWeakening
		default:
			v.State = storage.MarketOpsSRIStateNeutral
		}
		v.Components["relative_strength_score"] = rs
		v.Components["momentum_score"] = mo
		v.Components["momentum_acceleration_score"] = ac
		v.Components["composite_score"] = co
	}
	sort.Slice(valid, func(a, b int) bool { return *raw[valid[a]].Composite > *raw[valid[b]].Composite })
	for n, i := range valid {
		rank := n + 1
		raw[i].Rank = &rank
	}
	return raw
}
func ret(x []PricePoint, n int) float64 {
	if len(x) <= n || x[len(x)-1-n].Close == 0 {
		return 0
	}
	return x[len(x)-1].Close/x[len(x)-1-n].Close - 1
}
func retAt(x []PricePoint, offset, n int) float64 {
	end := len(x) - 1 - offset
	if end-n < 0 || x[end-n].Close == 0 {
		return 0
	}
	return x[end].Close/x[end-n].Close - 1
}
func percentiles(rows []ScoredSegment, idx []int, key string) map[int]float64 {
	order := append([]int{}, idx...)
	sort.Slice(order, func(i, j int) bool { return rows[order[i]].Components[key] < rows[order[j]].Components[key] })
	out := map[int]float64{}
	for rank, i := range order {
		if len(order) == 1 {
			out[i] = 50
		} else {
			out[i] = 100 * float64(rank) / float64(len(order)-1)
		}
	}
	return out
}
