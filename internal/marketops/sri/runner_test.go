package sri

import (
	"testing"
	"time"
)

func TestCommonSessionsRequiresEveryRegistryETF(t *testing.T) {
	date := func(day int) time.Time { return time.Date(2026, 8, day, 0, 0, 0, 0, time.UTC) }
	prices := map[string][]PricePoint{
		"AAA": {{Session: date(3)}, {Session: date(4)}, {Session: date(5)}},
		"BBB": {{Session: date(3)}, {Session: date(5)}},
		"CCC": {{Session: date(3)}, {Session: date(5)}},
	}

	got := commonSessions(prices, []string{"AAA", "BBB", "CCC"})
	if len(got) != 2 || !got[0].Equal(date(3)) || !got[1].Equal(date(5)) {
		t.Fatalf("common sessions = %v, want Aug 3 and Aug 5", got)
	}
}
