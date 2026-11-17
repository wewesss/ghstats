package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
)

const fullProfile = `version: 1
render:
  layout: wide
  base_path: assets/ghstats/
profile:
  name: wewesss
  tagline: SOFTWARE · GAMES · EXPERIMENTS
  location: caen, fr
  note: no fixed stack
  focus:
    - name: WEB & DESKTOP
      building: [full-stack apps, desktop software]
    - name: GAMES
      building: [gameplay systems]
  links:
    - youtube: "@wewesss14"
    - website: https://miralabs.tech
    - discord: {url: "https://discord.com/invite/xyz", label: "chat"}
    - twitch: ""
    - github_sponsors: wewesss
`

func TestProfileFullValidExample(t *testing.T) {
	c, err := config.Parse(strings.NewReader(fullProfile))
	if err != nil {
		t.Fatal(err)
	}
	if c.Render.Layout != "wide" || c.Render.BasePath != "assets/ghstats/" {
		t.Fatalf("render = %+v", c.Render)
	}
	p := c.Profile
	if p.Name != "wewesss" || p.Location != "caen, fr" || len(p.Focus) != 2 || p.Focus[0].Building[1] != "desktop software" || len(p.Links) != 5 {
		t.Fatalf("profile = %+v", p)
	}
	if !p.HasIdentity() {
		t.Fatal("identity expected")
	}
	got := p.ResolvedLinks()
	want := []config.ResolvedLink{
		{Kind: "youtube", URL: "https://www.youtube.com/@wewesss14", Label: "@wewesss14"},
		{Kind: "website", URL: "https://miralabs.tech", Label: "miralabs.tech"},
		{Kind: "discord", URL: "https://discord.com/invite/xyz", Label: "chat"},
		{Kind: "github_sponsors", URL: "https://github.com/sponsors/wewesss", Label: "sponsor"},
	}
	if len(got) != len(want) {
		t.Fatalf("links = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("link %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestProfileDefaults(t *testing.T) {
	d := config.Default()
	if d.Render.Layout != "card" || d.Render.BasePath != "ghstats/" {
		t.Fatalf("render = %+v", d.Render)
	}
	if d.Profile.HasIdentity() || len(d.Profile.ResolvedLinks()) != 0 {
		t.Fatal("empty profile must produce nothing")
	}
	if !(config.Profile{Tagline: "x"}).HasIdentity() {
		t.Fatal("a tagline alone gives an identity")
	}
}

func TestLinkURLsAndLabels(t *testing.T) {
	tests := []struct{ kind, value, url, label string }{
		{"youtube", "@h", "https://www.youtube.com/@h", "@h"},
		{"youtube", "h", "https://www.youtube.com/@h", "@h"},
		{"youtube", "https://www.youtube.com/@chan", "https://www.youtube.com/@chan", "@chan"},
		{"twitch", "h", "https://www.twitch.tv/h", "h"},
		{"kick", "h", "https://kick.com/h", "h"},
		{"discord", "abc", "https://discord.gg/abc", "discord"},
		{"tiktok", "@h", "https://www.tiktok.com/@h", "@h"},
		{"instagram", "h", "https://www.instagram.com/h", "@h"},
		{"x", "@h", "https://x.com/h", "@h"},
		{"bluesky", "h.bsky.social", "https://bsky.app/profile/h.bsky.social", "h.bsky.social"},
		{"threads", "h", "https://www.threads.net/@h", "@h"},
		{"mastodon", "https://mastodon.social/@h", "https://mastodon.social/@h", "mastodon.social/@h"},
		{"reddit", "h", "https://www.reddit.com/user/h", "u/h"},
		{"linkedin", "h", "https://www.linkedin.com/in/h", "h"},
		{"steam", "https://steamcommunity.com/id/h", "https://steamcommunity.com/id/h", "steam"},
		{"itchio", "h", "https://h.itch.io", "h"},
		{"itchio", "https://h.itch.io", "https://h.itch.io", "h"},
		{"website", "https://miralabs.tech/path", "https://miralabs.tech/path", "miralabs.tech"},
		{"github_sponsors", "h", "https://github.com/sponsors/h", "sponsor"},
		{"open_collective", "h", "https://opencollective.com/h", "h"},
		{"buy_me_a_coffee", "h", "https://buymeacoffee.com/h", "coffee"},
		{"ko_fi", "h", "https://ko-fi.com/h", "ko-fi"},
		{"patreon", "h", "https://www.patreon.com/h", "h"},
	}
	for _, tt := range tests {
		t.Run(tt.kind+"/"+tt.value, func(t *testing.T) {
			u, label, err := config.Link{Kind: tt.kind, Value: tt.value}.Resolve()
			if err != nil {
				t.Fatal(err)
			}
			if u != tt.url || label != tt.label {
				t.Fatalf("got %q %q, want %q %q", u, label, tt.url, tt.label)
			}
		})
	}
	if got := len(config.LinkKinds()); got != 21 {
		t.Fatalf("kinds = %d, want 21", got)
	}
}

func TestProfileRejects(t *testing.T) {
	tests := []struct {
		name, yaml, want string
	}{
		{"unknown kind", "links:\n    - myspace: x", `unknown kind "myspace"`},
		{"duplicate kind", "links:\n    - youtube: a\n    - youtube: b", `kind "youtube" is listed more than once`},
		{"duplicate with empty", "links:\n    - youtube: a\n    - youtube: \"\"", "more than once"},
		{"url-only kind with a handle", "links:\n    - website: miralabs", "needs a full https:// URL"},
		{"mastodon with a handle", "links:\n    - mastodon: \"@me\"", "needs a full https:// URL"},
		{"steam with a handle", "links:\n    - steam: me", "needs a full https:// URL"},
		{"http URL", "links:\n    - website: http://miralabs.tech", "https://"},
		{"ftp URL", "links:\n    - youtube: ftp://x.y/z", "https://"},
		{"URL without host", "links:\n    - website: \"https://\"", "no host"},
		{"handle with spaces", "links:\n    - twitch: a b", "invalid handle"},
		{"link not a map", "links:\n    - youtube", "exactly one key"},
		{"link with two keys", "links:\n    - youtube: a\n      twitch: b", "exactly one key"},
		{"map without url", "links:\n    - discord: {label: x}", "needs a url"},
		{"custom as a scalar", "links:\n    - custom: https://example.com", "needs the map form"},
		{"custom without a label", "links:\n    - custom: {url: \"https://example.com\"}", "needs a label"},
		{"custom with a blank label", "links:\n    - custom: {url: \"https://example.com\", label: \" \"}", "needs a label"},
		{"custom without a url", "links:\n    - custom: {label: blog}", "needs a url"},
		{"custom http URL", "links:\n    - custom: {url: \"http://example.com\", label: blog}", "https://"},
		{"custom icon unknown name", "links:\n    - custom: {url: \"https://example.com\", label: blog, icon: nope}", "neither a link kind"},
		{"custom icon is custom", "links:\n    - custom: {url: \"https://example.com\", label: blog, icon: custom}", "neither a link kind"},
		{"custom icon path with a tag", "links:\n    - custom: {url: \"https://example.com\", label: blog, icon: \"M0 0<script>\"}", "must not contain"},
		{"custom icon path with a quote", "links:\n    - custom: {url: \"https://example.com\", label: blog, icon: 'M0 0\"x'}", "must not contain"},
		{"custom icon path with an ampersand", "links:\n    - custom: {url: \"https://example.com\", label: blog, icon: \"M0 0&amp;\"}", "must not contain"},
		{"custom icon not a string", "links:\n    - custom: {url: \"https://example.com\", label: blog, icon: [a]}", "must be a string"},
		{"icon on another kind", "links:\n    - youtube: {url: \"https://www.youtube.com/@a\", icon: discord}", "only valid for"},
		{"custom unknown key", "links:\n    - custom: {url: \"https://example.com\", label: blog, name: x}", "url, label and icon"},
		{"unknown link style", "link_style: round", "profile.link_style"},
		{"link style lists the values", "link_style: round", "full, icon, text"},
		{"link style is case sensitive", "link_style: Icon", "profile.link_style"},
		{"map unknown key", "links:\n    - discord: {url: \"https://x.y\", name: x}", `unknown key "name"`},
		{"map non-https", "links:\n    - discord: {url: \"http://x.y\"}", "https://"},
		{"link value is a list", "links:\n    - discord: [a]", "must be a handle"},
		{"no focus entries", "focus: []", "1 to 6"},
		{"seven focus entries", "focus:\n" + strings.Repeat("    - {name: A, building: [x]}\n", 7), "1 to 6"},
		{"empty focus name", "focus:\n    - {name: \"\", building: [x]}", "name must not be empty"},
		{"no building words", "focus:\n    - {name: A, building: []}", "1 to 4"},
		{"missing building", "focus:\n    - {name: A}", "1 to 4"},
		{"five building words", "focus:\n    - {name: A, building: [a, b, c, d, e]}", "1 to 4"},
		{"blank building word", "focus:\n    - {name: A, building: [\" \"]}", "must not be empty"},
		{"unknown profile key", "colour: red", "colour"},
		{"unknown focus key", "focus:\n    - {name: A, building: [x], extra: 1}", "extra"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Parse(strings.NewReader("version: 1\nprofile:\n  " + tt.yaml + "\n"))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestCustomLinkAndLinkStyle(t *testing.T) {
	const doc = `version: 1
profile:
  link_style: icon
  links:
    - custom: {url: "https://example.com/blog", label: "my blog", icon: youtube}
`
	c, err := config.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	got := c.Profile.ResolvedLinks()
	want := config.ResolvedLink{Kind: "custom", URL: "https://example.com/blog", Label: "my blog", Icon: "youtube", Style: "icon"}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("links = %+v, want %+v", got, want)
	}
	for _, icon := range []string{"", "mastodon", "M12 2l3 7h7z", "m1 1h2"} {
		l := config.Link{Kind: "custom", Value: "https://example.com", Label: "x", Icon: icon}
		if _, _, err := l.Resolve(); err != nil {
			t.Fatalf("icon %q: %v", icon, err)
		}
	}
	for _, style := range []string{"", "full", "icon", "text"} {
		c := config.Default()
		c.Profile.LinkStyle = style
		if err := c.Validate(); err != nil {
			t.Fatalf("style %q: %v", style, err)
		}
	}
	if !config.LinkKindIsMapOnly("custom") || config.LinkKindIsMapOnly("website") {
		t.Fatal("only custom is map-only")
	}
}

func TestRenderLayoutAndBasePathRejects(t *testing.T) {
	tests := []struct{ name, yaml, want string }{
		{"unknown layout", "layout: huge", "render.layout"},
		{"empty layout", "layout: \"\"", "render.layout"},
		{"no trailing slash", "base_path: ghstats", "render.base_path"},
		{"empty base path", "base_path: \"\"", "render.base_path"},
		{"parent segment", "base_path: ../ghstats/", "render.base_path"},
		{"quote", "base_path: 'a\"b/'", "render.base_path"},
		{"space", "base_path: 'a b/'", "render.base_path"},
		{"backslash", "base_path: 'a\\b/'", "render.base_path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Parse(strings.NewReader("version: 1\nrender:\n  " + tt.yaml + "\n"))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.want)
			}
		})
	}
	for _, ok := range []string{"ghstats/", "/ghstats/", "https://cdn.example.com/x/", "./ghstats/", "a/b/c/"} {
		if _, err := config.Parse(strings.NewReader("version: 1\nrender:\n  base_path: " + ok + "\n")); err != nil {
			t.Fatalf("%q: %v", ok, err)
		}
	}
}

func TestProfileValidateJoinsErrors(t *testing.T) {
	c := config.Default()
	c.Profile = config.Profile{
		Focus: []config.Focus{{Name: "", Building: nil}},
		Links: []config.Link{{Kind: "nope", Value: "x"}, {Kind: "website", Value: "x"}},
	}
	c.Render.Layout = "huge"
	err := c.Validate()
	for _, want := range []string{"render.layout", "name must not be empty", "1 to 4", `unknown kind "nope"`, "needs a full https:// URL"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("err = %v, want it to mention %q", err, want)
		}
	}
}

func TestExampleProfileSectionIsValidWhenUncommented(t *testing.T) {
	raw, err := os.ReadFile("example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	in := false
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "# profile:" {
			in = true
		}
		if in && strings.HasPrefix(line, "# ") {
			line = strings.TrimPrefix(line, "# ")
		}
		out = append(out, line)
	}
	if !in {
		t.Fatal("commented profile section missing")
	}
	c, err := config.Parse(strings.NewReader(strings.Join(out, "\n")))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Profile.Focus) != 2 || len(c.Profile.ResolvedLinks()) != 5 || c.Render.Layout != "card" || c.Render.BasePath != "ghstats/" {
		t.Fatalf("profile = %+v render = %+v", c.Profile, c.Render)
	}
}
