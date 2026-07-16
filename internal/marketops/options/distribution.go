package options

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const DefaultWindowName = "10_trade_days"

type bucketTotals struct {
	CallOpenInterest int64 `json:"call_open_interest"`
	PutOpenInterest  int64 `json:"put_open_interest"`
	CallVolume       int64 `json:"call_volume"`
	PutVolume        int64 `json:"put_volume"`
	ContractCount    int   `json:"contract_count"`
}

func BuildDistribution(tenantID string, symbol string, tradeDate time.Time, rows []storage.MarketOpsOptionsChainRecord) storage.MarketOpsOptionsDistributionRecord {
	tradeDate = dayOnly(tradeDate)
	sourceDates := sourceTradeDates(rows)
	latestRows := []storage.MarketOpsOptionsChainRecord{}
	for _, row := range rows {
		if sameDay(row.TradeDate, tradeDate) {
			latestRows = append(latestRows, row)
		}
	}
	if len(latestRows) == 0 {
		latestRows = rows
	}

	record := storage.MarketOpsOptionsDistributionRecord{TenantID: strings.TrimSpace(tenantID), Symbol: strings.ToUpper(strings.TrimSpace(symbol)), TradeDate: tradeDate, WindowName: DefaultWindowName, Provider: "massive", TradeDays: len(sourceDates), SourceTradeDates: sourceDates}
	moneyness := map[string]*bucketTotals{}
	expiration := map[string]*bucketTotals{}
	for _, row := range latestRows {
		record.ContractCount++
		if record.SourceID == "" {
			record.SourceID = row.SourceID
		}
		call := strings.EqualFold(row.ContractType, "call")
		put := strings.EqualFold(row.ContractType, "put")
		if call {
			record.CallContractCount++
		}
		if put {
			record.PutContractCount++
		}
		if row.OpenInterest == nil {
			record.MissingOpenInterestCount++
		}
		openInterest := int64(0)
		if row.OpenInterest != nil {
			openInterest = *row.OpenInterest
		}
		volume := int64(0)
		if row.Volume != nil {
			volume = *row.Volume
		}
		if call {
			record.TotalCallOpenInterest += openInterest
			record.TotalCallVolume += volume
		}
		if put {
			record.TotalPutOpenInterest += openInterest
			record.TotalPutVolume += volume
		}
		addBucket(moneyness, moneynessBucket(row), call, put, openInterest, volume)
		addBucket(expiration, expirationBucket(row, tradeDate), call, put, openInterest, volume)
	}
	record.CallPutOpenInterestRatio = roundRatio(float64(record.TotalCallOpenInterest) / float64(maxInt64(record.TotalPutOpenInterest, 1)))
	record.CallPutVolumeRatio = roundRatio(float64(record.TotalCallVolume) / float64(maxInt64(record.TotalPutVolume, 1)))

	ratios := ratiosByDate(rows)
	if len(ratios) > 0 {
		current := ratios[len(ratios)-1]
		if sameDay(current.date, tradeDate) {
			if len(ratios) > 1 {
				previous := ratios[len(ratios)-2].ratio
				record.RatioDelta = roundRatio(current.ratio - previous)
				if previous != 0 {
					record.RatioChangePct = roundRatio(((current.ratio - previous) / previous) * 100)
				}
			}
			mean, stddev := stats(ratios[:len(ratios)-1])
			if stddev > 0 {
				record.RatioZScore = roundRatio((current.ratio - mean) / stddev)
				record.ChangePointScore = roundRatio(math.Abs(record.RatioZScore))
				record.Confidence = roundRatio(math.Min(0.99, math.Abs(record.RatioZScore)/4))
			}
		}
	}
	record.MoneynessDistributionJSON, _ = json.Marshal(materializeBuckets(moneyness))
	record.ExpirationDistributionJSON, _ = json.Marshal(materializeBuckets(expiration))
	record.MetricsJSON, _ = json.Marshal(map[string]any{"primary_metric": "open_interest", "secondary_metric": "volume", "window_name": DefaultWindowName})
	return record
}

func addBucket(buckets map[string]*bucketTotals, name string, call bool, put bool, openInterest int64, volume int64) {
	if name == "" {
		name = "unknown"
	}
	bucket := buckets[name]
	if bucket == nil {
		bucket = &bucketTotals{}
		buckets[name] = bucket
	}
	bucket.ContractCount++
	if call {
		bucket.CallOpenInterest += openInterest
		bucket.CallVolume += volume
	}
	if put {
		bucket.PutOpenInterest += openInterest
		bucket.PutVolume += volume
	}
}

func materializeBuckets(values map[string]*bucketTotals) map[string]bucketTotals {
	out := map[string]bucketTotals{}
	for key, value := range values {
		out[key] = *value
	}
	return out
}

func moneynessBucket(row storage.MarketOpsOptionsChainRecord) string {
	if row.Moneyness == nil || *row.Moneyness <= 0 {
		if row.UnderlyingClose == nil || *row.UnderlyingClose <= 0 {
			return "unknown"
		}
		value := row.StrikePrice / *row.UnderlyingClose
		row.Moneyness = &value
	}
	switch value := *row.Moneyness; {
	case value < 0.90:
		return "<90%"
	case value < 0.95:
		return "90-95%"
	case value <= 1.00:
		return "95-100%"
	case value <= 1.05:
		return "100-105%"
	case value <= 1.10:
		return "105-110%"
	default:
		return ">110%"
	}
}

func expirationBucket(row storage.MarketOpsOptionsChainRecord, tradeDate time.Time) string {
	days := int(dayOnly(row.ExpirationDate).Sub(dayOnly(tradeDate)).Hours() / 24)
	switch {
	case days <= 7:
		return "0-7d"
	case days <= 30:
		return "8-30d"
	case days <= 60:
		return "31-60d"
	default:
		return "61d+"
	}
}

type datedRatio struct {
	date  time.Time
	ratio float64
}

func ratiosByDate(rows []storage.MarketOpsOptionsChainRecord) []datedRatio {
	byDate := map[time.Time][2]int64{}
	for _, row := range rows {
		date := dayOnly(row.TradeDate)
		totals := byDate[date]
		oi := int64(0)
		if row.OpenInterest != nil {
			oi = *row.OpenInterest
		}
		if strings.EqualFold(row.ContractType, "call") {
			totals[0] += oi
		}
		if strings.EqualFold(row.ContractType, "put") {
			totals[1] += oi
		}
		byDate[date] = totals
	}
	dates := make([]time.Time, 0, len(byDate))
	for date := range byDate {
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	out := make([]datedRatio, 0, len(dates))
	for _, date := range dates {
		totals := byDate[date]
		out = append(out, datedRatio{date: date, ratio: roundRatio(float64(totals[0]) / float64(maxInt64(totals[1], 1)))})
	}
	return out
}

func sourceTradeDates(rows []storage.MarketOpsOptionsChainRecord) []time.Time {
	seen := map[time.Time]struct{}{}
	for _, row := range rows {
		if row.TradeDate.IsZero() {
			continue
		}
		seen[dayOnly(row.TradeDate)] = struct{}{}
	}
	out := make([]time.Time, 0, len(seen))
	for date := range seen {
		out = append(out, date)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

func stats(values []datedRatio) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value.ratio
	}
	mean := sum / float64(len(values))
	variance := 0.0
	for _, value := range values {
		delta := value.ratio - mean
		variance += delta * delta
	}
	return mean, math.Sqrt(variance / float64(len(values)))
}

func sameDay(a time.Time, b time.Time) bool {
	return dayOnly(a).Equal(dayOnly(b))
}

func dayOnly(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func roundRatio(value float64) float64 {
	return math.Round(value*1000000) / 1000000
}
