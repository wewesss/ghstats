package render_test

import (
	"os"
	"path/filepath"
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

func splitColumns(t *testing.T, svg string) (left, right string) {
	t.Helper()
	const open = `<g transform="translate(505 0)">`
	const closing = `</g><line x1="500"`
	i := strings.Index(svg, open)
	if i < 0 {
		t.Fatal("no right column group")
	}
	rest := svg[i+len(open):]
	j := strings.Index(rest, closing)
	if j < 0 {
		t.Fatal("right column group is not followed by the divider")
	}
	return svg[:i], rest[:j]
}

func TestWideWithoutLanguagesFillsBothColumns(t *testing.T) {
	for _, tt := range []struct {
		name        string
		blocks      []config.Block
		left, right string
	}{
		{"no languages", []config.Block{{Name: "stats", Variants: []string{"totals"}}, {Name: "history", Variants: []string{"views"}}}, "STARS", "VIEWS"},
		{"with languages", []config.Block{{Name: "stats", Variants: []string{"totals"}}, {Name: "languages", Variants: []string{"bars"}}}, "STARS", "Go"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := sample.WideRender()
			cfg.Blocks = tt.blocks
			svg := renderSVG(t, sample.Report(), cfg, render.Dark)
			assertWellFormed(t, svg)
			if w, _ := viewBox(t, svg); w != 1000 {
				t.Fatalf("viewBox width = %d", w)
			}
			left, right := splitColumns(t, svg)
			if strings.Count(left, `<g class="gsb`) != 1 || strings.Count(right, `<g class="gsb`) != 1 {
				t.Fatalf("each column must hold one block, got left %d right %d", strings.Count(left, `<g class="gsb`), strings.Count(right, `<g class="gsb`))
			}
			if !strings.Contains(left, tt.left) || strings.Contains(right, tt.left) {
				t.Fatalf("%s must be drawn in the left column only", tt.left)
			}
			if !strings.Contains(right, tt.right) || strings.Contains(left, tt.right) {
				t.Fatalf("%s must be drawn in the right column only", tt.right)
			}
			if !strings.Contains(svg, `x1="500"`) {
				t.Fatal("the columns must be separated")
			}
		})
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
	if s := render.Snippet(c); !strings.Contains(s, `width="495"`) {
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
	if !strings.Contains(s, `src="ghstats/identity-dark.svg" width="98%"`) {
		t.Fatalf("a lone identity in a row is 98%% wide:\n%s", s)
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

func TestSnippetRowLoneCards(t *testing.T) {
	cfg := config.Default()
	cfg.Render.Cards = []config.Card{
		{Name: "a", Blocks: []config.Block{{Name: "stats", Variants: []string{"totals"}}, {Name: "history", Variants: []string{"views"}}}, Layout: config.LayoutWide},
		{Name: "b", Blocks: []config.Block{{Name: "stats", Variants: []string{"totals"}}}},
	}
	cfg.Render.Rows = []config.Row{{Items: []string{"a"}}, {Items: []string{"b"}}}
	s := render.Snippet(cfg)
	if !strings.Contains(s, `src="ghstats/a-dark.svg" width="98%"`) || !strings.Contains(s, `src="ghstats/b-dark.svg" width="495"`) {
		t.Fatalf("snippet = %s", s)
	}
}

var firstBlockYRe = regexp.MustCompile(`<g class="gsb\d+" transform="translate\(0 (\d+)\)"`)

func firstBlockY(t *testing.T, svg string) int {
	t.Helper()
	m := firstBlockYRe.FindStringSubmatch(svg)
	if m == nil {
		t.Fatal("no block group")
	}
	y, _ := strconv.Atoi(m[1])
	return y
}

func TestStretchedCardCentersItsBlocks(t *testing.T) {
	for _, layout := range []string{config.LayoutCard, config.LayoutWide} {
		short := []config.Block{{Name: "stats", Variants: []string{"totals"}}, {Name: "history", Variants: []string{"clones"}}}
		tall := []config.Block{{Name: "languages", Variants: []string{"treemap"}}, {Name: "stats", Variants: []string{"totals"}}}
		cfg := config.Default()
		cfg.Render.Cards = []config.Card{
			{Name: "short", Blocks: short, Layout: layout},
			{Name: "tall", Blocks: tall, Layout: config.LayoutCard},
		}
		cfg.Render.Rows = []config.Row{{Items: []string{"short", "tall"}}}
		stretched, err := render.Cards(sample.Report(), cfg, render.Dark)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Render.Rows = nil
		plain, err := render.Cards(sample.Report(), cfg, render.Dark)
		if err != nil {
			t.Fatal(err)
		}
		_, hs := viewBox(t, stretched["short-dark.svg"])
		_, hp := viewBox(t, plain["short-dark.svg"])
		if hs <= hp {
			t.Fatalf("%s: card must be stretched, %d vs %d", layout, hs, hp)
		}
		ys, yp := firstBlockY(t, stretched["short-dark.svg"]), firstBlockY(t, plain["short-dark.svg"])
		if want := yp + (hs-hp)/2; ys != want {
			t.Fatalf("%s: first block y = %d, want %d", layout, ys, want)
		}
		if !strings.Contains(stretched["short-dark.svg"], `y1="46" y2="46"`) {
			t.Fatalf("%s: header line must stay at the top", layout)
		}
		assertWellFormed(t, stretched["short-dark.svg"])
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

func TestRowCardsShareTheTallestHeight(t *testing.T) {
	short := []config.Block{{Name: "stats", Variants: []string{"totals"}}}
	tall := []config.Block{{Name: "languages", Variants: []string{"bars"}}}
	cfg := config.Default()
	cfg.Render.Cards = []config.Card{
		{Name: "short", Blocks: short},
		{Name: "tall", Blocks: tall},
		{Name: "alone", Blocks: short},
	}
	cfg.Render.Rows = []config.Row{{Items: []string{"short", "tall"}}, {Items: []string{"alone"}}}

	files, err := render.Cards(sample.Report(), cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	_, hs := viewBox(t, files["short-dark.svg"])
	_, ht := viewBox(t, files["tall-dark.svg"])
	_, ha := viewBox(t, files["alone-dark.svg"])
	if hs != ht {
		t.Fatalf("heights %d and %d must match", hs, ht)
	}
	cfg.Render.Rows = nil
	plain, err := render.Cards(sample.Report(), cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	_, ps := viewBox(t, plain["short-dark.svg"])
	_, pt := viewBox(t, plain["tall-dark.svg"])
	if ps >= pt || ht != pt || ha != ps {
		t.Fatalf("short %d tall %d alone %d; without rows short %d tall %d", hs, ht, ha, ps, pt)
	}
	if files["tall-dark.svg"] != plain["tall-dark.svg"] {
		t.Fatal("the tallest card must not change")
	}
	assertWellFormed(t, files["short-dark.svg"])
	if !strings.Contains(files["short-dark.svg"], "STARS") {
		t.Fatal("content must stay in the stretched card")
	}
}

func TestRowHeightIgnoresIdentity(t *testing.T) {
	cfg := sample.Profile()
	cfg.Render.Cards = []config.Card{{Name: "a", Blocks: []config.Block{{Name: "stats", Variants: []string{"totals"}}}}}
	base, err := render.Cards(sample.Report(), cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Render.Rows = []config.Row{{Items: []string{"identity", "a"}}}
	got, err := render.Cards(sample.Report(), cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	if got["a-dark.svg"] != base["a-dark.svg"] || got["identity-dark.svg"] != base["identity-dark.svg"] {
		t.Fatal("identity must not take part in the height of a row")
	}
}

func TestLanguageMultipliersOption(t *testing.T) {
	off := false
	cfg := config.DefaultRender()
	cfg.Blocks = []config.Block{{Name: "languages", Variants: []string{"bars"}}}
	cfg.Motion = false
	if !strings.Contains(renderSVG(t, sample.Report(), cfg, render.Dark), "×0.35") {
		t.Fatal("multipliers are shown by default")
	}
	cfg.Languages.Multipliers = &off
	if svg := renderSVG(t, sample.Report(), cfg, render.Dark); strings.Contains(svg, "×") {
		t.Fatal("multipliers: false must drop every multiplier label")
	}
}

func docSnippetExample(t *testing.T, heading string) (yamlSrc, html string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "docs", "en", "guide", "configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := strings.ReplaceAll(string(b), "\r\n", "\n")
	_, section, ok := strings.Cut(doc, "### "+heading+"\n")
	if !ok {
		t.Fatalf("no %q section in configuration.md", heading)
	}
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	if end := strings.Index(section, "\n### "); end >= 0 {
		section = section[:end]
	}
	block := func(src, fence string) (string, string) {
		_, rest, ok := strings.Cut(src, "```"+fence+"\n")
		if !ok {
			t.Fatalf("no %s block in the %q section", fence, heading)
		}
		body, after, ok := strings.Cut(rest, "```")
		if !ok {
			t.Fatalf("unterminated %s block in the %q section", fence, heading)
		}
		return body, after
	}
	yamlSrc, after := block(section, "yaml")
	html, _ = block(after, "html")
	return yamlSrc, html
}

func TestDocSnippetExamplesMatchSnippet(t *testing.T) {
	for _, heading := range []string{"Several cards", "Layout in the README"} {
		t.Run(heading, func(t *testing.T) {
			src, want := docSnippetExample(t, heading)
			cfg, err := config.Parse(strings.NewReader(src))
			if err != nil {
				t.Fatal(err)
			}
			if got := render.Snippet(cfg); strings.TrimSpace(got) != strings.TrimSpace(want) {
				t.Fatalf("docs/en/guide/configuration.md differs from render.Snippet\n--- Snippet\n%s\n--- docs\n%s", got, want)
			}
		})
	}
}
