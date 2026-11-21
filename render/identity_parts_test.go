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

type svgText struct {
	x, y, size float64
	anchor     string
	body       string
}

var textRe = regexp.MustCompile(`<text x="([\d.]+)" y="([\d.]+)" font-family="[^"]*" font-size="([\d.]+)" font-weight="\d+" fill="[^"]*" text-anchor="(\w+)"[^>]*>(.*?)</text>`)

func texts(t *testing.T, svg string) []svgText {
	t.Helper()
	var out []svgText
	for _, m := range textRe.FindAllStringSubmatch(svg, -1) {
		x, _ := strconv.ParseFloat(m[1], 64)
		y, _ := strconv.ParseFloat(m[2], 64)
		size, _ := strconv.ParseFloat(m[3], 64)
		out = append(out, svgText{x: x, y: y, size: size, anchor: m[4], body: m[5]})
	}
	if len(out) == 0 {
		t.Fatal("no text element parsed")
	}
	return out
}

func findText(ts []svgText, sub string) (svgText, bool) {
	for _, s := range ts {
		if strings.Contains(s.body, sub) {
			return s, true
		}
	}
	return svgText{}, false
}

func identityCfg(show []string, layout string) config.Config {
	cfg := sample.Profile()
	cfg.Render.Motion = false
	cfg.Profile.Identity = config.IdentityOptions{Show: show, Layout: layout}
	return cfg
}

func TestIdentityShowDrawsOnlyListedParts(t *testing.T) {
	svg := renderIdentity(t, identityCfg([]string{"name", "location"}, ""), render.Dark)
	assertWellFormed(t, svg)
	ts := texts(t, svg)
	for _, want := range []string{"wewesss", "caen, fr"} {
		if _, ok := findText(ts, want); !ok {
			t.Errorf("listed part %q is missing", want)
		}
	}
	for _, absent := range []string{"/ profile", "SOFTWARE", "no fixed stack", "building", "FOCUS", "WEB &amp; DESKTOP", "github.com/"} {
		if _, ok := findText(ts, absent); ok {
			t.Errorf("part %q must not be drawn when it is not listed", absent)
		}
	}
	if strings.Contains(svg, `x1="640"`) {
		t.Error("no focus column rule expected")
	}
	full := renderIdentity(t, identityCfg(nil, ""), render.Dark)
	for _, present := range []string{"/ profile", "SOFTWARE", "no fixed stack", "building", "FOCUS", "github.com/"} {
		if _, ok := findText(texts(t, full), present); !ok {
			t.Errorf("without show, %q must be drawn", present)
		}
	}
}

func TestIdentityShowAdjustsHeightWithoutGaps(t *testing.T) {
	heightOf := func(show ...string) int {
		_, h := viewBox(t, renderIdentity(t, identityCfg(show, ""), render.Dark))
		return h
	}
	name := heightOf("name")
	withTagline := heightOf("name", "tagline")
	withNote := heightOf("name", "tagline", "note")
	withFooter := heightOf("name", "tagline", "note", "footer")
	if !(name < withTagline && withTagline < withNote && withNote < withFooter) {
		t.Fatalf("heights must grow with each part: %d %d %d %d", name, withTagline, withNote, withFooter)
	}
	if got, want := withTagline-name, 26; got != want {
		t.Fatalf("the tagline adds %d px, want its own height %d", got, want)
	}

	ts := texts(t, renderIdentity(t, identityCfg([]string{"name", "note"}, ""), render.Dark))
	nm, _ := findText(ts, "wewesss")
	note, _ := findText(ts, "no fixed stack")
	if gap := note.y - nm.y; gap != 16+14+9 {
		t.Fatalf("note baseline is %v px under the name baseline, want 39 (no hole for the hidden parts)", gap)
	}
}

func TestIdentityWithoutFocusPartHasNoRightColumn(t *testing.T) {
	without := renderIdentity(t, identityCfg([]string{"name", "tagline", "footer"}, ""), render.Dark)
	if strings.Contains(without, `x1="640"`) || strings.Contains(without, "FOCUS") {
		t.Fatal("focus column rule or label drawn although focus is not listed")
	}
	with := renderIdentity(t, identityCfg([]string{"name", "focus"}, ""), render.Dark)
	if !strings.Contains(with, `x1="640"`) {
		t.Fatal("focus column rule missing although focus is listed")
	}
	if _, ok := findText(texts(t, with), "WEB &amp; DESKTOP"); !ok {
		t.Fatal("focus entry missing although focus is listed")
	}
}

func TestIdentityCardLayoutStacksEverythingIn495(t *testing.T) {
	svg := renderIdentity(t, identityCfg(nil, config.LayoutCard), render.Dark)
	assertWellFormed(t, svg)
	w, h := viewBox(t, svg)
	if w != 495 {
		t.Fatalf("viewBox width = %d, want 495", w)
	}
	if strings.Contains(svg, `x1="640"`) {
		t.Fatal("the vertical focus rule belongs to the wide layout")
	}
	ts := texts(t, svg)
	for _, s := range ts {
		if s.anchor == "start" && s.x >= 495 || s.anchor == "end" && s.x > 495 {
			t.Fatalf("text %q is outside the card: x=%v", s.body, s.x)
		}
	}
	var nameBig svgText
	for _, s := range ts {
		if s.size > 30 {
			nameBig = s
		}
	}
	if nameBig.body != "wewesss" {
		t.Fatalf("big name not found: %+v", nameBig)
	}
	focus, ok := findText(ts, "WEB &amp; DESKTOP")
	if !ok || focus.y <= nameBig.y {
		t.Fatalf("focus entries must sit under the text, focus=%+v name=%+v", focus, nameBig)
	}
	last, _ := findText(ts, "github.com/")
	if last.y <= focus.y || last.y > float64(h) {
		t.Fatalf("footer must be the lowest text inside the card: %v vs %d", last.y, h)
	}
	_, wideH := viewBox(t, renderIdentity(t, identityCfg(nil, ""), render.Dark))
	if h <= wideH {
		t.Fatalf("stacked card (%d) must be taller than the wide one (%d)", h, wideH)
	}
}

func TestIdentityCardShrinksLongNamesToFit447(t *testing.T) {
	short := identityCfg([]string{"name"}, config.LayoutCard)
	long := identityCfg([]string{"name"}, config.LayoutCard)
	long.Profile.Name = "Alexandria Montgomery-Wellington"
	sizeOf := func(cfg config.Config) float64 {
		var best float64
		for _, s := range texts(t, renderIdentity(t, cfg, render.Dark)) {
			best = max(best, s.size)
		}
		return best
	}
	if got := sizeOf(short); got != 76 {
		t.Fatalf("a short name keeps size 76, got %v", got)
	}
	got := sizeOf(long)
	if got >= 76 {
		t.Fatalf("a %d-character name must shrink, got %v", len(long.Profile.Name), got)
	}
	spaces := float64(strings.Count(long.Profile.Name, " "))
	letters := float64(len(long.Profile.Name)) - spaces
	if width := (letters*0.56 + spaces*0.28) * got; width > 447 {
		t.Fatalf("estimated width %v exceeds 447", width)
	}
}

func TestIdentityPlacedInARowNeedsNoTaglineOrFocus(t *testing.T) {
	cfg := config.Default()
	cfg.Profile = config.Profile{Name: "solo", Location: "x"}
	cfg.Render.Rows = []config.Row{{Items: []string{"identity"}}}
	files, err := render.Cards(sample.Report(), cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	svg, ok := files["identity-dark.svg"]
	if !ok {
		t.Fatal("identity card missing although placed in a row")
	}
	if w, h := viewBox(t, svg); w != 1000 || h != 288 {
		t.Fatalf("viewBox = %dx%d, want the unchanged 1000x288", w, h)
	}
	if !strings.Contains(render.Snippet(cfg), "ghstats/identity-dark.svg") {
		t.Fatal("snippet omits the identity placed in a row")
	}

	cfg.Render.Rows = nil
	files, err = render.Cards(sample.Report(), cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["identity-dark.svg"]; ok {
		t.Fatal("no identity expected without tagline, focus, show or row")
	}
}

func TestIdentityShowAloneEnablesTheCard(t *testing.T) {
	cfg := config.Default()
	cfg.Profile = config.Profile{Name: "solo", Identity: config.IdentityOptions{Show: []string{"name"}}}
	files, err := render.Cards(sample.Report(), cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["identity-dark.svg"]; !ok {
		t.Fatal("identity.show must enable the identity card")
	}
}

func TestPerLinkStyleChangesOnlyThatBadge(t *testing.T) {
	cfg := sample.Profile()
	cfg.Profile.Links = []config.Link{
		{Kind: "youtube", Value: "@a"},
		{Kind: "website", Value: "https://example.com", Style: config.LinkStyleIcon},
	}
	files, err := render.Cards(sample.Report(), cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := viewBox(t, files["links/website-dark.svg"]); w != 40 || h != 40 {
		t.Fatalf("icon badge = %dx%d, want 40x40", w, h)
	}
	if w, _ := viewBox(t, files["links/youtube-dark.svg"]); w <= 40 {
		t.Fatalf("the other badge keeps the full style, width %d", w)
	}
}
