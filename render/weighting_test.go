package render_test

import (
	"bytes"
	"encoding/xml"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/sample"
)

var translateRe = regexp.MustCompile(`translate\(([\d.]+) ([\d.]+)\)`)

func attr(se xml.StartElement, name string) string {
	for _, a := range se.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func number(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func textsAndBottom(t *testing.T, svg string) (texts []string, bottom float64) {
	t.Helper()
	type frame struct {
		offset float64
		text   bool
	}
	dec := xml.NewDecoder(strings.NewReader(svg))
	var stack []frame
	var cur strings.Builder
	inText := func() bool { return slices.ContainsFunc(stack, func(f frame) bool { return f.text }) }
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch e := tok.(type) {
		case xml.StartElement:
			offset := 0.0
			if n := len(stack); n > 0 {
				offset = stack[n-1].offset
			}
			f := frame{offset: offset}
			switch e.Name.Local {
			case "g":
				if m := translateRe.FindStringSubmatch(attr(e, "transform")); m != nil {
					f.offset += number(m[2])
				}
			case "text":
				if !inText() {
					f.text = true
					cur.Reset()
					bottom = max(bottom, offset+number(attr(e, "y"))+3)
				}
			case "rect":
				if number(attr(e, "width")) < 400 && attr(e, "class") != "blink" {
					bottom = max(bottom, offset+number(attr(e, "y"))+number(attr(e, "height")))
				}
			}
			stack = append(stack, f)
		case xml.EndElement:
			f := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if f.text {
				texts = append(texts, cur.String())
			}
		case xml.CharData:
			if inText() {
				cur.Write(e)
			}
		}
	}
	return texts, bottom
}

func weightingTraces(texts []string) []string {
	var out []string
	for _, s := range texts {
		low := strings.ToLower(strings.TrimSpace(s))
		switch {
		case strings.Contains(s, "×"):
			out = append(out, "multiplier label "+s)
		case low == "adjusted" || low == "raw":
			out = append(out, "mode tag "+s)
		case s == "RAW" || s == "ADJ":
			out = append(out, "strip label "+s)
		case strings.Contains(s, "--adjusted") || strings.Contains(s, "--raw"):
			out = append(out, "terminal flag "+s)
		}
	}
	return out
}

func weightingCfg(variant string, weighting bool) config.Render {
	cfg := config.DefaultRender()
	cfg.Blocks = []config.Block{{Name: "languages", Variants: []string{variant}}}
	cfg.Motion = false
	cfg.Languages.Weighting = &weighting
	return cfg
}

func TestWeightingFalseLeavesNoTraceInAnyVariant(t *testing.T) {
	for _, mode := range []string{"adjusted", "raw"} {
		rep := sample.Report()
		rep.Languages.Mode = mode
		if mode == "raw" {
			rep.Languages.Multipliers = nil
		}
		for _, variant := range []string{"bars", "iso", "cells", "treemap", "terminal"} {
			t.Run(mode+"/"+variant, func(t *testing.T) {
				on, _ := textsAndBottom(t, renderSVG(t, rep, weightingCfg(variant, true), render.Dark))
				if len(weightingTraces(on)) == 0 {
					t.Fatalf("weighting: true must show the weighting, texts = %q", on)
				}
				off, _ := textsAndBottom(t, renderSVG(t, rep, weightingCfg(variant, false), render.Dark))
				if got := weightingTraces(off); len(got) != 0 {
					t.Fatalf("weighting: false left traces %q", got)
				}
			})
		}
	}
}

func TestWeightingDefaultShowsEveryTraceOnBars(t *testing.T) {
	cfg := weightingCfg("bars", true)
	cfg.Languages.Weighting = nil
	texts, _ := textsAndBottom(t, renderSVG(t, sample.Report(), cfg, render.Dark))
	joined := "\n" + strings.Join(texts, "\n") + "\n"
	for _, want := range []string{"\n×0.35\n", "\nadjusted\n", "\nRAW\n", "\nADJ\n"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("default bars lack %q in %q", strings.TrimSpace(want), texts)
		}
	}
}

func TestWeightingFalseTerminalCommandHasNoFlag(t *testing.T) {
	on, _ := textsAndBottom(t, renderSVG(t, sample.Report(), weightingCfg("terminal", true), render.Dark))
	if !slices.Contains(on, "$ ghstats languages --adjusted") {
		t.Fatalf("default terminal command missing: %q", on)
	}
	off, _ := textsAndBottom(t, renderSVG(t, sample.Report(), weightingCfg("terminal", false), render.Dark))
	if !slices.Contains(off, "$ ghstats languages") {
		t.Fatalf("terminal command must read exactly %q: %q", "$ ghstats languages", off)
	}
}

func TestWeightingFalseBarsCardClosesTheStrip(t *testing.T) {
	rep := sample.Report()
	svgOn := renderSVG(t, rep, weightingCfg("bars", true), render.Dark)
	svgOff := renderSVG(t, rep, weightingCfg("bars", false), render.Dark)
	_, hOn := viewBox(t, svgOn)
	_, hOff := viewBox(t, svgOff)
	if hOn-hOff < 40 {
		t.Fatalf("card height %d with weighting, %d without: the strip must be gone from the height", hOn, hOff)
	}
	_, bottomOn := textsAndBottom(t, svgOn)
	_, bottomOff := textsAndBottom(t, svgOff)
	gapOn, gapOff := float64(hOn)-bottomOn, float64(hOff)-bottomOff
	if gapOff > gapOn {
		t.Fatalf("empty band under the bars: %.1f px to the card bottom, %.1f with the strip", gapOff, gapOn)
	}
	if gapOff > 24 {
		t.Fatalf("last element ends %.1f px above the card bottom", gapOff)
	}
}

func TestWeightingFalseBarsHeightFollowsTheRows(t *testing.T) {
	rep := sample.Report()
	rep.Languages.Display = rep.Languages.Display[:3]
	svg := renderSVG(t, rep, weightingCfg("bars", false), render.Dark)
	_, h := viewBox(t, svg)
	_, bottom := textsAndBottom(t, svg)
	if gap := float64(h) - bottom; gap > 24 {
		t.Fatalf("3 rows leave %.1f px under the last bar", gap)
	}
}

func TestWeightingFalseRefusesShift(t *testing.T) {
	off := false
	const want = "render.languages.weighting: false hides the weighting that the shift variant shows; remove shift or set weighting: true"

	blocks := config.DefaultRender()
	blocks.Blocks = []config.Block{{Name: "languages", Variants: []string{"bars", "shift"}}}
	blocks.Languages.Weighting = &off

	cards := config.DefaultRender()
	cards.Cards = []config.Card{
		{Name: "a", Blocks: []config.Block{{Name: "stats", Variants: []string{"totals"}}}},
		{Name: "b", Blocks: []config.Block{{Name: "languages", Variants: []string{"shift"}}}},
	}
	cards.Languages.Weighting = &off

	for name, cfg := range map[string]config.Render{"blocks": blocks, "cards": cards} {
		var buf bytes.Buffer
		err := render.SVG(&buf, sample.Report(), cfg, render.Dark)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: err = %v, want %q", name, err, want)
		}
	}
	full := sample.Profile()
	full.Render = cards
	if _, err := render.Cards(sample.Report(), full, render.Dark); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("cards: err = %v, want %q", err, want)
	}
}
