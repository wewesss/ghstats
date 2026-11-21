package config_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
)

func parseProfile(t *testing.T, yaml string) (config.Config, error) {
	t.Helper()
	return config.Parse(strings.NewReader("version: 1\n" + yaml))
}

func TestIdentityOptionsParseAndReject(t *testing.T) {
	c, err := parseProfile(t, "profile:\n  name: a\n  identity:\n    layout: card\n    show: [name, location, footer]\n")
	if err != nil {
		t.Fatal(err)
	}
	if c.Profile.Identity.Layout != config.LayoutCard || !reflect.DeepEqual(c.Profile.Identity.Show, []string{"name", "location", "footer"}) {
		t.Fatalf("identity = %+v", c.Profile.Identity)
	}
	for name, tt := range map[string]struct{ yaml, want string }{
		"unknown part":   {"    show: [name, colour]\n", `profile.identity.show[1]: unknown part "colour"`},
		"duplicate part": {"    show: [name, note, name]\n", `profile.identity.show[2]: part "name" is listed more than once`},
		"bad layout":     {"    layout: tall\n", `profile.identity.layout: must be "wide" or "card", got "tall"`},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseProfile(t, "profile:\n  name: a\n  identity:\n"+tt.yaml)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestIdentityShows(t *testing.T) {
	if !(config.IdentityOptions{}).Shows("footer") {
		t.Fatal("an empty show list must show every part")
	}
	o := config.IdentityOptions{Show: []string{"name"}}
	if !o.Shows("name") || o.Shows("footer") {
		t.Fatal("a show list must keep only the listed parts")
	}
}

func TestConfigHasIdentity(t *testing.T) {
	rows := []config.Row{{Items: []string{"identity"}}}
	tests := []struct {
		name string
		cfg  func(c *config.Config)
		want bool
	}{
		{"default config", func(c *config.Config) {}, false},
		{"name and location only", func(c *config.Config) { c.Profile = config.Profile{Name: "a", Location: "b"} }, false},
		{"tagline", func(c *config.Config) { c.Profile.Tagline = "t" }, true},
		{"focus", func(c *config.Config) {
			c.Profile.Focus = []config.Focus{{Name: "a", Building: []string{"b"}}}
		}, true},
		{"show list", func(c *config.Config) { c.Profile.Identity.Show = []string{"name"} }, true},
		{"placed in a row", func(c *config.Config) { c.Render.Rows = rows }, true},
		{"layout alone", func(c *config.Config) { c.Profile.Identity.Layout = config.LayoutCard }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := config.Default()
			tt.cfg(&c)
			if got := c.HasIdentity(); got != tt.want {
				t.Fatalf("HasIdentity = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPerLinkStyle(t *testing.T) {
	c, err := parseProfile(t, `profile:
  link_style: text
  links:
    - youtube: "@a"
    - website: {url: "https://example.com", style: icon}
    - discord: {url: "https://discord.gg/x", label: chat, style: full}
`)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, l := range c.Profile.ResolvedLinks() {
		got = append(got, l.Kind+"="+l.Style)
	}
	if want := []string{"youtube=text", "website=icon", "discord=full"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("styles = %v, want %v", got, want)
	}

	var buf bytes.Buffer
	if err := config.Encode(&buf, c); err != nil {
		t.Fatal(err)
	}
	back, err := config.Parse(&buf)
	if err != nil {
		t.Fatalf("re-parse: %v\n%s", err, buf.String())
	}
	if !reflect.DeepEqual(back.Profile.Links, c.Profile.Links) {
		t.Fatalf("links after round trip = %+v, want %+v", back.Profile.Links, c.Profile.Links)
	}

	_, err = parseProfile(t, "profile:\n  links:\n    - website: {url: \"https://example.com\", style: big}\n")
	if err == nil || !strings.Contains(err.Error(), `profile.links[0].style: must be one of full, icon, text, got "big"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestLinkRowItems(t *testing.T) {
	const base = "render:\n  rows:\n"
	ok := []string{
		"    - items: [links/youtube]\n",
		"    - items: [links/youtube, links/website]\n    - items: [links]\n",
		"    - items: [links, links/youtube]\n",
		"    - items: [identity]\n    - items: [links/custom]\n",
	}
	for _, rows := range ok {
		if _, err := parseProfile(t, base+rows); err != nil {
			t.Fatalf("rows %q: %v", rows, err)
		}
	}
	for name, tt := range map[string]struct{ rows, want string }{
		"unknown kind":       {"    - items: [links/nope]\n", `unknown item "links/nope"`},
		"placed twice":       {"    - items: [links/youtube]\n    - items: [links/youtube]\n", `"links/youtube" is already placed`},
		"with the identity":  {"    - items: [links/youtube, identity]\n", "link items cannot share a row"},
		"with a card":        {"    - items: [stats, links]\n", "link items cannot share a row"},
		"links placed twice": {"    - items: [links]\n    - items: [links]\n", `"links" is already placed`},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseProfile(t, base+tt.rows)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
}
