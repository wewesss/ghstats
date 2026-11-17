package render_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
)

func profileConfig() config.Config {
	c := config.Default()
	c.Render.Theme.Accent = config.Accent{Name: "cycle"}
	c.Profile = config.Profile{
		Name:     "wewesss",
		Tagline:  "SOFTWARE · GAMES · EXPERIMENTS",
		Location: "caen, fr",
		Note:     "no fixed stack — it depends on what I'm making",
		Focus: []config.Focus{
			{Name: "WEB & DESKTOP", Building: []string{"full-stack apps", "desktop software"}},
			{Name: "GAMES", Building: []string{"gameplay systems", "game engines"}},
			{Name: "AI & AUTOMATION", Building: []string{"coding agents", "automated workflows"}},
			{Name: "SYSTEMS", Building: []string{"runtimes & compilers", "native tooling"}},
		},
		Links: []config.Link{
			{Kind: "youtube", Value: "@wewesss14"},
			{Kind: "website", Value: "https://miralabs.tech"},
			{Kind: "github_sponsors", Value: "wewesss"},
		},
	}
	return c
}

func renderIdentity(t *testing.T, cfg config.Config, mode render.Mode) string {
	t.Helper()
	var buf bytes.Buffer
	if err := render.IdentitySVG(&buf, sampleReport(), cfg, mode); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func wideConfig() config.Render {
	c := fullConfig()
	c.Layout = config.LayoutWide
	return c
}

func TestGoldenIdentity(t *testing.T) {
	for _, mode := range []render.Mode{render.Dark, render.Light} {
		t.Run(string(mode), func(t *testing.T) {
			svg := renderIdentity(t, profileConfig(), mode)
			assertWellFormed(t, svg)
			golden(t, "identity-"+string(mode)+".svg", svg)
		})
	}
	t.Run("motion-off", func(t *testing.T) {
		cfg := profileConfig()
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
	cfg := profileConfig()
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

	cfg = profileConfig()
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
					cfg := profileConfig()
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
	if err := render.IdentitySVG(&buf, sampleReport(), profileConfig(), "sepia"); err == nil {
		t.Fatal("unknown mode must fail")
	}
	cfg := profileConfig()
	cfg.Profile.Focus = []config.Focus{}
	if err := render.IdentitySVG(&buf, sampleReport(), cfg, render.Dark); err == nil {
		t.Fatal("invalid profile must fail")
	}
}

func TestGoldenWideStats(t *testing.T) {
	for _, mode := range []render.Mode{render.Dark, render.Light} {
		t.Run(string(mode), func(t *testing.T) {
			svg := renderSVG(t, sampleReport(), wideConfig(), mode)
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
		allBlocks,
	} {
		for _, motion := range []bool{true, false} {
			cfg := wideConfig()
			cfg.Blocks = blocks
			cfg.Motion = motion
			assertWellFormed(t, renderSVG(t, sampleReport(), cfg, render.Dark))
		}
	}
}

func TestAccentSharedAcrossCards(t *testing.T) {
	stats := renderSVG(t, sampleReport(), fullConfig(), render.Dark)
	wide := renderSVG(t, sampleReport(), wideConfig(), render.Dark)
	ident := renderIdentity(t, profileConfig(), render.Dark)
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

func TestGoldenBadges(t *testing.T) {
	cfg := profileConfig()
	for _, l := range cfg.Profile.ResolvedLinks() {
		t.Run(l.Kind, func(t *testing.T) {
			var buf bytes.Buffer
			if err := render.LinkBadgeSVG(&buf, l, cfg.Render.Theme, render.Dark); err != nil {
				t.Fatal(err)
			}
			assertWellFormed(t, buf.String())
			if strings.Contains(buf.String(), "animation") {
				t.Fatal("badges are static")
			}
			golden(t, "badge-"+l.Kind+"-dark.svg", buf.String())
		})
	}
}

func TestEveryLinkKindHasABadge(t *testing.T) {
	for _, kind := range config.LinkKinds() {
		value := "handle"
		if kind == "website" || kind == "mastodon" || kind == "steam" {
			value = "https://example.com/@handle"
		}
		l := config.Link{Kind: kind, Value: value}
		u, label, err := l.Resolve()
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		for _, mode := range []render.Mode{render.Dark, render.Light} {
			var buf bytes.Buffer
			if err := render.LinkBadgeSVG(&buf, config.ResolvedLink{Kind: kind, URL: u, Label: label}, config.DefaultRender().Theme, mode); err != nil {
				t.Fatalf("%s: %v", kind, err)
			}
			assertWellFormed(t, buf.String())
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

func TestSnippet(t *testing.T) {
	cfg := profileConfig()
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
