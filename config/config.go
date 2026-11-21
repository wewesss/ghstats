// Package config defines the user configuration and its strict YAML loading.
package config

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	Version = 1

	ModeRaw      = "raw"
	ModeAdjusted = "adjusted"

	HiddenShareOther       = "other"
	HiddenShareRenormalize = "renormalize"
)

type Config struct {
	Version      int          `yaml:"version"`
	Scope        Scope        `yaml:"scope"`
	Languages    Languages    `yaml:"languages"`
	Distribution Distribution `yaml:"distribution"`
	Render       Render       `yaml:"render"`
	Profile      Profile      `yaml:"profile,omitempty"`
}

type Scope struct {
	Owners          []string           `yaml:"owners,omitempty"`
	IncludeForks    bool               `yaml:"include_forks"`
	IncludeArchived bool               `yaml:"include_archived"`
	IncludePrivate  bool               `yaml:"include_private"`
	ExcludeRepos    []string           `yaml:"exclude_repos,omitempty"`
	RepoWeights     map[string]float64 `yaml:"repo_weights,omitempty"`
}

type Languages struct {
	Mode        string             `yaml:"mode"`
	Weights     map[string]float64 `yaml:"weights,omitempty"`
	Exclude     []string           `yaml:"exclude,omitempty"`
	Hide        []string           `yaml:"hide,omitempty"`
	HiddenShare string             `yaml:"hidden_share"`
	MaxVisible  int                `yaml:"max_visible"`
	OtherLabel  string             `yaml:"other_label"`
}

type Distribution struct {
	ReleaseAssets bool `yaml:"release_assets"`
	Clones        bool `yaml:"clones"`
	Views         bool `yaml:"views"`
}

func Default() Config {
	return Config{
		Version: Version,
		Scope:   Scope{IncludePrivate: true},
		Languages: Languages{
			Mode:        ModeRaw,
			HiddenShare: HiddenShareOther,
			MaxVisible:  6,
			OtherLabel:  "Other",
		},
		Distribution: Distribution{ReleaseAssets: true, Clones: true, Views: true},
		Render:       DefaultRender(),
	}
}

func Parse(r io.Reader) (Config, error) {
	c := Default()
	c.Version = 0
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		if errors.Is(err, io.EOF) {
			return Default(), nil
		}
		return Config{}, fmt.Errorf("config: parse: %w", err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("config: open: %w", err)
	}
	defer f.Close()
	return Parse(f)
}

func (c Config) HasIdentity() bool {
	if c.Profile.HasIdentity() || len(c.Profile.Identity.Show) > 0 {
		return true
	}
	for _, row := range c.Render.Rows {
		if slices.Contains(row.Items, ItemIdentity) {
			return true
		}
	}
	return false
}

func (c Config) Validate() error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("config: "+format, args...))
	}

	if c.Version != Version {
		fail("version: must be %d, got %d", Version, c.Version)
	}

	for i, p := range c.Scope.ExcludeRepos {
		if _, err := path.Match(strings.ToLower(p), ""); err != nil {
			fail("scope.exclude_repos[%d]: invalid pattern %q: %v", i, p, err)
		}
	}
	for _, name := range sortedKeys(c.Scope.RepoWeights) {
		w := c.Scope.RepoWeights[name]
		if !strings.Contains(name, "/") {
			fail("scope.repo_weights: key %q must be owner/name", name)
		}
		if !validWeight(w) {
			fail("scope.repo_weights[%q]: must be a finite number >= 0, got %v", name, w)
		}
	}
	if dup := duplicateFold(c.Scope.RepoWeights); dup != "" {
		fail("scope.repo_weights: %q is defined more than once (names are case-insensitive)", dup)
	}

	switch c.Languages.Mode {
	case ModeRaw, ModeAdjusted:
	default:
		fail("languages.mode: must be %q or %q, got %q", ModeRaw, ModeAdjusted, c.Languages.Mode)
	}
	for _, name := range sortedKeys(c.Languages.Weights) {
		if w := c.Languages.Weights[name]; !validWeight(w) {
			fail("languages.weights[%q]: must be a finite number >= 0, got %v", name, w)
		}
	}
	if dup := duplicateFold(c.Languages.Weights); dup != "" {
		fail("languages.weights: %q is defined more than once (names are case-insensitive)", dup)
	}
	switch c.Languages.HiddenShare {
	case HiddenShareOther, HiddenShareRenormalize:
	default:
		fail("languages.hidden_share: must be %q or %q, got %q", HiddenShareOther, HiddenShareRenormalize, c.Languages.HiddenShare)
	}
	if c.Languages.MaxVisible < 0 {
		fail("languages.max_visible: must be >= 0, got %d", c.Languages.MaxVisible)
	}
	if c.Languages.OtherLabel == "" {
		fail("languages.other_label: must not be empty")
	}
	c.Render.validate(fail)
	c.Profile.validate(fail)
	return errors.Join(errs...)
}

func validWeight(w float64) bool {
	return w >= 0 && !math.IsInf(w, 0)
}

func duplicateFold(m map[string]float64) string {
	seen := make(map[string]bool, len(m))
	for _, k := range sortedKeys(m) {
		lk := strings.ToLower(k)
		if seen[lk] {
			return k
		}
		seen[lk] = true
	}
	return ""
}
