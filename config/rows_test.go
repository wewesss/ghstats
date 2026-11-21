package config_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
)

const rowsCards = `version: 1
render:
  cards:
    - name: overview
      blocks: [{stats: [totals]}]
    - name: langs
      header: false
      blocks: [{languages: [treemap]}]
`

func TestRenderRowsReject(t *testing.T) {
	tests := []struct {
		name, rows, want string
	}{
		{"unknown align", "    - align: top\n      items: [overview]\n", "render.rows[0].align"},
		{"empty items", "    - items: []\n", "render.rows[0].items: must list at least one item"},
		{"unknown item", "    - items: [nope]\n", `unknown item "nope"`},
		{"placed twice in a row", "    - items: [overview, overview]\n", `"overview" is already placed`},
		{"placed in two rows", "    - items: [overview]\n    - items: [overview]\n", `"overview" is already placed`},
		{"links with another item", "    - items: [links, overview]\n", `link items cannot share a row`},
		{"links after another item", "    - items: [overview, links]\n", `link items cannot share a row`},
		{"13 rows", strings.Repeat("    - items: [identity]\n", 13), "at most 12 rows"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Parse(strings.NewReader(rowsCards + "  rows:\n" + tt.rows))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestRenderRowsAccept(t *testing.T) {
	rows := "  rows:\n" +
		"    - items: [identity]\n" +
		"    - align: left\n      items: [overview, langs]\n" +
		"    - align: right\n      items: [links]\n"
	got, err := config.Parse(strings.NewReader(rowsCards + rows))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Render.Rows) != 3 || got.Render.Rows[1].Align != "left" || got.Render.Rows[2].Items[0] != "links" {
		t.Fatalf("rows = %+v", got.Render.Rows)
	}
}

func TestRenderRowsAcceptTwelve(t *testing.T) {
	var ok strings.Builder
	ok.WriteString("version: 1\nrender:\n  cards:\n")
	for i := range 12 {
		fmt.Fprintf(&ok, "    - name: c%d\n      blocks: [{stats: [totals]}]\n", i)
	}
	ok.WriteString("  rows:\n")
	for i := range 12 {
		fmt.Fprintf(&ok, "    - items: [c%d]\n", i)
	}
	if _, err := config.Parse(strings.NewReader(ok.String())); err != nil {
		t.Fatal(err)
	}
}

func TestRenderRowsWithoutCardsUseStats(t *testing.T) {
	if _, err := config.Parse(strings.NewReader("version: 1\nrender:\n  rows:\n    - items: [stats, identity]\n")); err != nil {
		t.Fatal(err)
	}
}

func TestCardHeaderDefaultsToTrue(t *testing.T) {
	got, err := config.Parse(strings.NewReader(rowsCards))
	if err != nil {
		t.Fatal(err)
	}
	cards := got.Render.CardList()
	if !cards[0].HasHeader() || cards[1].HasHeader() {
		t.Fatalf("headers = %v %v", cards[0].HasHeader(), cards[1].HasHeader())
	}
	if !config.DefaultRender().CardList()[0].HasHeader() {
		t.Fatal("the implicit stats card must keep its header")
	}
}

func TestEncodeRoundTripRowsAndHeader(t *testing.T) {
	off := false
	c := config.Default()
	c.Render.Cards = []config.Card{
		{Name: "overview", Blocks: []config.Block{{Name: "stats", Variants: []string{"totals"}}}},
		{Name: "langs", Header: &off, Blocks: []config.Block{{Name: "languages", Variants: []string{"treemap"}}}},
	}
	c.Render.Rows = []config.Row{
		{Items: []string{"identity"}},
		{Align: "right", Items: []string{"overview", "langs"}},
		{Align: "left", Items: []string{"links"}},
	}
	var buf bytes.Buffer
	if err := config.Encode(&buf, c); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "header: false") || !strings.Contains(out, "rows:") {
		t.Fatalf("output lacks header or rows:\n%s", out)
	}
	got, err := config.Parse(strings.NewReader(out))
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !reflect.DeepEqual(got, c) {
		t.Fatalf("round trip differs\ngot:  %+v\nwant: %+v\n%s", got, c, out)
	}
}

func TestLanguageMultipliersDefaultAndRoundTrip(t *testing.T) {
	if !config.DefaultRender().Languages.ShowMultipliers() {
		t.Fatal("multipliers are shown by default")
	}
	off := false
	c := config.Default()
	c.Render.Languages.Multipliers = &off
	var buf bytes.Buffer
	if err := config.Encode(&buf, c); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "multipliers: false") {
		t.Fatalf("output lacks multipliers:\n%s", buf.String())
	}
	got, err := config.Parse(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Render.Languages.ShowMultipliers() || !reflect.DeepEqual(got, c) {
		t.Fatalf("round trip differs: %+v", got.Render.Languages)
	}
}
