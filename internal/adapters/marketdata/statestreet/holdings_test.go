package statestreet

import (
	"archive/zip"
	"bytes"
	"testing"
	"time"
)

func TestParseWorkbook(t *testing.T) {
	var body bytes.Buffer
	archive := zip.NewWriter(&body)
	write := func(name, value string) {
		file, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	write("xl/sharedStrings.xml", `<?xml version="1.0"?><sst><si><t>Fund Name:</t></si><si><t>State Street Test Fund</t></si><si><t>Holdings:</t></si><si><t>As of 07-Aug-2026</t></si><si><t>Name</t></si><si><t>Ticker</t></si><si><t>Identifier</t></si><si><t>SEDOL</t></si><si><t>Weight</t></si><si><t>Sector</t></si><si><t>Shares Held</t></si><si><t>Local Currency</t></si><si><t>Alpha Corp</t></si><si><t>ALP</t></si><si><t>ID1</t></si><si><t>SED1</t></si><si><t>Technology</t></si><si><t>USD</t></si><si><t>Beta Corp</t></si><si><t>BET</t></si></sst>`)
	write("xl/worksheets/sheet1.xml", `<?xml version="1.0"?><worksheet><sheetData>
<row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>
<row r="3"><c r="A3" t="s"><v>2</v></c><c r="B3" t="s"><v>3</v></c></row>
<row r="5"><c r="A5" t="s"><v>4</v></c><c r="B5" t="s"><v>5</v></c><c r="C5" t="s"><v>6</v></c><c r="D5" t="s"><v>7</v></c><c r="E5" t="s"><v>8</v></c><c r="F5" t="s"><v>9</v></c><c r="G5" t="s"><v>10</v></c><c r="H5" t="s"><v>11</v></c></row>
<row r="6"><c r="A6" t="s"><v>12</v></c><c r="B6" t="s"><v>13</v></c><c r="C6" t="s"><v>14</v></c><c r="D6" t="s"><v>15</v></c><c r="E6"><v>60.5</v></c><c r="F6" t="s"><v>16</v></c><c r="G6"><v>10</v></c><c r="H6" t="s"><v>17</v></c></row>
<row r="7"><c r="A7" t="s"><v>18</v></c><c r="B7" t="s"><v>19</v></c><c r="E7"><v>39.5</v></c><c r="G7"><v>20</v></c></row>
</sheetData></worksheet>`)
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := parseWorkbook(body.Bytes(), "XLK", "https://example.test/xlk.xlsx", time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.FundName != "State Street Test Fund" || snapshot.EffectiveDate.Format("2006-01-02") != "2026-08-07" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
	if len(snapshot.Holdings) != 2 || snapshot.Holdings[0].Ticker != "ALP" || snapshot.TotalWeight != 100 || snapshot.TopTenWeight != 100 {
		t.Fatalf("unexpected holdings: %#v", snapshot)
	}
}
