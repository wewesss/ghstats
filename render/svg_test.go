package render_test

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wewesss/ghstats/compute"
	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/history"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/report"
	"github.com/wewesss/ghstats/sample"
	"github.com/wewesss/ghstats/snapshot"
)

var update = flag.Bool("update", false, "rewrite the golden files")

func i64(v int64) *int64 { return &v }

func renderSVG(t *testing.T, r report.Report, cfg config.Render, mode render.Mode) string {
	t.Helper()
	var buf bytes.Buffer
	if err := render.SVG(&buf, r, cfg, mode); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func assertWellFormed(t *testing.T, svg string) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(svg))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("not well-formed XML: %v", err)
		}
	}
	for _, bad := range []string{"NaN", "Inf", "__", "<nil>"} {
		if strings.Contains(svg, bad) {
			i := strings.Index(svg, bad)
			t.Fatalf("output contains %q near %q", bad, svg[max(0, i-60):min(len(svg), i+60)])
		}
	}
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test ./render -update)", err)
	}
	if string(want) != got {
		t.Fatalf("%s differs from the rendered output (run go test ./render -update and review the diff)", name)
	}
}

func TestGoldenCards(t *testing.T) {
	r := sample.Report()
	for _, mode := range []render.Mode{render.Dark, render.Light} {
		t.Run("default-"+string(mode), func(t *testing.T) {
			svg := renderSVG(t, r, config.DefaultRender(), mode)
			assertWellFormed(t, svg)
			golden(t, "default-"+string(mode)+".svg", svg)
		})
		t.Run("full-"+string(mode), func(t *testing.T) {
			svg := renderSVG(t, r, sample.FullRender(), mode)
			assertWellFormed(t, svg)
			golden(t, "full-"+string(mode)+".svg", svg)
		})
	}
}

func TestEveryPresetPaletteAndMode(t *testing.T) {
	r := sample.Report()
	presets := []string{"noir", "carbon", "graphite", "midnight", "slate", "dune", "transparent"}
	palettes := []config.Palette{
		{Name: "accent"}, {Name: "github"}, {Name: "mono"}, {Name: "spectrum"}, {Name: "duotone"},
		{Custom: map[string]string{"TypeScript": "#3178C6", "go": "#00add8"}},
	}
	for _, preset := range presets {
		for i, pal := range palettes {
			for _, mode := range []render.Mode{render.Dark, render.Light} {
				for _, motion := range []bool{true, false} {
					t.Run(fmt.Sprintf("%s/%d/%s/%v", preset, i, mode, motion), func(t *testing.T) {
						cfg := sample.FullRender()
						cfg.Theme.Preset = preset
						cfg.Languages.Colors = pal
						cfg.Motion = motion
						assertWellFormed(t, renderSVG(t, r, cfg, mode))
					})
				}
			}
		}
	}
}

func TestEveryAccent(t *testing.T) {
	r := sample.Report()
	accents := []config.Accent{
		{Name: "ice"}, {Name: "mint"}, {Name: "amber"}, {Name: "silver"}, {Name: "violet"}, {Name: "rose"}, {Name: "lime"}, {Name: "coral"},
		{Name: "cycle"}, {Dark: "#112233", Light: "#445566"},
	}
	for _, a := range accents {
		for _, mode := range []render.Mode{render.Dark, render.Light} {
			cfg := sample.FullRender()
			cfg.Theme.Accent = a
			cfg.Languages.Colors = config.Palette{Name: "duotone"}
			assertWellFormed(t, renderSVG(t, r, cfg, mode))
		}
	}
}

func TestOverridesApplyLast(t *testing.T) {
	cfg := config.DefaultRender()
	cfg.Theme.Overrides = map[string]string{"bg": "#123456", "border": "#abcdef", "fg": "#fedcba"}
	svg := renderSVG(t, sample.Report(), cfg, render.Dark)
	for _, want := range []string{`fill="#123456"`, `stroke="#abcdef"`, `fill="#fedcba"`} {
		if !strings.Contains(svg, want) {
			t.Fatalf("missing %s", want)
		}
	}
	cfg.Theme.Overrides = map[string]string{"bg": "none"}
	if svg := renderSVG(t, sample.Report(), cfg, render.Dark); !strings.Contains(svg, `rx="10" fill="none"`) {
		t.Fatal("bg none not applied")
	}
}

func TestAnimationRules(t *testing.T) {
	svg := renderSVG(t, sample.Report(), sample.FullRender(), render.Dark)
	for _, want := range []string{
		"@media (prefers-reduced-motion: reduce)",
		"animation-name:gsb1fade_200;animation-delay:0s",
		"animation-name:gsb1fade_500;animation-delay:5s",
		`<g class="slide" style="animation-delay:0s" opacity="1">`,
		`<g class="slide" style="animation-delay:5s" opacity="0">`,
		"@property --ghs-accent",
		".gs,.gs *{animation:none!important}",
	} {
		if !strings.Contains(svg, want) {
			t.Fatalf("missing %q", want)
		}
	}

	cfg := sample.FullRender()
	cfg.Motion = false
	still := renderSVG(t, sample.Report(), cfg, render.Dark)
	for _, bad := range []string{"@keyframes", "prefers-reduced-motion", "<animate", `class="slide"`, `class="dot"`} {
		if strings.Contains(still, bad) {
			t.Fatalf("motion off still contains %q", bad)
		}
	}
}

func TestDataMapping(t *testing.T) {
	cfg := config.DefaultRender()
	cfg.Blocks = []config.Block{{Name: "stats", Variants: []string{"totals"}}, {Name: "languages", Variants: []string{"bars"}}}
	cfg.Motion = false
	svg := renderSVG(t, sample.Report(), cfg, render.Dark)
	for _, want := range []string{">128<", ">42<", ">17<", ">56<", "PRIVATE 11", "names hidden", "×0.35", ">31.0%<", "synced · 17 Nov", ">wewesss<", ">adjusted<"} {
		if !strings.Contains(svg, want) {
			t.Fatalf("missing %q", want)
		}
	}

	cfg.PrivateBar = false
	if svg := renderSVG(t, sample.Report(), cfg, render.Dark); strings.Contains(svg, "PRIVATE") {
		t.Fatal("private bar rendered while disabled")
	}

	r := sample.Report()
	r.Totals.PrivateRepos = 0
	cfg.PrivateBar = true
	if svg := renderSVG(t, r, cfg, render.Dark); strings.Contains(svg, "PRIVATE") {
		t.Fatal("private bar rendered without private repositories")
	}
}

func TestThousandsSeparator(t *testing.T) {
	r := sample.Report()
	r.Totals.Stars = 3412
	r.Totals.Followers = 1234567
	cfg := config.DefaultRender()
	cfg.Blocks = []config.Block{{Name: "stats", Variants: []string{"totals"}}}
	svg := renderSVG(t, r, cfg, render.Dark)
	if !strings.Contains(svg, ">3,412<") || !strings.Contains(svg, ">1,234,567<") {
		t.Fatal("thousands separators missing")
	}
}

func TestDistributionShowsNotAvailable(t *testing.T) {
	cfg := config.DefaultRender()
	cfg.Blocks = []config.Block{{Name: "stats", Variants: []string{"distribution"}}}
	cfg.Motion = false
	svg := renderSVG(t, sample.Report(), cfg, render.Dark)
	if !strings.Contains(svg, ">3,412<") || !strings.Contains(svg, ">287<") || !strings.Contains(svg, ">1,905<") {
		t.Fatal("distribution counts missing")
	}
	if !strings.Contains(svg, ">5,000<") || !strings.Contains(svg, ">VIEWS TOTAL<") {
		t.Fatal("cumulative views missing")
	}
	if strings.Contains(svg, ">n/a<") || strings.Contains(svg, "ARCHIVES") {
		t.Fatal("source archives must not be shown")
	}

	r := sample.Report()
	r.Distribution = []report.Metric{
		{Channel: report.ChannelClones, Window: report.WindowLast14Days, Reliability: report.ReliabilityUnavailable},
		{Channel: report.ChannelViews, Count: i64(0), Window: report.WindowLast14Days, Reliability: report.ReliabilityUnavailable},
	}
	svg = renderSVG(t, r, cfg, render.Dark)
	if strings.Count(svg, ">n/a<") != 4 {
		t.Fatalf("want four n/a, got %d", strings.Count(svg, ">n/a<"))
	}
	if strings.Contains(svg, ">0<") {
		t.Fatal("an unavailable metric rendered as 0")
	}
}

func TestMoreThanSevenLanguagesFoldIntoTheLastEntry(t *testing.T) {
	r := sample.Report()
	names := []string{"A1", "B2", "C3", "D4", "E5", "F6", "G7", "H8", "I9"}
	r.Languages.Raw = nil
	r.Languages.Display = nil
	r.Languages.Multipliers = nil
	for _, n := range names {
		r.Languages.Raw = append(r.Languages.Raw, report.LangShare{Name: n, Share: 1.0 / 9})
		r.Languages.Display = append(r.Languages.Display, report.LangShare{Name: n, Share: 1.0 / 9})
	}
	cfg := config.DefaultRender()
	cfg.Blocks = []config.Block{{Name: "languages", Variants: []string{"bars"}}}
	cfg.Motion = false
	svg := renderSVG(t, r, cfg, render.Dark)
	assertWellFormed(t, svg)
	if !strings.Contains(svg, ">Other<") || strings.Contains(svg, ">G7<") || strings.Contains(svg, ">H8<") || strings.Contains(svg, ">I9<") {
		t.Fatal("languages beyond the sixth were not folded into Other")
	}
	if !strings.Contains(svg, ">33.3%<") {
		t.Fatal("folded entry share missing")
	}
}

func TestMissingDataRendersFramesNotNumbers(t *testing.T) {
	r := sample.Report()
	r.Languages = report.Languages{Mode: "raw"}
	r.Trends = nil
	r.Distribution = nil
	svg := renderSVG(t, r, sample.FullRender(), render.Light)
	assertWellFormed(t, svg)
	if !strings.Contains(svg, "no language data yet") || !strings.Contains(svg, "no traffic data yet") {
		t.Fatal("empty messages missing")
	}
	if strings.Contains(svg, "<path") && strings.Contains(svg, "linearGradient") {
		t.Fatal("history drew a path without data")
	}
}

func TestEmptyReport(t *testing.T) {
	r, err := compute.Compute(snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion}, history.History{}, config.Default(), time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for _, cfg := range []config.Render{config.DefaultRender(), sample.FullRender()} {
		for _, mode := range []render.Mode{render.Dark, render.Light} {
			assertWellFormed(t, renderSVG(t, r, cfg, mode))
		}
	}
	assertWellFormed(t, renderSVG(t, report.Report{}, sample.FullRender(), render.Dark))
	var buf bytes.Buffer
	if err := render.JSON(&buf, r); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidInputsAreRejected(t *testing.T) {
	var buf bytes.Buffer
	if err := render.SVG(&buf, sample.Report(), config.DefaultRender(), "sepia"); err == nil {
		t.Fatal("unknown mode accepted")
	}
	bad := config.DefaultRender()
	bad.Theme.Preset = "neon"
	if err := render.SVG(&buf, sample.Report(), bad, render.Dark); err == nil {
		t.Fatal("invalid configuration accepted")
	}
}

func TestTextIsEscaped(t *testing.T) {
	r := sample.Report()
	r.Login = `a<b>&"c`
	r.Languages.Display[0].Name = "C<&>"
	r.Languages.Raw[1].Name = "C<&>"
	assertWellFormed(t, renderSVG(t, r, sample.FullRender(), render.Dark))
}

func TestDeterministic(t *testing.T) {
	r := sample.Report()
	cfg := sample.FullRender()
	cfg.Languages.Colors = config.Palette{Custom: map[string]string{"Go": "#00add8", "Rust": "#dea584"}}
	first := renderSVG(t, r, cfg, render.Dark)
	for range 5 {
		if renderSVG(t, r, cfg, render.Dark) != first {
			t.Fatal("output changes between runs")
		}
	}
}
