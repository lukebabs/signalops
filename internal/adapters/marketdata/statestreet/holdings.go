package statestreet

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const Source = "state_street"

var supportedSymbols = map[string]struct{}{
	"KRE": {}, "XLB": {}, "XLC": {}, "XLE": {}, "XLF": {}, "XLI": {},
	"XLK": {}, "XLP": {}, "XLRE": {}, "XLU": {}, "XLV": {}, "XLY": {},
}

type Holding struct {
	Rank       int
	Ticker     string
	Name       string
	Identifier string
	SEDOL      string
	Sector     string
	Currency   string
	Weight     float64
	SharesHeld float64
}

type Snapshot struct {
	ETFSymbol     string
	FundName      string
	EffectiveDate time.Time
	RetrievedAt   time.Time
	SourceURL     string
	ContentHash   string
	Holdings      []Holding
	TotalWeight   float64
	TopTenWeight  float64
}

type Client struct {
	HTTPClient *http.Client
	Now        func() time.Time
}

func Supports(symbol string) bool {
	_, ok := supportedSymbols[strings.ToUpper(strings.TrimSpace(symbol))]
	return ok
}

func HoldingsURL(symbol string) (string, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if !Supports(symbol) {
		return "", fmt.Errorf("state street holdings are not configured for %s", symbol)
	}
	return "https://www.ssga.com/library-content/products/fund-data/etfs/us/holdings-daily-us-en-" + strings.ToLower(symbol) + ".xlsx", nil
}

func (c Client) DownloadCurrent(ctx context.Context, symbol string) (Snapshot, error) {
	sourceURL, err := HoldingsURL(symbol)
	if err != nil {
		return Snapshot{}, err
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return Snapshot{}, fmt.Errorf("build state street request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return Snapshot{}, fmt.Errorf("download state street holdings: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Snapshot{}, fmt.Errorf("state street holdings request returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 10<<20))
	if err != nil {
		return Snapshot{}, fmt.Errorf("read state street holdings: %w", err)
	}
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now().UTC()
	}
	return parseWorkbook(body, strings.ToUpper(strings.TrimSpace(symbol)), sourceURL, now)
}

type sharedStrings struct {
	Items []sharedString `xml:"si"`
}

type sharedString struct {
	Text string     `xml:"t"`
	Runs []richText `xml:"r"`
}

type richText struct {
	Text string `xml:"t"`
}

type worksheet struct {
	SheetData sheetData `xml:"sheetData"`
}

type sheetData struct {
	Rows []row `xml:"row"`
}

type row struct {
	Cells []cell `xml:"c"`
}

type cell struct {
	Reference string `xml:"r,attr"`
	Type      string `xml:"t,attr"`
	Value     string `xml:"v"`
	Inline    string `xml:"is>t"`
}

func parseWorkbook(raw []byte, symbol, sourceURL string, retrievedAt time.Time) (Snapshot, error) {
	if len(raw) == 0 {
		return Snapshot{}, errors.New("state street workbook is empty")
	}
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return Snapshot{}, fmt.Errorf("open state street workbook: %w", err)
	}
	read := func(name string) ([]byte, error) {
		for _, file := range archive.File {
			if file.Name != name {
				continue
			}
			reader, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer reader.Close()
			return io.ReadAll(io.LimitReader(reader, 10<<20))
		}
		return nil, fmt.Errorf("workbook file %s is missing", name)
	}
	sharedRaw, err := read("xl/sharedStrings.xml")
	if err != nil {
		return Snapshot{}, err
	}
	var shared sharedStrings
	if err := xml.Unmarshal(sharedRaw, &shared); err != nil {
		return Snapshot{}, fmt.Errorf("decode shared strings: %w", err)
	}
	values := make([]string, len(shared.Items))
	for i, item := range shared.Items {
		values[i] = strings.TrimSpace(item.Text)
		for _, run := range item.Runs {
			values[i] += run.Text
		}
		values[i] = strings.TrimSpace(values[i])
	}
	sheetRaw, err := read("xl/worksheets/sheet1.xml")
	if err != nil {
		return Snapshot{}, err
	}
	var sheet worksheet
	if err := xml.Unmarshal(sheetRaw, &sheet); err != nil {
		return Snapshot{}, fmt.Errorf("decode holdings sheet: %w", err)
	}
	cellValue := func(value cell) (string, error) {
		if value.Type == "s" {
			index, err := strconv.Atoi(strings.TrimSpace(value.Value))
			if err != nil || index < 0 || index >= len(values) {
				return "", fmt.Errorf("invalid shared string index %q", value.Value)
			}
			return values[index], nil
		}
		if value.Type == "inlineStr" {
			return strings.TrimSpace(value.Inline), nil
		}
		return strings.TrimSpace(value.Value), nil
	}
	rowValues := func(source row) (map[int]string, error) {
		out := map[int]string{}
		for _, item := range source.Cells {
			column := columnIndex(item.Reference)
			if column < 0 {
				continue
			}
			value, err := cellValue(item)
			if err != nil {
				return nil, err
			}
			out[column] = value
		}
		return out, nil
	}
	var fundName string
	var effectiveDate time.Time
	headers := map[string]int{}
	headerRow := -1
	for index, item := range sheet.SheetData.Rows {
		cells, err := rowValues(item)
		if err != nil {
			return Snapshot{}, err
		}
		if strings.EqualFold(cells[0], "Fund Name:") {
			fundName = cells[1]
		}
		if strings.EqualFold(cells[0], "Holdings:") {
			effectiveDate, err = time.Parse("As of 02-Jan-2006", cells[1])
			if err != nil {
				return Snapshot{}, fmt.Errorf("parse state street effective date %q: %w", cells[1], err)
			}
		}
		for column, value := range cells {
			headers[strings.ToLower(strings.TrimSpace(value))] = column
		}
		if _, ok := headers["name"]; ok {
			if _, ok := headers["weight"]; ok {
				if _, ok := headers["shares held"]; ok {
					headerRow = index
					break
				}
			}
		}
		headers = map[string]int{}
	}
	if fundName == "" || effectiveDate.IsZero() || headerRow < 0 {
		return Snapshot{}, errors.New("state street workbook does not contain expected holdings metadata")
	}
	nameColumn, tickerColumn, weightColumn := headers["name"], headers["ticker"], headers["weight"]
	sharesColumn := headers["shares held"]
	identifierColumn, hasIdentifier := headers["identifier"]
	sedolColumn, hasSEDOL := headers["sedol"]
	sectorColumn, hasSector := headers["sector"]
	currencyColumn, hasCurrency := headers["local currency"]
	out := Snapshot{ETFSymbol: symbol, FundName: fundName, EffectiveDate: effectiveDate.UTC(), RetrievedAt: retrievedAt.UTC(), SourceURL: sourceURL}
	for _, item := range sheet.SheetData.Rows[headerRow+1:] {
		cells, err := rowValues(item)
		if err != nil {
			return Snapshot{}, err
		}
		name := strings.TrimSpace(cells[nameColumn])
		weight, ok := decimal(cells[weightColumn])
		if name == "" || !ok {
			continue
		}
		shares, _ := decimal(cells[sharesColumn])
		holding := Holding{Rank: len(out.Holdings) + 1, Ticker: strings.ToUpper(strings.TrimSpace(cells[tickerColumn])), Name: name, Weight: weight, SharesHeld: shares}
		if hasIdentifier {
			holding.Identifier = strings.TrimSpace(cells[identifierColumn])
		}
		if hasSEDOL {
			holding.SEDOL = strings.TrimSpace(cells[sedolColumn])
		}
		if hasSector {
			holding.Sector = strings.TrimSpace(cells[sectorColumn])
		}
		if hasCurrency {
			holding.Currency = strings.TrimSpace(cells[currencyColumn])
		}
		out.Holdings = append(out.Holdings, holding)
		out.TotalWeight += holding.Weight
		if holding.Rank <= 10 {
			out.TopTenWeight += holding.Weight
		}
	}
	if len(out.Holdings) == 0 {
		return Snapshot{}, errors.New("state street workbook contains no holdings")
	}
	hash := sha256.Sum256(raw)
	out.ContentHash = hex.EncodeToString(hash[:])
	return out, nil
}

func decimal(value string) (float64, bool) {
	value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), "%"))
	if value == "" || value == "-" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", ""), 64)
	return parsed, err == nil
}

func columnIndex(reference string) int {
	index := 0
	count := 0
	for _, rune := range strings.ToUpper(reference) {
		if rune < 'A' || rune > 'Z' {
			break
		}
		index = index*26 + int(rune-'A'+1)
		count++
	}
	if count == 0 {
		return -1
	}
	return index - 1
}
