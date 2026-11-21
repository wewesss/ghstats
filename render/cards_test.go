package render_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/report"
	"github.com/wewesss/ghstats/sample"
)

func TestCardsSplitsBlocksIntoFiles(t *testing.T) {
	cfg := config.Default()
	cfg.Render.Cards = []config.Card{
		{Name: "overview", Blocks: []config.Block{{Name: "stats", Variants: []string{"totals"}}, {Name: "history", Variants: []string{"views"}}}},
		{Name: "langs", Layout: config.LayoutWide, Blocks: []config.Block{{Name: "languages", Variants: []string{"treemap"}}}},
	}
	files, err := render.Cards(sample.Report(), cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	if got := strings.Join(names, ","); got != "langs-dark.svg,overview-dark.svg" {
		t.Fatalf("files = %s", got)
	}
	overview, langs := files["overview-dark.svg"], files["langs-dark.svg"]
	assertWellFormed(t, overview)
	assertWellFormed(t, langs)
	if !strings.Contains(overview, "STARS") || !strings.Contains(overview, "VIEWS") {
		t.Fatal("overview must hold the stats and history blocks")
	}
	if strings.Contains(langs, "STARS") || strings.Contains(langs, "VIEWS") {
		t.Fatal("langs must hold only the languages block")
	}
	if !strings.Contains(overview, `viewBox="0 0 495 `) || !strings.Contains(langs, `viewBox="0 0 495 `) {
		t.Fatal("overview inherits card and a wide card with one block falls back to 495")
	}
}

func TestCardsWithoutCardsWritesStats(t *testing.T) {
	files, err := render.Cards(sample.Report(), config.Default(), render.Light)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files["stats-light.svg"] == "" {
		t.Fatalf("files = %v", files)
	}
}

func TestSnippetListsEveryCard(t *testing.T) {
	cfg := config.Default()
	cfg.Render.Cards = []config.Card{
		{Name: "overview", Blocks: []config.Block{{Name: "stats", Variants: []string{"totals"}}}},
		{Name: "langs", Layout: config.LayoutWide, Blocks: []config.Block{{Name: "languages", Variants: []string{"treemap"}}, {Name: "history", Variants: []string{"views"}}}},
	}
	s := render.Snippet(cfg)
	order := []string{
		`<img alt="GitHub stats: overview" src="ghstats/overview-dark.svg" width="495">`,
		`<img alt="GitHub stats: langs" src="ghstats/langs-dark.svg" width="100%">`,
	}
	pos := 0
	for _, want := range order {
		i := strings.Index(s[pos:], want)
		if i < 0 {
			t.Fatalf("missing or out of order %q in\n%s", want, s)
		}
		pos += i + len(want)
	}
	if strings.Contains(s, "stats-dark.svg") {
		t.Fatalf("no default stats card expected:\n%s", s)
	}
}

func TestCardsHidePartialShowsNAOnPartialMetrics(t *testing.T) {
	rep := sample.Report()
	for i, m := range rep.Distribution {
		if m.Channel == report.ChannelViews && m.Window == report.WindowLast14Days {
			rep.Distribution[i].Reliability = report.ReliabilityPartial
		}
	}
	cfg := config.Default()
	cfg.Render.Cards = []config.Card{{Name: "traffic", Blocks: []config.Block{{Name: "stats", Variants: []string{"distribution"}}}}}
	files, err := render.Cards(rep, cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files["traffic-dark.svg"], ">1,905<") {
		t.Fatal("a partial count must be shown by default")
	}
	cfg.Distribution.HidePartial = true
	files, err = render.Cards(rep, cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	svg := files["traffic-dark.svg"]
	if strings.Contains(svg, ">1,905<") || !strings.Contains(svg, ">n/a<") {
		t.Fatal("a partial count must be n/a with hide_partial")
	}
	if !strings.Contains(svg, ">5,000<") {
		t.Fatal("an observed count must stay visible")
	}
}
