package config_test

import (
	"reflect"
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

func TestDefaultValues(t *testing.T) {
	d := config.Default()
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	if d.Languages.Mode != "raw" || d.Languages.HiddenShare != "other" || d.Languages.MaxVisible != 6 || d.Languages.OtherLabel != "Other" {
		t.Fatalf("languages = %+v", d.Languages)
	}
	if !d.Scope.IncludePrivate || d.Scope.IncludeForks || d.Scope.IncludeArchived {
		t.Fatalf("scope = %+v", d.Scope)
	}
	if !d.Distribution.ReleaseAssets || !d.Distribution.Clones || !d.Distribution.Views {
		t.Fatalf("distribution = %+v", d.Distribution)
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

func TestParseKeepsLanguageNamesAsWritten(t *testing.T) {
	in := "version: 1\nlanguages:\n  weights:\n    lua: 0.5\n    TYPESCRIPT: 1.5\n"
	got, err := config.Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if got.Languages.Weights["lua"] != 0.5 || got.Languages.Weights["TYPESCRIPT"] != 1.5 {
		t.Fatalf("weights = %v", got.Languages.Weights)
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
