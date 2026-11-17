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
	"github.com/wewesss/ghstats/snapshot"
)

var update = flag.Bool("update", false, "rewrite the golden files")

var allBlocks = []config.Block{
	{Name: "stats", Variants: []string{"totals", "distribution"}},
	{Name: "languages", Variants: []string{"bars", "iso", "cells", "treemap", "shift", "terminal"}},
	{Name: "history", Variants: []string{"views", "clones"}},
}

func fullConfig() config.Render {
	c := config.DefaultRender()
	c.Blocks = allBlocks
	c.Theme.Accent = config.Accent{Name: "cycle"}
	return c
}

func i64(v int64) *int64 { return &v }

func sampleReport() report.Report {
	lang := func(name string, share float64) report.LangShare { return report.LangShare{Name: name, Share: share} }
	weeks := func(vals ...int64) []report.WeekCount {
		start := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
		out := make([]report.WeekCount, len(vals))
		for i, v := range vals {
			out[i] = report.WeekCount{Start: start.AddDate(0, 0, 7*i).Format("2006-01-02"), Count: v}
		}
		return out
	}
	return report.Report{
		SchemaVersion: report.SchemaVersion,
		GeneratedAt:   time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC),
		CollectedAt:   time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC),
		Login:         "wewesss",
		Totals:        report.Totals{Repos: 42, PublicRepos: 31, PrivateRepos: 11, Stars: 128, Forks: 17, Followers: 56},
		Languages: report.Languages{
			Mode: "adjusted",
			Raw: []report.LangShare{
				lang("Lua", .317), lang("TypeScript", .246), lang("Go", .175), lang("Makefile", .087),
				lang("Rust", .079), lang("Python", .056), lang("Shell", .04),
			},
			Display: []report.LangShare{
				lang("TypeScript", .31), lang("Go", .22), lang("Lua", .14), lang("Rust", .10),
				lang("Python", .07), lang("Shell", .05), lang("Other", .11),
			},
			Multipliers: map[string]float64{"Lua": 0.35},
		},
		Distribution: []report.Metric{
			{Channel: report.ChannelReleaseAssets, Count: i64(3412), Window: report.WindowAllTime, Reliability: report.ReliabilityObserved},
			{Channel: report.ChannelClones, Count: i64(287), Window: report.WindowLast14Days, Reliability: report.ReliabilityObserved},
			{Channel: report.ChannelClones, Count: i64(900), Window: "since:2026-01-01", Reliability: report.ReliabilityObserved},
			{Channel: report.ChannelViews, Count: i64(1905), Window: report.WindowLast14Days, Reliability: report.ReliabilityObserved},
			{Channel: report.ChannelViews, Count: i64(5000), Window: "since:2026-01-01", Reliability: report.ReliabilityObserved},
			{Channel: report.ChannelSourceArchives, Window: report.WindowAllTime, Reliability: report.ReliabilityUnavailable},
		},
		Trends: []report.Trend{
			{Channel: report.ChannelClones, Weeks: weeks(8, 12, 9, 15, 11, 7, 14, 18, 13, 10, 21, 17, 12, 19, 25, 16, 14, 22, 28, 19, 17, 24, 31, 22, 27, 35)},
			{Channel: report.ChannelViews, Weeks: weeks(42, 38, 51, 47, 60, 55, 49, 72, 66, 58, 81, 77, 69, 90, 84, 79, 102, 95, 88, 110, 97, 121, 115, 108, 134, 128)},
		},
		Warnings: []string{},
	}
}

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
	r := sampleReport()
	for _, mode := range []render.Mode{render.Dark, render.Light} {
		t.Run("default-"+string(mode), func(t *testing.T) {
			svg := renderSVG(t, r, config.DefaultRender(), mode)
			assertWellFormed(t, svg)
			golden(t, "default-"+string(mode)+".svg", svg)
		})
		t.Run("full-"+string(mode), func(t *testing.T) {
			svg := renderSVG(t, r, fullConfig(), mode)
			assertWellFormed(t, svg)
			golden(t, "full-"+string(mode)+".svg", svg)
		})
	}
}

func TestGoldenLanguageVariants(t *testing.T) {
	r := sampleReport()
	for _, v := range []string{"bars", "iso", "cells", "treemap", "shift", "terminal"} {
		t.Run(v, func(t *testing.T) {
			cfg := config.DefaultRender()
			cfg.Blocks = []config.Block{{Name: "languages", Variants: []string{v}}}
			cfg.Motion = false
			svg := renderSVG(t, r, cfg, render.Dark)
			assertWellFormed(t, svg)
			golden(t, "languages-"+v+".svg", svg)
		})
	}
}

func TestEveryPresetPaletteAndMode(t *testing.T) {
	r := sampleReport()
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
						cfg := fullConfig()
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
	r := sampleReport()
	accents := []config.Accent{
		{Name: "ice"}, {Name: "mint"}, {Name: "amber"}, {Name: "silver"}, {Name: "violet"}, {Name: "rose"}, {Name: "lime"}, {Name: "coral"},
		{Name: "cycle"}, {Dark: "#112233", Light: "#445566"},
	}
	for _, a := range accents {
		for _, mode := range []render.Mode{render.Dark, render.Light} {
			cfg := fullConfig()
			cfg.Theme.Accent = a
			cfg.Languages.Colors = config.Palette{Name: "duotone"}
			assertWellFormed(t, renderSVG(t, r, cfg, mode))
		}
	}
}

func TestOverridesApplyLast(t *testing.T) {
	cfg := config.DefaultRender()
	cfg.Theme.Overrides = map[string]string{"bg": "#123456", "border": "#abcdef", "fg": "#fedcba"}
	svg := renderSVG(t, sampleReport(), cfg, render.Dark)
	for _, want := range []string{`fill="#123456"`, `stroke="#abcdef"`, `fill="#fedcba"`} {
		if !strings.Contains(svg, want) {
			t.Fatalf("missing %s", want)
		}
	}
	cfg.Theme.Overrides = map[string]string{"bg": "none"}
	if svg := renderSVG(t, sampleReport(), cfg, render.Dark); !strings.Contains(svg, `rx="10" fill="none"`) {
		t.Fatal("bg none not applied")
	}
}

func TestAnimationRules(t *testing.T) {
	svg := renderSVG(t, sampleReport(), fullConfig(), render.Dark)
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

	cfg := fullConfig()
	cfg.Motion = false
	still := renderSVG(t, sampleReport(), cfg, render.Dark)
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
	svg := renderSVG(t, sampleReport(), cfg, render.Dark)
	for _, want := range []string{">128<", ">42<", ">17<", ">56<", "PRIVATE 11", "names hidden", "×0.35", ">31.0%<", "synced · 17 Nov", ">wewesss<", ">adjusted<"} {
		if !strings.Contains(svg, want) {
			t.Fatalf("missing %q", want)
		}
	}

	cfg.PrivateBar = false
	if svg := renderSVG(t, sampleReport(), cfg, render.Dark); strings.Contains(svg, "PRIVATE") {
		t.Fatal("private bar rendered while disabled")
	}

	r := sampleReport()
	r.Totals.PrivateRepos = 0
	cfg.PrivateBar = true
	if svg := renderSVG(t, r, cfg, render.Dark); strings.Contains(svg, "PRIVATE") {
		t.Fatal("private bar rendered without private repositories")
	}
}

func TestThousandsSeparator(t *testing.T) {
	r := sampleReport()
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
	svg := renderSVG(t, sampleReport(), cfg, render.Dark)
	if !strings.Contains(svg, ">3,412<") || !strings.Contains(svg, ">287<") || !strings.Contains(svg, ">1,905<") {
		t.Fatal("distribution counts missing")
	}
	if strings.Count(svg, ">n/a<") != 1 {
		t.Fatalf("want exactly one n/a (source archives), got %d", strings.Count(svg, ">n/a<"))
	}

	r := sampleReport()
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
	r := sampleReport()
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
	r := sampleReport()
	r.Languages = report.Languages{Mode: "raw"}
	r.Trends = nil
	r.Distribution = nil
	svg := renderSVG(t, r, fullConfig(), render.Light)
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
	for _, cfg := range []config.Render{config.DefaultRender(), fullConfig()} {
		for _, mode := range []render.Mode{render.Dark, render.Light} {
			assertWellFormed(t, renderSVG(t, r, cfg, mode))
		}
	}
	assertWellFormed(t, renderSVG(t, report.Report{}, fullConfig(), render.Dark))
	var buf bytes.Buffer
	if err := render.JSON(&buf, r); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidInputsAreRejected(t *testing.T) {
	var buf bytes.Buffer
	if err := render.SVG(&buf, sampleReport(), config.DefaultRender(), "sepia"); err == nil {
		t.Fatal("unknown mode accepted")
	}
	bad := config.DefaultRender()
	bad.Theme.Preset = "neon"
	if err := render.SVG(&buf, sampleReport(), bad, render.Dark); err == nil {
		t.Fatal("invalid configuration accepted")
	}
}

func TestTextIsEscaped(t *testing.T) {
	r := sampleReport()
	r.Login = `a<b>&"c`
	r.Languages.Display[0].Name = "C<&>"
	r.Languages.Raw[1].Name = "C<&>"
	assertWellFormed(t, renderSVG(t, r, fullConfig(), render.Dark))
}

func TestDeterministic(t *testing.T) {
	r := sampleReport()
	cfg := fullConfig()
	cfg.Languages.Colors = config.Palette{Custom: map[string]string{"Go": "#00add8", "Rust": "#dea584"}}
	first := renderSVG(t, r, cfg, render.Dark)
	for range 5 {
		if renderSVG(t, r, cfg, render.Dark) != first {
			t.Fatal("output changes between runs")
		}
	}
}

func TestJSON(t *testing.T) {
	var a, b bytes.Buffer
	r := sampleReport()
	if err := render.JSON(&a, r); err != nil {
		t.Fatal(err)
	}
	if err := render.JSON(&b, r); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Fatal("JSON is not deterministic")
	}
	out := a.String()
	if !strings.HasPrefix(out, "{\n  \"schema_version\": 1,") || !strings.HasSuffix(out, "}\n") {
		t.Fatalf("unexpected shape: %q", out[:min(len(out), 60)])
	}
}

func TestPrivateRepositoryNamesNeverLeak(t *testing.T) {
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	day := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	secret := snapshot.Repo{
		ID: "ID_secret", NameWithOwner: "me/secret-repo", Owner: "me", Private: true, Stars: 2,
		Languages: []snapshot.LanguageSize{{Name: "Go", Bytes: 5000}},
		Traffic: &snapshot.Traffic{
			Views:  []snapshot.DayCount{{Day: day, Count: 9, Uniques: 3}},
			Clones: []snapshot.DayCount{{Day: day, Count: 4, Uniques: 1}},
		},
	}
	public := snapshot.Repo{
		ID: "ID_pub", NameWithOwner: "me/open", Owner: "me", Stars: 1,
		Languages: []snapshot.LanguageSize{{Name: "Lua", Bytes: 1000}},
	}
	s := snapshot.Snapshot{
		SchemaVersion: snapshot.SchemaVersion, CollectedAt: now,
		Viewer: snapshot.Viewer{Login: "me", Followers: 3},
		Repos:  []snapshot.Repo{secret, public},
		Errors: []snapshot.RepoError{{Repo: "me/secret-repo", Stage: "traffic", Message: "boom secret-repo"}},
	}
	r, err := compute.Compute(s, history.History{}, config.Default(), now)
	if err != nil {
		t.Fatal(err)
	}
	if r.Totals.PrivateRepos != 1 {
		t.Fatalf("totals = %+v", r.Totals)
	}
	for _, mode := range []render.Mode{render.Dark, render.Light} {
		svg := renderSVG(t, r, fullConfig(), mode)
		assertWellFormed(t, svg)
		if strings.Contains(svg, "secret-repo") {
			t.Fatal("SVG leaks a private repository name")
		}
		if !strings.Contains(svg, "PRIVATE 1") {
			t.Fatal("private bar missing")
		}
	}
	var buf bytes.Buffer
	if err := render.JSON(&buf, r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "secret-repo") {
		t.Fatal("JSON leaks a private repository name")
	}
}
