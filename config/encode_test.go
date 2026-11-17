package config_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/sample"
)

func richConfig() config.Config {
	c := config.Default()
	c.Scope.Owners = []string{"wewesss", "miralabs"}
	c.Scope.IncludeForks = true
	c.Scope.ExcludeRepos = []string{"wewesss/old-*"}
	c.Scope.RepoWeights = map[string]float64{"wewesss/big": 0.5, "wewesss/small": 2}
	c.Languages.Mode = config.ModeAdjusted
	c.Languages.Weights = map[string]float64{"Lua": 0.35, "Go": 1.5}
	c.Languages.Exclude = []string{"Makefile"}
	c.Languages.Hide = []string{"Shell"}
	c.Languages.HiddenShare = config.HiddenShareRenormalize
	c.Languages.MaxVisible = 0
	c.Render.Blocks = sample.AllBlocks()
	c.Render.Languages.Colors = config.Palette{Custom: map[string]string{"Go": "#00add8", "Lua": "#000080"}}
	c.Render.Theme.Accent = config.Accent{Dark: "#112233", Light: "#445566"}
	c.Render.Theme.Overrides = map[string]string{"bg": "none", "fg": "#fedcba"}
	c.Render.Motion = false
	c.Render.Layout = config.LayoutWide
	c.Render.BasePath = "assets/ghstats/"
	c.Profile = sample.Profile().Profile
	c.Profile.Links = append(c.Profile.Links, config.Link{Kind: "mastodon", Value: "https://mastodon.social/@me", Label: "fediverse"})
	c.Profile.Links = append(c.Profile.Links, sample.CustomLinks()["named"])
	c.Profile.LinkStyle = config.LinkStyleIcon
	return c
}

func customPathConfig() config.Config {
	c := sample.Profile()
	c.Profile.Links = append(c.Profile.Links, sample.CustomLinks()["path"])
	c.Profile.LinkStyle = config.LinkStyleText
	return c
}

func allLinksConfig() config.Config {
	c := config.Default()
	c.Profile.Links = sample.AllLinks()
	return c
}

func TestEncodeRoundTrip(t *testing.T) {
	tests := map[string]config.Config{
		"default": config.Default(),
		"profile": sample.Profile(),
		"rich":    richConfig(),
		"custom":  customPathConfig(),
		"all":     allLinksConfig(),
	}
	for name, c := range tests {
		t.Run(name, func(t *testing.T) {
			if err := c.Validate(); err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			if err := config.Encode(&buf, c); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(buf.String(), "version: 1\n") {
				t.Fatalf("output must start with the version:\n%s", buf.String())
			}
			got, err := config.Parse(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("%v\n%s", err, buf.String())
			}
			if !reflect.DeepEqual(got, c) {
				t.Fatalf("round trip differs\ngot:  %+v\nwant: %+v\n%s", got, c, buf.String())
			}
		})
	}
}

func TestEncodeIndentsTwoSpaces(t *testing.T) {
	var buf bytes.Buffer
	if err := config.Encode(&buf, config.Default()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\nscope:\n  include_forks: false\n  include_archived: false\n") {
		t.Fatalf("want 2-space indentation:\n%s", buf.String())
	}
}

func TestOptionAccessorsAreCopies(t *testing.T) {
	p := config.Presets()
	p[0] = "changed"
	if config.Presets()[0] == "changed" {
		t.Fatal("Presets must return a copy")
	}
	blocks := config.BlockVariants()
	if len(blocks) != 3 || blocks[0].Name != "stats" || blocks[1].Name != "languages" || blocks[2].Name != "history" {
		t.Fatalf("blocks = %+v", blocks)
	}
	if !config.LinkKindIsURLOnly("website") || config.LinkKindIsURLOnly("youtube") {
		t.Fatal("URL-only kinds wrong")
	}
}
