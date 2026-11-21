package render_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/sample"
)

func renderIdentity(t *testing.T, cfg config.Config, mode render.Mode) string {
	t.Helper()
	var buf bytes.Buffer
	if err := render.IdentitySVG(&buf, sample.Report(), cfg, mode); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestGoldenIdentity(t *testing.T) {
	for _, mode := range []render.Mode{render.Dark, render.Light} {
		t.Run(string(mode), func(t *testing.T) {
			svg := renderIdentity(t, sample.Profile(), mode)
			assertWellFormed(t, svg)
			golden(t, "identity-"+string(mode)+".svg", svg)
		})
	}
	t.Run("motion-off", func(t *testing.T) {
		cfg := sample.Profile()
		cfg.Render.Motion = false
		svg := renderIdentity(t, cfg, render.Dark)
		assertWellFormed(t, svg)
		if strings.Contains(svg, "@keyframes") || strings.Contains(svg, "animation:") {
			t.Fatal("animation present with motion off")
		}
		golden(t, "identity-dark-nomotion.svg", svg)
	})
}

func TestIdentityShape(t *testing.T) {
	cfg := sample.Profile()
	svg := renderIdentity(t, cfg, render.Dark)
	if !strings.Contains(svg, `viewBox="0 0 1000 300"`) {
		t.Fatal("4 focus entries must give a 300 px card")
	}
	if got := strings.Count(svg, `class="word`); got != 8 {
		t.Fatalf("words = %d, want 8", got)
	}
	for _, want := range []string{"animation-delay:-18s", "animation-delay:-12s", "animation-delay:-6s", "github.com/wewesss", "caen, fr", "WEB &amp; DESKTOP"} {
		if !strings.Contains(svg, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(svg, "links below") {
		t.Fatal("footer right must be empty")
	}

	cfg.Profile.Focus = append(cfg.Profile.Focus, config.Focus{Name: "FIFTH", Building: []string{"a"}}, config.Focus{Name: "SIXTH", Building: []string{"b", "c", "d", "e"}})
	svg = renderIdentity(t, cfg, render.Light)
	assertWellFormed(t, svg)
	if !strings.Contains(svg, `viewBox="0 0 1000 368"`) {
		t.Fatal("6 focus entries must grow the card to 368 px")
	}

	cfg = sample.Profile()
	cfg.Profile.Location = ""
	cfg.Profile.Note = ""
	cfg.Profile.Name = ""
	svg = renderIdentity(t, cfg, render.Dark)
	assertWellFormed(t, svg)
	if strings.Contains(svg, "caen") || strings.Contains(svg, "no fixed stack") {
		t.Fatal("empty location and note must be omitted")
	}

	cfg.Profile.Focus = nil
	svg = renderIdentity(t, cfg, render.Dark)
	assertWellFormed(t, svg)
	if strings.Contains(svg, "FOCUS") || strings.Contains(svg, "building") {
		t.Fatal("no focus must drop the focus column")
	}
}

func TestIdentityEveryPresetAndAccent(t *testing.T) {
	for _, preset := range []string{"noir", "carbon", "graphite", "midnight", "slate", "dune", "transparent"} {
		for _, acc := range []config.Accent{{Name: "ice"}, {Name: "cycle"}, {Dark: "#112233", Light: "#445566"}} {
			for _, mode := range []render.Mode{render.Dark, render.Light} {
				for _, motion := range []bool{true, false} {
					cfg := sample.Profile()
					cfg.Render.Theme.Preset = preset
					cfg.Render.Theme.Accent = acc
					cfg.Render.Motion = motion
					assertWellFormed(t, renderIdentity(t, cfg, mode))
				}
			}
		}
	}
}

func TestIdentityRejectsBadInput(t *testing.T) {
	var buf bytes.Buffer
	if err := render.IdentitySVG(&buf, sample.Report(), sample.Profile(), "sepia"); err == nil {
		t.Fatal("unknown mode must fail")
	}
	cfg := sample.Profile()
	cfg.Profile.Focus = []config.Focus{}
	if err := render.IdentitySVG(&buf, sample.Report(), cfg, render.Dark); err == nil {
		t.Fatal("invalid profile must fail")
	}
}

func TestGoldenWideStats(t *testing.T) {
	for _, mode := range []render.Mode{render.Dark, render.Light} {
		t.Run(string(mode), func(t *testing.T) {
			svg := renderSVG(t, sample.Report(), sample.WideRender(), mode)
			assertWellFormed(t, svg)
			if !strings.Contains(svg, `viewBox="0 0 1000 `) || !strings.Contains(svg, `translate(505 0)`) {
				t.Fatal("wide layout shape missing")
			}
			golden(t, "stats-wide-"+string(mode)+".svg", svg)
		})
	}
}

func TestWideLayoutVariants(t *testing.T) {
	for _, blocks := range [][]config.Block{
		{{Name: "languages", Variants: []string{"bars"}}},
		{{Name: "stats", Variants: []string{"totals"}}, {Name: "history", Variants: []string{"views"}}},
		sample.AllBlocks(),
	} {
		for _, motion := range []bool{true, false} {
			cfg := sample.WideRender()
			cfg.Blocks = blocks
			cfg.Motion = motion
			assertWellFormed(t, renderSVG(t, sample.Report(), cfg, render.Dark))
		}
	}
}

func TestAccentSharedAcrossCards(t *testing.T) {
	stats := renderSVG(t, sample.Report(), sample.FullRender(), render.Dark)
	wide := renderSVG(t, sample.Report(), sample.WideRender(), render.Dark)
	ident := renderIdentity(t, sample.Profile(), render.Dark)
	css := func(svg string) string {
		i := strings.Index(svg, "@property")
		j := strings.Index(svg, "@keyframes gsaccent")
		if i < 0 || j < 0 {
			return ""
		}
		k := strings.Index(svg[j:], "}}")
		return svg[i : j+k+2]
	}
	want := css(stats)
	if want == "" || css(wide) != want || css(ident) != want || !strings.Contains(want, "40s") {
		t.Fatal("accent cycle CSS differs between cards")
	}
}

func TestEveryLinkKindHasABadge(t *testing.T) {
	for _, kind := range config.LinkKinds() {
		value := "handle"
		if kind == "website" || kind == "mastodon" || kind == "steam" || kind == config.KindCustom {
			value = "https://example.com/@handle"
		}
		l := config.Link{Kind: kind, Value: value}
		if kind == config.KindCustom {
			l.Label = "blog"
		}
		u, label, err := l.Resolve()
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		for _, style := range []string{"", config.LinkStyleFull, config.LinkStyleIcon, config.LinkStyleText} {
			for _, mode := range []render.Mode{render.Dark, render.Light} {
				var buf bytes.Buffer
				if err := render.LinkBadgeSVG(&buf, config.ResolvedLink{Kind: kind, URL: u, Label: label, Style: style}, config.DefaultRender().Theme, mode); err != nil {
					t.Fatalf("%s/%s: %v", kind, style, err)
				}
				assertWellFormed(t, buf.String())
			}
		}
	}
}

func badgeFor(t *testing.T, l config.Link, style string) string {
	t.Helper()
	p := config.Profile{Links: []config.Link{l}, LinkStyle: style}
	links := p.ResolvedLinks()
	if len(links) != 1 {
		t.Fatalf("link %+v did not resolve", l)
	}
	var buf bytes.Buffer
	if err := render.LinkBadgeSVG(&buf, links[0], config.DefaultRender().Theme, render.Dark); err != nil {
		t.Fatal(err)
	}
	assertWellFormed(t, buf.String())
	return buf.String()
}

func TestGoldenBadgeStyles(t *testing.T) {
	links := []config.Link{
		{Kind: "youtube", Value: "@wewesss14"},
		{Kind: "website", Value: "https://miralabs.tech"},
	}
	for _, l := range links {
		for _, style := range []string{config.LinkStyleIcon, config.LinkStyleText} {
			t.Run(l.Kind+"/"+style, func(t *testing.T) {
				golden(t, "badge-"+l.Kind+"-"+style+"-dark.svg", badgeFor(t, l, style))
			})
		}
	}
}

func TestBadgeStyleShapes(t *testing.T) {
	l := config.Link{Kind: "youtube", Value: "@wewesss14"}
	full := badgeFor(t, l, "")
	if explicit := badgeFor(t, l, config.LinkStyleFull); explicit != full {
		t.Fatal("an explicit full style must equal the default")
	}
	icon := badgeFor(t, l, config.LinkStyleIcon)
	for _, want := range []string{`viewBox="0 0 40 40"`, `width="40" height="40"`, `aria-label="@wewesss14"`, `<title>@wewesss14</title>`, "<path"} {
		if !strings.Contains(icon, want) {
			t.Fatalf("icon style misses %q:\n%s", want, icon)
		}
	}
	if strings.Contains(icon, "<text") {
		t.Fatal("icon style must not draw the label")
	}
	text := badgeFor(t, l, config.LinkStyleText)
	if strings.Contains(text, "<path") || strings.Contains(text, "<g ") || strings.Contains(text, "<line") {
		t.Fatalf("text style must not draw an icon:\n%s", text)
	}
	if !strings.Contains(text, "@wewesss14") || !strings.Contains(text, `height="40"`) {
		t.Fatalf("text style misses the label:\n%s", text)
	}
	var buf bytes.Buffer
	err := render.LinkBadgeSVG(&buf, config.ResolvedLink{Kind: "youtube", Label: "x", Style: "round"}, config.DefaultRender().Theme, render.Dark)
	if err == nil || !strings.Contains(err.Error(), "round") {
		t.Fatalf("unknown style must fail, got %v", err)
	}
}

func TestGoldenCustomBadges(t *testing.T) {
	for name, l := range sample.CustomLinks() {
		t.Run(name, func(t *testing.T) {
			svg := badgeFor(t, l, "")
			if strings.Contains(svg, "animation") {
				t.Fatal("badges are static")
			}
			golden(t, "badge-custom-"+name+"-dark.svg", svg)
		})
	}
	named, path, none := badgeFor(t, sample.CustomLinks()["named"], ""), badgeFor(t, sample.CustomLinks()["path"], ""), badgeFor(t, sample.CustomLinks()["none"], "")
	if !strings.Contains(path, `d="M12 2l3 7h7l-5.5 4.5L18.5 21 12 16.8 5.5 21l2-7.5L2 9h7z"`) {
		t.Fatal("the custom path must be drawn")
	}
	if !strings.Contains(none, "<circle") || strings.Contains(named, "<circle") || strings.Contains(path, "<circle") {
		t.Fatal("only the custom link without an icon draws the globe")
	}
	youtube := badgeFor(t, config.Link{Kind: "youtube", Value: "@a"}, "")
	_, rest, _ := strings.Cut(youtube, `<path d="`)
	yt, _, _ := strings.Cut(rest, `" fill=`)
	if yt == "" || !strings.Contains(named, `<path d="`+yt+`"`) {
		t.Fatal("a named icon must reuse the embedded icon of that kind")
	}
}

func TestCustomBadgeRejectsBadIcon(t *testing.T) {
	th := config.DefaultRender().Theme
	for _, icon := range []string{"nope", "M0 0\"><script>", "custom"} {
		var buf bytes.Buffer
		err := render.LinkBadgeSVG(&buf, config.ResolvedLink{Kind: "custom", URL: "https://example.com", Label: "x", Icon: icon}, th, render.Dark)
		if err == nil {
			t.Fatalf("icon %q must fail", icon)
		}
	}
}

func TestBadgeThemeAndEscaping(t *testing.T) {
	th := config.DefaultRender().Theme
	th.Overrides = map[string]string{"fg": "#123456", "bg": "none"}
	var buf bytes.Buffer
	err := render.LinkBadgeSVG(&buf, config.ResolvedLink{Kind: "x", URL: "https://x.com/a", Label: "a<b>&c"}, th, render.Light)
	if err != nil {
		t.Fatal(err)
	}
	svg := buf.String()
	assertWellFormed(t, svg)
	if !strings.Contains(svg, `fill="#123456"`) || !strings.Contains(svg, `fill="none"`) || !strings.Contains(svg, "a&lt;b&gt;&amp;c") {
		t.Fatalf("theme or escaping not applied: %s", svg)
	}
	if err := render.LinkBadgeSVG(&buf, config.ResolvedLink{Kind: "nope", Label: "x"}, th, render.Dark); err == nil {
		t.Fatal("unknown kind must fail")
	}
	if err := render.LinkBadgeSVG(&buf, config.ResolvedLink{Kind: "x", Label: "x"}, th, "sepia"); err == nil {
		t.Fatal("unknown mode must fail")
	}
}

func TestSnippetIsTheSameForEveryLinkStyle(t *testing.T) {
	cfg := sample.Profile()
	cfg.Profile.Links = append(sample.Profile().Profile.Links, sample.CustomLinks()["path"])
	want := render.Snippet(cfg)
	if !strings.Contains(want, `srcset="ghstats/links/custom-light.svg"`) {
		t.Fatalf("custom link missing:\n%s", want)
	}
	for _, style := range []string{config.LinkStyleIcon, config.LinkStyleText} {
		cfg.Profile.LinkStyle = style
		if got := render.Snippet(cfg); got != want {
			t.Fatalf("snippet differs for style %q", style)
		}
	}
}

func TestSnippet(t *testing.T) {
	cfg := sample.Profile()
	cfg.Profile.Links = []config.Link{
		{Kind: "website", Value: "https://miralabs.tech"},
		{Kind: "twitch", Value: ""},
		{Kind: "youtube", Value: "@wewesss14"},
		{Kind: "discord", Value: "https://discord.com/invite/xyz", Label: "chat"},
	}
	cfg.Render.BasePath = "assets/ghstats/"
	s := render.Snippet(cfg)
	order := []string{
		`srcset="assets/ghstats/identity-dark.svg"`,
		`src="assets/ghstats/identity-dark.svg" width="100%"`,
		`src="assets/ghstats/stats-dark.svg" width="495"`,
		`<p align="center">`,
		`<a href="https://miralabs.tech">`,
		`srcset="assets/ghstats/links/website-light.svg"`,
		`<a href="https://www.youtube.com/@wewesss14">`,
		`<img alt="youtube" src="assets/ghstats/links/youtube-dark.svg">`,
		`<a href="https://discord.com/invite/xyz">`,
		`</p>`,
	}
	pos := 0
	for _, want := range order {
		i := strings.Index(s[pos:], want)
		if i < 0 {
			t.Fatalf("missing or out of order %q in\n%s", want, s)
		}
		pos += i + len(want)
	}
	if strings.Contains(s, "twitch") {
		t.Fatal("empty link must be absent")
	}

	cfg.Render.Layout = config.LayoutWide
	if s := render.Snippet(cfg); !strings.Contains(s, `src="assets/ghstats/stats-dark.svg" width="100%"`) {
		t.Fatal("wide stats card must be 100% wide")
	}

	cfg.Profile = config.Profile{}
	s = render.Snippet(cfg)
	if strings.Contains(s, "identity") || strings.Contains(s, "<p ") || !strings.Contains(s, "assets/ghstats/stats-light.svg") {
		t.Fatalf("empty profile snippet = %s", s)
	}
}
