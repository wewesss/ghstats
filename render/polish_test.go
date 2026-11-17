package render_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/report"
)

func historyReport(weekVals, dayVals []int64) report.Report {
	r := sampleReport()
	trend := report.Trend{Channel: report.ChannelViews}
	ws := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	for i, v := range weekVals {
		trend.Weeks = append(trend.Weeks, report.WeekCount{Start: ws.AddDate(0, 0, 7*i).Format("2006-01-02"), Count: v})
	}
	ds := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	for i, v := range dayVals {
		trend.Days = append(trend.Days, report.DayCount{Day: ds.AddDate(0, 0, i).Format("2006-01-02"), Count: v})
	}
	r.Trends = []report.Trend{trend}
	return r
}

func padded(n int, tail ...int64) []int64 {
	out := make([]int64, n)
	copy(out[n-len(tail):], tail)
	return out
}

func filled(n int, v int64) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = v
	}
	return out
}

var plotRe = regexp.MustCompile(`d="(M[^"]*)" pathLength`)

func plottedPoints(t *testing.T, svg string) int {
	t.Helper()
	m := plotRe.FindStringSubmatch(svg)
	if m == nil {
		t.Fatal("no plotted line")
	}
	return strings.Count(m[1], "L") + 1
}

func TestHistoryTrimsYoungHistories(t *testing.T) {
	cfg := config.DefaultRender()
	cfg.Blocks = []config.Block{{Name: "history", Variants: []string{"views"}}}
	cfg.Motion = false

	tests := []struct {
		name   string
		weeks  []int64
		days   []int64
		label  string
		points int
	}{
		{"all 26 weeks", filled(26, 5), padded(28, 1), "VIEWS · 26 WEEKS", 26},
		{"6 weeks", padded(26, 1, 2, 3, 4, 5, 6), padded(28, 1), "VIEWS · SINCE 12 OCT", 6},
		{"4 weeks", padded(26, 1, 2, 3, 4), padded(28, 1), "VIEWS · SINCE 26 OCT", 4},
		{"2 weeks fall back to days", padded(26, 3, 4), padded(28, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10), "VIEWS · DAILY · SINCE 8 NOV", 10},
		{"1 day keeps 2 points", padded(26, 0, 5), padded(28, 5), "VIEWS · DAILY · SINCE 16 NOV", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg := renderSVG(t, historyReport(tt.weeks, tt.days), cfg, render.Dark)
			assertWellFormed(t, svg)
			if !strings.Contains(svg, ">"+tt.label+"<") {
				t.Fatalf("label %q missing", tt.label)
			}
			if got := plottedPoints(t, svg); got != tt.points {
				t.Fatalf("points = %d, want %d", got, tt.points)
			}
		})
	}

	svg := renderSVG(t, historyReport(padded(26), padded(28)), cfg, render.Dark)
	if !strings.Contains(svg, "no traffic data yet") {
		t.Fatal("no-data message missing")
	}
}

func TestShiftDroppedWhenRawEqualsAdjusted(t *testing.T) {
	r := sampleReport()
	r.Languages.Display = r.Languages.Raw
	r.Languages.Multipliers = nil

	cfg := config.DefaultRender()
	cfg.Blocks = []config.Block{{Name: "languages", Variants: []string{"shift"}}}
	svg := renderSVG(t, r, cfg, render.Dark)
	assertWellFormed(t, svg)
	if strings.Contains(svg, "RAW → ADJUSTED") || !strings.Contains(svg, "LANGUAGES · BARS") {
		t.Fatal("a lone shift variant was not replaced by bars")
	}
	if strings.Contains(svg, `class="slide"`) || strings.Contains(svg, `class="dot"`) {
		t.Fatal("a single remaining variant still rotates")
	}

	cfg.Blocks = []config.Block{{Name: "languages", Variants: []string{"bars", "shift", "cells"}}}
	svg = renderSVG(t, r, cfg, render.Dark)
	assertWellFormed(t, svg)
	if strings.Contains(svg, "RAW → ADJUSTED") {
		t.Fatal("shift kept although raw equals adjusted")
	}
	if got := strings.Count(svg, `class="slide"`); got != 2 {
		t.Fatalf("slides = %d, want 2", got)
	}
	if !strings.Contains(svg, ".gsb0 .slide{animation:gsb0slide 10s linear infinite}") {
		t.Fatal("rotation timing does not use the remaining variant count")
	}

	cfg.Blocks = []config.Block{{Name: "languages", Variants: []string{"shift"}}}
	if svg := renderSVG(t, sampleReport(), cfg, render.Dark); !strings.Contains(svg, "RAW → ADJUSTED") {
		t.Fatal("shift dropped although raw and adjusted differ")
	}
}

func TestShiftStrokeWidthIsCapped(t *testing.T) {
	cfg := config.DefaultRender()
	cfg.Blocks = []config.Block{{Name: "languages", Variants: []string{"shift"}}}
	cfg.Motion = false
	svg := renderSVG(t, sampleReport(), cfg, render.Dark)
	if !strings.Contains(svg, `stroke-width="4.0"`) || strings.Contains(svg, `stroke-width="4.9"`) {
		t.Fatal("a 31% curve is not capped at 4")
	}
	if !strings.Contains(svg, `stroke-width="1.6"`) {
		t.Fatal("a 5% curve should be 1.6 wide")
	}
}

func TestFoldedLanguagesAreNamedOther(t *testing.T) {
	build := func(extra string) report.Report {
		r := sampleReport()
		r.Languages.Raw, r.Languages.Display, r.Languages.Multipliers = nil, nil, nil
		for _, n := range []string{"A1", "B2", "C3", "D4", "E5", "F6", "G7", "H8", "I9"} {
			r.Languages.Raw = append(r.Languages.Raw, report.LangShare{Name: n, Share: .1})
			r.Languages.Display = append(r.Languages.Display, report.LangShare{Name: n, Share: .1})
		}
		if extra != "" {
			r.Languages.Display = append(r.Languages.Display, report.LangShare{Name: extra, Share: .05})
		}
		return r
	}
	cfg := config.DefaultRender()
	cfg.Blocks = []config.Block{{Name: "languages", Variants: []string{"bars"}}}
	cfg.Motion = false

	svg := renderSVG(t, build(""), cfg, render.Dark)
	if !strings.Contains(svg, ">"+config.Default().Languages.OtherLabel+"<") || strings.Contains(svg, ">G7<") {
		t.Fatal("folded entry is not named with the default Other label")
	}
	svg = renderSVG(t, build("Misc"), cfg, render.Dark)
	if !strings.Contains(svg, ">Misc<") || strings.Contains(svg, ">Other<") || strings.Contains(svg, ">I9<") {
		t.Fatal("the report's own Other name was not reused")
	}
}

var doubleStyleRe = regexp.MustCompile(`style="[^"]*"[^<>]*\sstyle="`)

func TestCycleAccentUsesACustomProperty(t *testing.T) {
	svg := renderSVG(t, sampleReport(), fullConfig(), render.Dark)
	assertWellFormed(t, svg)
	for _, want := range []string{
		"@property --ghs-accent{syntax:'&lt;color&gt;';inherits:true;initial-value:#9ec5ff}",
		"@keyframes gsaccent{",
		"@media (prefers-reduced-motion: reduce){.gs,.gs *{animation:none!important}}",
		"fill:var(--ghs-accent)",
		"stroke:var(--ghs-accent)",
		"stop-color:var(--ghs-accent)",
		`fill="#9ec5ff" style="fill:var(--ghs-accent)`,
	} {
		if !strings.Contains(svg, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(svg, "<animate") {
		t.Fatal("SMIL animation left in the output")
	}
	if doubleStyleRe.MatchString(svg) {
		t.Fatal("an element carries two style attributes")
	}

	cfg := fullConfig()
	cfg.Theme.Accent = config.Accent{Name: "mint"}
	plain := renderSVG(t, sampleReport(), cfg, render.Dark)
	if strings.Contains(plain, "--ghs-accent") || strings.Contains(plain, "<animate") {
		t.Fatal("a fixed accent uses the cycle machinery")
	}
}
