package config_test

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
)

func TestParseEmptyIsDefaults(t *testing.T) {
	for name, in := range map[string]string{"empty": "", "comments only": "# nothing here\n"} {
		t.Run(name, func(t *testing.T) {
			got, err := config.Parse(strings.NewReader(in))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, config.Default()) {
				t.Fatalf("got %+v, want defaults", got)
			}
		})
	}
}

func docDefaults(t *testing.T) (tables map[string]string, render map[string]string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "docs", "en", "guide", "configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	rowRe := regexp.MustCompile(`^\| ` + "`([a-z_.]+)`" + ` \| (.*?) \|`)
	literalRe := regexp.MustCompile("^(`[^`]*`|empty)$")
	markedRe := regexp.MustCompile("`([^`]+)` \\([^)]*default\\)|default `([^`]+)`")
	tables, render = map[string]string{}, map[string]string{}
	for _, line := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		m := rowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, cell := m[1], strings.TrimSpace(m[2])
		switch {
		case strings.HasPrefix(key, "render."):
			if d := markedRe.FindStringSubmatch(cell); d != nil {
				render[key] = d[1] + d[2]
			}
		case literalRe.MatchString(cell):
			tables[key] = strings.Trim(cell, "`")
		}
	}
	return tables, render
}

func emptyOr[T any](v []T) string {
	if len(v) == 0 {
		return "empty"
	}
	return fmt.Sprint(v)
}

func TestDefaultValuesMatchDocs(t *testing.T) {
	d := config.Default()
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	want, _ := docDefaults(t)
	got := map[string]string{
		"scope.owners":                emptyOr(d.Scope.Owners),
		"scope.include_forks":         fmt.Sprint(d.Scope.IncludeForks),
		"scope.include_archived":      fmt.Sprint(d.Scope.IncludeArchived),
		"scope.include_private":       fmt.Sprint(d.Scope.IncludePrivate),
		"scope.exclude_repos":         emptyOr(d.Scope.ExcludeRepos),
		"scope.repo_weights":          emptyOr(slices.Collect(maps.Keys(d.Scope.RepoWeights))),
		"languages.mode":              d.Languages.Mode,
		"languages.weights":           emptyOr(slices.Collect(maps.Keys(d.Languages.Weights))),
		"languages.exclude":           emptyOr(d.Languages.Exclude),
		"languages.hide":              emptyOr(d.Languages.Hide),
		"languages.hidden_share":      d.Languages.HiddenShare,
		"languages.max_visible":       fmt.Sprint(d.Languages.MaxVisible),
		"languages.other_label":       d.Languages.OtherLabel,
		"distribution.release_assets": fmt.Sprint(d.Distribution.ReleaseAssets),
		"distribution.clones":         fmt.Sprint(d.Distribution.Clones),
		"distribution.views":          fmt.Sprint(d.Distribution.Views),
	}
	for key, doc := range want {
		code, ok := got[key]
		if !ok {
			t.Errorf("%s is documented with default %q but the test does not know the key", key, doc)
			continue
		}
		if code != doc {
			t.Errorf("%s: code default %q, docs default %q", key, code, doc)
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			t.Errorf("%s has no default in docs/en/guide/configuration.md", key)
		}
	}
}

func TestParseKeepsDefaultsForMissingKeys(t *testing.T) {
	got, err := config.Parse(strings.NewReader("version: 1\nlanguages:\n  mode: adjusted\n  max_visible: 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Languages.Mode != "adjusted" || got.Languages.MaxVisible != 0 || got.Languages.OtherLabel != "Other" || !got.Scope.IncludePrivate {
		t.Fatalf("got %+v", got)
	}
}

func TestParseRejects(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"unknown top-level key", "version: 1\nbogus: true\n", "bogus"},
		{"unknown nested key", "version: 1\nscope:\n  include_forkz: true\n", "include_forkz"},
		{"missing version", "scope:\n  include_forks: true\n", "version"},
		{"wrong version", "version: 2\n", "version"},
		{"invalid yaml", "version: [1\n", "config: parse"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Parse(strings.NewReader(tt.in))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Config)
		want   string
	}{
		{"version", func(c *config.Config) { c.Version = 2 }, "version"},
		{"bad glob", func(c *config.Config) { c.Scope.ExcludeRepos = []string{"owner/[a-"} }, "scope.exclude_repos[0]"},
		{"negative repo weight", func(c *config.Config) { c.Scope.RepoWeights = map[string]float64{"o/r": -1} }, "scope.repo_weights"},
		{"repo weight key", func(c *config.Config) { c.Scope.RepoWeights = map[string]float64{"repo": 1} }, "owner/name"},
		{"duplicate repo weight", func(c *config.Config) { c.Scope.RepoWeights = map[string]float64{"o/r": 1, "O/R": 2} }, "scope.repo_weights"},
		{"bad mode", func(c *config.Config) { c.Languages.Mode = "fancy" }, "languages.mode"},
		{"negative language weight", func(c *config.Config) { c.Languages.Weights = map[string]float64{"Lua": -0.1} }, "languages.weights"},
		{"duplicate language weight", func(c *config.Config) { c.Languages.Weights = map[string]float64{"Lua": 1, "lua": 2} }, "languages.weights"},
		{"bad hidden_share", func(c *config.Config) { c.Languages.HiddenShare = "drop" }, "languages.hidden_share"},
		{"negative max_visible", func(c *config.Config) { c.Languages.MaxVisible = -1 }, "languages.max_visible"},
		{"empty other_label", func(c *config.Config) { c.Languages.OtherLabel = "" }, "languages.other_label"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := config.Default()
			tt.mutate(&c)
			err := c.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestExampleIsValid(t *testing.T) {
	c, err := config.Load("example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	w := c.Languages.Weights
	if w["Lua"] != 0.35 || w["TypeScript"] != 1.2 || w["HTML"] != 0 {
		t.Fatalf("weights = %v", w)
	}
	if _, ok := w["HTML"]; !ok {
		t.Fatal("HTML weight missing")
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := config.Load("does-not-exist.yaml"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestRenderDefaultsMatchDocs(t *testing.T) {
	r := config.Default().Render
	var blocks []string
	for _, b := range r.Blocks {
		blocks = append(blocks, b.Name+":"+strings.Join(b.Variants, "+"))
	}
	if got := strings.Join(blocks, ","); got != "stats:totals,languages:bars,history:views" {
		t.Fatalf("blocks = %s", got)
	}
	_, want := docDefaults(t)
	got := map[string]string{
		"render.private_bar":         fmt.Sprint(r.PrivateBar),
		"render.languages.colors":    r.Languages.Colors.Name,
		"render.languages.weighting": fmt.Sprint(r.Languages.ShowWeighting()),
		"render.theme.preset":        r.Theme.Preset,
		"render.theme.accent":        r.Theme.Accent.Name,
		"render.motion":              fmt.Sprint(r.Motion),
		"render.layout":              r.Layout,
		"render.base_path":           r.BasePath,
	}
	for key, code := range got {
		doc, ok := want[key]
		if !ok {
			t.Errorf("%s has no default marked in docs/en/guide/configuration.md", key)
			continue
		}
		if code != doc {
			t.Errorf("%s: code default %q, docs default %q", key, code, doc)
		}
	}
	for key, doc := range want {
		if _, ok := got[key]; !ok {
			t.Errorf("%s is documented with default %q but the test does not know the key", key, doc)
		}
	}
}

func TestRenderParsesFullExample(t *testing.T) {
	in := `version: 1
render:
  blocks:
    - stats: [totals, distribution]
    - languages: [bars, iso, cells, treemap, shift, terminal]
    - history: [views, clones]
  private_bar: false
  languages:
    colors:
      TypeScript: "#3178c6"
      Go: "#00ADD8"
  theme:
    preset: dune
    accent:
      dark: "#112233"
      light: "#445566"
    overrides:
      bg: none
      fg: "#ffffff"
  motion: false
`
	got, err := config.Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	r := got.Render
	if len(r.Blocks) != 3 || r.Blocks[1].Name != "languages" || len(r.Blocks[1].Variants) != 6 || r.Blocks[2].Variants[1] != "clones" {
		t.Fatalf("blocks = %+v", r.Blocks)
	}
	if r.PrivateBar || r.Motion {
		t.Fatalf("flags = %+v", r)
	}
	if r.Languages.Colors.Name != "" || r.Languages.Colors.Custom["Go"] != "#00ADD8" {
		t.Fatalf("colors = %+v", r.Languages.Colors)
	}
	if r.Theme.Preset != "dune" || r.Theme.Accent != (config.Accent{Dark: "#112233", Light: "#445566"}) || r.Theme.Overrides["bg"] != "none" {
		t.Fatalf("theme = %+v", r.Theme)
	}
}

func TestRenderStringAccentAndPalette(t *testing.T) {
	got, err := config.Parse(strings.NewReader("version: 1\nrender:\n  languages:\n    colors: github\n  theme:\n    accent: cycle\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Render.Languages.Colors.Name != "github" || got.Render.Theme.Accent.Name != "cycle" || got.Render.Theme.Preset != "noir" || len(got.Render.Blocks) != 3 {
		t.Fatalf("render = %+v", got.Render)
	}
}

func TestRenderRejects(t *testing.T) {
	tests := []struct {
		name, render, want string
	}{
		{"unknown block", "blocks:\n    - bogus: [a]", `unknown block "bogus"`},
		{"unknown variant", "blocks:\n    - stats: [nope]", `unknown variant "nope"`},
		{"empty variants", "blocks:\n    - stats: []", "at least one variant"},
		{"null variants", "blocks:\n    - stats:", "at least one variant"},
		{"duplicate block", "blocks:\n    - stats: [totals]\n    - stats: [distribution]", "more than once"},
		{"duplicate variant", "blocks:\n    - stats: [totals, totals]", `variant "totals" is listed more than once`},
		{"item not a map", "blocks:\n    - stats", "exactly one key"},
		{"item with two keys", "blocks:\n    - stats: [totals]\n      history: [views]", "exactly one key"},
		{"no blocks", "blocks: []", "at least one block"},
		{"unknown palette", "languages:\n    colors: neon", `unknown palette "neon"`},
		{"bad palette hex", "languages:\n    colors:\n      Go: \"#12345\"", "#rrggbb"},
		{"short palette hex", "languages:\n    colors:\n      Go: \"#fff\"", "#rrggbb"},
		{"unknown preset", "theme:\n    preset: neon", `unknown preset "neon"`},
		{"unknown accent", "theme:\n    accent: teal", `unknown accent "teal"`},
		{"accent missing light", "theme:\n    accent:\n      dark: \"#112233\"", "both dark and light"},
		{"accent missing dark", "theme:\n    accent:\n      light: \"#112233\"", "both dark and light"},
		{"accent bad hex", "theme:\n    accent:\n      dark: red\n      light: \"#112233\"", "theme.accent.dark"},
		{"accent unknown key", "theme:\n    accent:\n      dark: \"#112233\"\n      lite: \"#112233\"", `unknown key "lite"`},
		{"unknown override", "theme:\n    overrides:\n      accent: \"#112233\"", `unknown key "accent"`},
		{"bad override hex", "theme:\n    overrides:\n      fg: blue", "theme.overrides"},
		{"none only for bg", "theme:\n    overrides:\n      fg: none", "theme.overrides"},
		{"unknown render key", "bogus: 1", "bogus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Parse(strings.NewReader("version: 1\nrender:\n  " + tt.render + "\n"))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestRenderValidateJoinsErrors(t *testing.T) {
	c := config.Default()
	c.Render.Theme.Preset = "neon"
	c.Render.Languages.Colors = config.Palette{Name: "neon"}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "preset") || !strings.Contains(err.Error(), "palette") {
		t.Fatalf("err = %v", err)
	}
}

const twoCards = `version: 1
render:
  layout: card
  cards:
    - name: overview
      blocks:
        - stats: [totals]
        - history: [views]
    - name: langs
      layout: wide
      blocks:
        - languages: [treemap]
`

func TestRenderParsesCards(t *testing.T) {
	got, err := config.Parse(strings.NewReader(twoCards))
	if err != nil {
		t.Fatal(err)
	}
	cards := got.Render.CardList()
	if len(cards) != 2 || cards[0].Name != "overview" || cards[0].Layout != config.LayoutCard || cards[1].Name != "langs" || cards[1].Layout != config.LayoutWide {
		t.Fatalf("cards = %+v", cards)
	}
	if len(cards[0].Blocks) != 2 || cards[1].Blocks[0].Variants[0] != "treemap" {
		t.Fatalf("blocks = %+v", cards)
	}
}

func TestRenderCardsIgnoreBlocks(t *testing.T) {
	in := "version: 1\nrender:\n  blocks: []\n  cards:\n    - name: a\n      blocks:\n        - stats: [totals]\n"
	if _, err := config.Parse(strings.NewReader(in)); err != nil {
		t.Fatal(err)
	}
}

func TestRenderCardsRejects(t *testing.T) {
	card := func(name string) string {
		return "    - name: " + name + "\n      blocks:\n        - stats: [totals]\n"
	}
	many := ""
	for i := range 13 {
		many += card(fmt.Sprintf("c%d", i))
	}
	tests := []struct {
		name, cards, want string
	}{
		{"duplicate name", card("a") + card("a"), "used by more than one card"},
		{"reserved identity", card("identity"), `"identity" is reserved`},
		{"reserved links", card("links"), `"links" is reserved`},
		{"uppercase and space", card(`"My Card"`), "must match"},
		{"leading dash", card("-x"), "must match"},
		{"too long", card(strings.Repeat("a", 41)), "must match"},
		{"empty name", card(`""`), "name: must not be empty"},
		{"no blocks", "    - name: a\n      blocks: []\n", "render.cards[0].blocks: must list at least one block"},
		{"unknown block", "    - name: a\n      blocks:\n        - bogus: [x]\n", `render.cards[0].blocks[0]: unknown block "bogus"`},
		{"unknown layout", "    - name: a\n      layout: tall\n      blocks:\n        - stats: [totals]\n", `render.cards[0].layout: must be "card" or "wide"`},
		{"13 cards", many, "at most 12 cards"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Parse(strings.NewReader("version: 1\nrender:\n  cards:\n" + tt.cards))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestRenderAcceptsTwelveCards(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("version: 1\nrender:\n  cards:\n")
	for i := range 12 {
		fmt.Fprintf(&sb, "    - name: c%d\n      blocks:\n        - stats: [totals]\n", i)
	}
	if _, err := config.Parse(strings.NewReader(sb.String())); err != nil {
		t.Fatal(err)
	}
}

func TestCardListFallsBackToBlocks(t *testing.T) {
	r := config.Default().Render
	r.Layout = config.LayoutWide
	cards := r.CardList()
	if len(cards) != 1 || cards[0].Name != "stats" || cards[0].Layout != config.LayoutWide || len(cards[0].Blocks) != 3 {
		t.Fatalf("cards = %+v", cards)
	}
}
