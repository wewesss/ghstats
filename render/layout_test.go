package render_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/sample"
)

var viewBoxRe = regexp.MustCompile(`viewBox="0 0 (\d+) (\d+)"`)

func viewBox(t *testing.T, svg string) (w, h int) {
	t.Helper()
	m := viewBoxRe.FindStringSubmatch(svg)
	if m == nil {
		t.Fatal("no viewBox")
	}
	w, _ = strconv.Atoi(m[1])
	h, _ = strconv.Atoi(m[2])
	return w, h
}

func TestWideWithoutLanguagesFillsBothColumns(t *testing.T) {
	cfg := sample.WideRender()
	cfg.Blocks = []config.Block{{Name: "stats", Variants: []string{"totals"}}, {Name: "history", Variants: []string{"views"}}}
	svg := renderSVG(t, sample.Report(), cfg, render.Dark)
	assertWellFormed(t, svg)
	if w, _ := viewBox(t, svg); w != 1000 {
		t.Fatalf("viewBox width = %d", w)
	}
	if !strings.Contains(svg, `translate(505 0)`) || !strings.Contains(svg, `x1="500"`) {
		t.Fatal("both columns must be occupied and separated")
	}
	if !strings.Contains(svg, "STARS") || !strings.Contains(svg, "VIEWS") {
		t.Fatal("both blocks must be drawn")
	}
}

func TestWideWithOneBlockFallsBackToCard(t *testing.T) {
	cfg := sample.WideRender()
	cfg.Blocks = []config.Block{{Name: "languages", Variants: []string{"bars"}}}
	svg := renderSVG(t, sample.Report(), cfg, render.Dark)
	assertWellFormed(t, svg)
	if w, _ := viewBox(t, svg); w != 495 {
		t.Fatalf("viewBox width = %d", w)
	}
	if strings.Contains(svg, `translate(505 0)`) {
		t.Fatal("a one-block card has a single column")
	}

	c := config.Default()
	c.Render.Cards = []config.Card{{Name: "langs", Layout: config.LayoutWide, Blocks: cfg.Blocks}}
	if s := render.Snippet(c); !strings.Contains(s, `width="495"`) || strings.Contains(s, `width="100%"`) {
		t.Fatalf("snippet = %s", s)
	}
}

func TestCardWithoutHeader(t *testing.T) {
	off := false
	blocks := []config.Block{{Name: "stats", Variants: []string{"totals"}}, {Name: "history", Variants: []string{"views"}}}
	for _, layout := range []string{config.LayoutCard, config.LayoutWide} {
		cfg := config.Default()
		cfg.Render.Layout = layout
		cfg.Render.Cards = []config.Card{
			{Name: "with", Blocks: blocks},
			{Name: "without", Header: &off, Blocks: blocks},
		}
		files, err := render.Cards(sample.Report(), cfg, render.Dark)
		if err != nil {
			t.Fatal(err)
		}
		with, without := files["with-dark.svg"], files["without-dark.svg"]
		assertWellFormed(t, without)
		if !strings.Contains(with, "synced") || strings.Contains(without, "synced") || strings.Contains(without, "/ ghstats") || strings.Contains(without, `cy="27"`) {
			t.Fatalf("%s: header must be absent only from the headerless card", layout)
		}
		_, hw := viewBox(t, with)
		_, hn := viewBox(t, without)
		if hw-hn != 40 {
			t.Fatalf("%s: height %d vs %d, want a 40 px difference", layout, hw, hn)
		}
	}
}

func TestSnippetRowsSideBySide(t *testing.T) {
	cfg := sample.Profile()
	cfg.Render.Cards = []config.Card{
		{Name: "overview", Blocks: []config.Block{{Name: "stats", Variants: []string{"totals"}}, {Name: "history", Variants: []string{"views"}}}, Layout: config.LayoutWide},
		{Name: "langs", Blocks: []config.Block{{Name: "languages", Variants: []string{"treemap"}}}},
		{Name: "extra", Blocks: []config.Block{{Name: "history", Variants: []string{"clones"}}}},
	}
	cfg.Render.Rows = []config.Row{
		{Align: "right", Items: []string{"langs", "extra"}},
		{Items: []string{"identity"}},
		{Align: "left", Items: []string{"links"}},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	s := render.Snippet(cfg)
	pic := func(name, width string) string {
		return `<picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/` + name + `-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/` + name + `-light.svg"><img alt="GitHub stats: ` + name + `" src="ghstats/` + name + `-dark.svg" width="` + width + `"></picture>`
	}
	want := "<p align=\"right\">\n  " + pic("langs", "49%") + " " + pic("extra", "49%") + "\n</p>\n"
	if !strings.HasPrefix(s, want) {
		t.Fatalf("first row differs\n%s", s)
	}
	order := []string{
		`<p align="right">`,
		`<p align="center">` + "\n  " + `<picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/identity-dark.svg">`,
		`<p align="left">` + "\n" + `<a href="`,
		"<picture>\n  <source media=\"(prefers-color-scheme: dark)\" srcset=\"ghstats/overview-dark.svg\">",
	}
	pos := 0
	for _, w := range order {
		i := strings.Index(s[pos:], w)
		if i < 0 {
			t.Fatalf("missing or out of order %q in\n%s", w, s)
		}
		pos += i + len(w)
	}
	if strings.Count(s, "<picture>") != 4+3 {
		t.Fatalf("each item is written once:\n%s", s)
	}
}

func TestSnippetRowWidthsFollowNaturalWidths(t *testing.T) {
	cfg := config.Default()
	cfg.Render.Cards = []config.Card{
		{Name: "a", Blocks: []config.Block{{Name: "stats", Variants: []string{"totals"}}, {Name: "history", Variants: []string{"views"}}}, Layout: config.LayoutWide},
		{Name: "b", Blocks: []config.Block{{Name: "stats", Variants: []string{"totals"}}}},
	}
	cfg.Render.Rows = []config.Row{{Items: []string{"a", "b"}}}
	s := render.Snippet(cfg)
	if !strings.Contains(s, `src="ghstats/a-dark.svg" width="65%"`) || !strings.Contains(s, `src="ghstats/b-dark.svg" width="32%"`) {
		t.Fatalf("snippet = %s", s)
	}
}

func TestSnippetRowsIgnoreAbsentItems(t *testing.T) {
	cfg := config.Default()
	cfg.Render.Rows = []config.Row{{Items: []string{"identity"}}, {Items: []string{"links"}}, {Align: "right", Items: []string{"stats"}}}
	s := render.Snippet(cfg)
	if strings.Contains(s, "identity") || strings.Contains(s, `<a href`) || !strings.HasPrefix(s, `<p align="right">`) {
		t.Fatalf("snippet = %s", s)
	}
}

func TestSnippetWithoutRowsIsUnchanged(t *testing.T) {
	cfg := sample.Profile()
	withRows := cfg
	withRows.Render.Rows = nil
	if render.Snippet(cfg) != render.Snippet(withRows) {
		t.Fatal("nil rows must not change the snippet")
	}
}
