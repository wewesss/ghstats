package render

import (
	"regexp"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
)

func TestSnippetLinkAnchorsHaveNoWhitespace(t *testing.T) {
	cfg, err := config.Parse(strings.NewReader(`version: 1
profile:
  tagline: T
  links:
    - youtube: "@a"
    - website: https://example.com
`))
	if err != nil {
		t.Fatal(err)
	}
	s := Snippet(cfg)
	anchors := regexp.MustCompile(`(?s)<a href=.*?</a>`).FindAllString(s, -1)
	if len(anchors) != 2 {
		t.Fatalf("want 2 anchors, got %d", len(anchors))
	}
	for _, a := range anchors {
		if strings.ContainsAny(a, "\n\t") || strings.Contains(a, "> <") {
			t.Errorf("anchor contains whitespace that GitHub renders as an underlined link: %q", a)
		}
	}
}

func snippetAnchor(kind, url string) string {
	return `<a href="` + url + `"><picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/links/` + kind + `-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/links/` + kind + `-light.svg"><img alt="` + kind + `" src="ghstats/links/` + kind + `-dark.svg"></picture></a>`
}

func linkRowConfig(t *testing.T, rows string) config.Config {
	t.Helper()
	cfg, err := config.Parse(strings.NewReader(`version: 1
render:
  rows:
` + rows + `profile:
  links:
    - youtube: "@a"
    - website: https://example.com
    - github_sponsors: a
`))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestSnippetPlacesLinkBadgesOneByOne(t *testing.T) {
	cfg := linkRowConfig(t, "    - align: left\n      items: [links/website, links/youtube]\n    - align: right\n      items: [links]\n")
	var tail []string
	for _, line := range strings.Split(Snippet(cfg), "\n") {
		if line != "" {
			tail = append(tail, line)
		}
	}
	idx := -1
	for i, line := range tail {
		if line == `<p align="left">` {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("no left row in %q", tail)
	}
	want := []string{
		`<p align="left">`,
		snippetAnchor("website", "https://example.com"),
		snippetAnchor("youtube", "https://www.youtube.com/@a"),
		`</p>`,
		`<p align="right">`,
		snippetAnchor("github_sponsors", "https://github.com/sponsors/a"),
		`</p>`,
	}
	got := tail[idx:]
	if len(got) < len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d:\n got %s\nwant %s", i, got[i], want[i])
		}
	}
}

func TestSnippetLeavesUnplacedLinksAtTheEnd(t *testing.T) {
	cfg := linkRowConfig(t, "    - items: [links/youtube]\n")
	s := Snippet(cfg)
	if n := strings.Count(s, `<a href="https://www.youtube.com/@a">`); n != 1 {
		t.Fatalf("youtube anchor count = %d, want 1", n)
	}
	blocks := strings.Split(s, "\n\n")
	last := blocks[len(blocks)-1]
	want := "<p align=\"center\">\n" +
		snippetAnchor("website", "https://example.com") + "\n" +
		snippetAnchor("github_sponsors", "https://github.com/sponsors/a") + "\n</p>\n"
	if last != want {
		t.Fatalf("last block:\n%s\nwant:\n%s", last, want)
	}
}

func TestSnippetSkipsLinkRowsWithoutLinks(t *testing.T) {
	cfg := linkRowConfig(t, "    - items: [links/twitch]\n")
	if strings.Contains(Snippet(cfg), `<p align="center">`+"\n</p>") {
		t.Fatal("an empty link row must not leave an empty paragraph")
	}
	if strings.Contains(Snippet(cfg), "twitch") {
		t.Fatal("a link that does not exist must be ignored")
	}
}

func TestSnippetIdentityCardLayoutIs495(t *testing.T) {
	cfg := config.Default()
	cfg.Profile = config.Profile{Name: "a", Identity: config.IdentityOptions{Layout: config.LayoutCard, Show: []string{"name"}}}
	cfg.Render.Rows = []config.Row{{Items: []string{"identity", "stats"}}}
	lines := strings.Split(Snippet(cfg), "\n")
	if len(lines) < 3 || !strings.Contains(lines[1], `src="ghstats/identity-dark.svg" width="49%"`) || !strings.Contains(lines[1], `src="ghstats/stats-dark.svg" width="49%"`) {
		t.Fatalf("identity (495) next to a card (495) must share the row 49%% / 49%%:\n%s", Snippet(cfg))
	}

	cfg.Render.Rows = []config.Row{{Items: []string{"identity"}}}
	if s := Snippet(cfg); !strings.Contains(s, `src="ghstats/identity-dark.svg" width="495"`) {
		t.Fatalf("a card-layout identity alone must be 495 wide:\n%s", s)
	}
	cfg.Profile.Identity.Layout = ""
	if s := Snippet(cfg); !strings.Contains(s, `src="ghstats/identity-dark.svg" width="98%"`) {
		t.Fatalf("a wide identity alone must be 98%% wide:\n%s", s)
	}
}
