package massive

import (
	"embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

//go:embed top50megacap.txt
var megacapSeedFS embed.FS

type MegacapCompanySeed struct {
	Rank        int
	Ticker      string
	TickerKey   string
	Company     string
	CompanyKey  string
	Sector      string
	SectorKey   string
	Industry    string
	IndustryKey string
}

var megacapLinePattern = regexp.MustCompile(`^(.+) \(([^)]+)\)\s+[–-]\s+(.+)$`)

func TopMegacapCompanies() ([]MegacapCompanySeed, error) {
	contents, err := megacapSeedFS.ReadFile("top50megacap.txt")
	if err != nil {
		return nil, err
	}
	return ParseMegacapCompanies(string(contents))
}

func ParseMegacapCompanies(contents string) ([]MegacapCompanySeed, error) {
	lines := strings.Split(contents, "\n")
	companies := make([]MegacapCompanySeed, 0, 50)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(strings.ToLower(line), "the top ") {
			continue
		}
		match := megacapLinePattern.FindStringSubmatch(line)
		if match == nil {
			return nil, fmt.Errorf("invalid megacap seed line: %q", line)
		}
		company := strings.TrimSpace(match[1])
		ticker := strings.ToUpper(strings.TrimSpace(match[2]))
		classification := classificationAfterValuation(match[3])
		sector, industry := splitClassification(classification)
		if company == "" || ticker == "" || sector == "" {
			return nil, fmt.Errorf("incomplete megacap seed line: %q", line)
		}

		companies = append(companies, MegacapCompanySeed{
			Rank:        len(companies) + 1,
			Ticker:      ticker,
			TickerKey:   tickerKey(ticker),
			Company:     company,
			CompanyKey:  slugKey(company),
			Sector:      sector,
			SectorKey:   slugKey(sector),
			Industry:    industry,
			IndustryKey: slugKey(industry),
		})
	}
	return companies, nil
}

func classificationAfterValuation(value string) string {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, "|")
	if len(parts) > 1 {
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return value
}

func splitClassification(value string) (string, string) {
	parts := strings.SplitN(value, "/", 2)
	sector := strings.TrimSpace(parts[0])
	if len(parts) == 1 {
		return sector, ""
	}
	return sector, strings.TrimSpace(parts[1])
}

func tickerKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			b.WriteRune('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func slugKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range value {
		isWord := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isWord {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteRune('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func MegacapSeedCSV(companies []MegacapCompanySeed) string {
	var b strings.Builder
	b.WriteString("rank,ticker,ticker_key,company,company_key,sector,sector_key,industry,industry_key\n")
	for _, company := range companies {
		b.WriteString(strconv.Itoa(company.Rank))
		b.WriteByte(',')
		b.WriteString(csvField(company.Ticker))
		b.WriteByte(',')
		b.WriteString(csvField(company.TickerKey))
		b.WriteByte(',')
		b.WriteString(csvField(company.Company))
		b.WriteByte(',')
		b.WriteString(csvField(company.CompanyKey))
		b.WriteByte(',')
		b.WriteString(csvField(company.Sector))
		b.WriteByte(',')
		b.WriteString(csvField(company.SectorKey))
		b.WriteByte(',')
		b.WriteString(csvField(company.Industry))
		b.WriteByte(',')
		b.WriteString(csvField(company.IndustryKey))
		b.WriteByte('\n')
	}
	return b.String()
}

func csvField(value string) string {
	if strings.ContainsAny(value, ",\"\n") {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return value
}
