package render_test

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
)

var tagsRe = regexp.MustCompile(`<[^>]*>`)

func visibleRunes(body string) int {
	return utf8.RuneCountInString(tagsRe.ReplaceAllString(body, ""))
}

func TestIdentityHeaderSlashSitsCloseToAMultiWordName(t *testing.T) {
	cases := map[string]config.Config{
		"legacy": identityCfg(nil, ""),
		"flow":   identityCfg([]string{"header", "name"}, ""),
		"card":   identityCfg([]string{"header", "name"}, config.LayoutCard),
	}
	for label, cfg := range cases {
		t.Run(label, func(t *testing.T) {
			cfg.Profile.Name = "Ada Lovelace"
			ts := texts(t, renderIdentity(t, cfg, render.Dark))
			var name svgText
			for _, s := range ts {
				if s.size == 14 && s.body == "Ada Lovelace" {
					name = s
				}
			}
			slash, ok := findText(ts, "/ profile")
			if name.body == "" || !ok {
				t.Fatalf("header name or slash missing: %+v %+v", name, slash)
			}
			end := name.x + 0.55*14*float64(utf8.RuneCountInString(name.body))
			if gap := slash.x - end; gap < 0 || gap > 12 {
				t.Fatalf("\"/ profile\" starts %.1f px after the estimated end of the name, want within 0..12", gap)
			}
		})
	}
}

func TestIdentityWithoutHeaderDrawsNoRuleAndHoistsLocation(t *testing.T) {
	heightOf := func(show ...string) int {
		_, h := viewBox(t, renderIdentity(t, identityCfg(show, ""), render.Dark))
		return h
	}
	svg := renderIdentity(t, identityCfg([]string{"name", "location"}, ""), render.Dark)
	if strings.Contains(svg, "<line") {
		t.Fatal("no rule line may be drawn when the header is hidden")
	}
	if strings.Contains(svg, "pulse") {
		t.Fatal("the header dot belongs to the header")
	}
	if got, want := heightOf("name", "location"), heightOf("name"); got != want {
		t.Fatalf("location without header added an empty band: height %d, want %d", got, want)
	}
	if withHeader := heightOf("header", "name", "location"); withHeader <= heightOf("name", "location") {
		t.Fatalf("the header row must still add its band: %d", withHeader)
	}

	ts := texts(t, svg)
	loc, ok := findText(ts, "caen, fr")
	name, _ := findText(ts, "wewesss")
	if !ok || loc.anchor != "end" || loc.x != 972 {
		t.Fatalf("location must sit at the right edge of the content: %+v", loc)
	}
	if loc.y >= name.y {
		t.Fatalf("location (y=%v) must sit at the top of the first line, above the name baseline %v", loc.y, name.y)
	}

	withTagline := texts(t, renderIdentity(t, identityCfg([]string{"tagline", "name", "location"}, ""), render.Dark))
	tag, _ := findText(withTagline, "SOFTWARE")
	loc, _ = findText(withTagline, "caen, fr")
	if loc.y != tag.y {
		t.Fatalf("location must share the first line baseline: %v vs %v", loc.y, tag.y)
	}
}

func TestIdentityCardWrapsLongTaglineAndNote(t *testing.T) {
	words := strings.Repeat("SOFTWARE / GAMES / EXPERIMENTS / ", 4)[:119] + "X"
	note := strings.Repeat("no fixed stack, whatever the day brings, ", 4)[:119] + "x"
	long := identityCfg([]string{"name", "tagline", "note", "footer"}, config.LayoutCard)
	long.Profile.Tagline, long.Profile.Note = words, note
	short := identityCfg([]string{"name", "tagline", "note", "footer"}, config.LayoutCard)
	svg := renderIdentity(t, long, render.Dark)
	assertWellFormed(t, svg)

	var tagLines, noteLines []svgText
	for _, s := range texts(t, svg) {
		switch s.size {
		case 11:
			if !strings.HasPrefix(s.body, "github.com/") {
				tagLines = append(tagLines, s)
			}
		case 12:
			noteLines = append(noteLines, s)
		}
	}
	check := func(label string, lines []svgText, advance float64, original string) {
		t.Helper()
		if len(lines) < 2 {
			t.Fatalf("%s: %d line(s), want several", label, len(lines))
		}
		var joined []string
		for _, l := range lines {
			if end := l.x + float64(utf8.RuneCountInString(l.body))*advance; end > 471 {
				t.Errorf("%s line %q ends at %.1f, past the 471 text edge", label, l.body, end)
			}
			joined = append(joined, l.body)
		}
		if got := strings.Join(joined, " "); got != original {
			t.Errorf("%s lost or changed words:\n got %q\nwant %q", label, got, original)
		}
	}
	check("tagline", tagLines, 0.6*11+2, words)
	check("note", noteLines, 0.6*12, note)

	_, longH := viewBox(t, svg)
	_, shortH := viewBox(t, renderIdentity(t, short, render.Dark))
	if longH <= shortH+2*15 {
		t.Fatalf("wrapped text must grow the card: %d vs %d", longH, shortH)
	}
}

func TestIdentityCardTruncatesWordsThatCannotFit(t *testing.T) {
	cfg := identityCfg([]string{"tagline", "building"}, config.LayoutCard)
	cfg.Profile.Tagline = strings.Repeat("W", 80)
	cfg.Profile.Focus = []config.Focus{{Name: "X", Building: []string{strings.Repeat("a", 60), "short"}}}
	ts := texts(t, renderIdentity(t, cfg, render.Dark))
	tag := ts[0]
	if !strings.HasSuffix(tag.body, "…") || tag.x+float64(utf8.RuneCountInString(tag.body))*(0.6*11+2) > 471 {
		t.Fatalf("a word wider than the text width must end with an ellipsis and fit: %q", tag.body)
	}
	for _, s := range ts {
		if s.size != 18 {
			continue
		}
		if end := s.x + float64(visibleRunes(s.body))*0.6*18; end > 471 {
			t.Fatalf("building line %q ends at %.1f, past 471", s.body, end)
		}
	}
	if !strings.Contains(renderIdentity(t, cfg, render.Dark), "…") {
		t.Fatal("expected a truncated building word")
	}
}
