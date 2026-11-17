package config

import (
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	PaletteAccent   = "accent"
	PaletteGitHub   = "github"
	PaletteMono     = "mono"
	PaletteSpectrum = "spectrum"
	PaletteDuotone  = "duotone"

	AccentCycle = "cycle"

	OverrideNone = "none"
)

var (
	blockVariants = map[string][]string{
		"stats":     {"totals", "distribution"},
		"languages": {"bars", "iso", "cells", "treemap", "shift", "terminal"},
		"history":   {"views", "clones"},
	}
	blockOrder   = []string{"stats", "languages", "history"}
	presets      = []string{"noir", "carbon", "graphite", "midnight", "slate", "dune", "transparent"}
	accents      = []string{"ice", "mint", "amber", "silver", "violet", "rose", "lime", "coral", AccentCycle}
	palettes     = []string{PaletteAccent, PaletteGitHub, PaletteMono, PaletteSpectrum, PaletteDuotone}
	overrideKeys = []string{"bg", "border", "fg", "muted", "faint"}
)

type Render struct {
	Blocks     []Block         `yaml:"blocks"`
	PrivateBar bool            `yaml:"private_bar"`
	Languages  RenderLanguages `yaml:"languages"`
	Theme      Theme           `yaml:"theme"`
	Motion     bool            `yaml:"motion"`
	Layout     string          `yaml:"layout"`
	BasePath   string          `yaml:"base_path"`
}

type Block struct {
	Name     string
	Variants []string
}

type RenderLanguages struct {
	Colors Palette `yaml:"colors"`
}

type Palette struct {
	Name   string
	Custom map[string]string
}

type Theme struct {
	Preset    string            `yaml:"preset"`
	Accent    Accent            `yaml:"accent"`
	Overrides map[string]string `yaml:"overrides,omitempty"`
}

type Accent struct {
	Name  string
	Dark  string
	Light string
}

func DefaultRender() Render {
	return Render{
		Blocks: []Block{
			{Name: "stats", Variants: []string{"totals"}},
			{Name: "languages", Variants: []string{"bars"}},
			{Name: "history", Variants: []string{"views"}},
		},
		PrivateBar: true,
		Languages:  RenderLanguages{Colors: Palette{Name: PaletteAccent}},
		Theme:      Theme{Preset: "noir", Accent: Accent{Name: "ice"}},
		Motion:     true,
		Layout:     LayoutCard,
		BasePath:   DefaultBasePath,
	}
}

func (b *Block) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode || len(n.Content) != 2 {
		return fmt.Errorf("line %d: a block item must be a map with exactly one key (block: [variants])", n.Line)
	}
	var variants []string
	if err := n.Content[1].Decode(&variants); err != nil {
		return fmt.Errorf("line %d: variants of block %q must be a list of names: %w", n.Line, n.Content[0].Value, err)
	}
	*b = Block{Name: n.Content[0].Value, Variants: variants}
	return nil
}

func (p *Palette) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		*p = Palette{Name: n.Value}
		return nil
	case yaml.MappingNode:
		var m map[string]string
		if err := n.Decode(&m); err != nil {
			return fmt.Errorf("line %d: languages.colors must map language names to \"#rrggbb\": %w", n.Line, err)
		}
		*p = Palette{Custom: m}
		return nil
	}
	return fmt.Errorf("line %d: languages.colors must be a palette name or a map of language colors", n.Line)
}

func (a *Accent) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		*a = Accent{Name: n.Value}
		return nil
	case yaml.MappingNode:
		var m map[string]string
		if err := n.Decode(&m); err != nil {
			return fmt.Errorf("line %d: theme.accent must map dark and light to \"#rrggbb\": %w", n.Line, err)
		}
		for k := range m {
			if k != "dark" && k != "light" {
				return fmt.Errorf("line %d: theme.accent: unknown key %q (want dark and light)", n.Line, k)
			}
		}
		*a = Accent{Dark: m["dark"], Light: m["light"]}
		return nil
	}
	return fmt.Errorf("line %d: theme.accent must be an accent name or a {dark, light} map", n.Line)
}

func (r Render) validate(fail func(format string, args ...any)) {
	if !validLayout(r.Layout) {
		fail("render.layout: must be %q or %q, got %q", LayoutCard, LayoutWide, r.Layout)
	}
	if !validBasePath(r.BasePath) {
		fail("render.base_path: must be a non-empty path ending with \"/\" without spaces, quotes, backslashes or \"..\", got %q", r.BasePath)
	}
	seen := map[string]bool{}
	if len(r.Blocks) == 0 {
		fail("render.blocks: must list at least one block")
	}
	for i, b := range r.Blocks {
		allowed, ok := blockVariants[b.Name]
		switch {
		case !ok:
			fail("render.blocks[%d]: unknown block %q (want %s)", i, b.Name, strings.Join(sortedKeys(blockVariants), ", "))
		case seen[b.Name]:
			fail("render.blocks[%d]: block %q is listed more than once", i, b.Name)
		}
		seen[b.Name] = true
		if len(b.Variants) == 0 {
			fail("render.blocks[%d]: block %q needs at least one variant", i, b.Name)
		}
		dups := map[string]bool{}
		for _, v := range b.Variants {
			switch {
			case ok && !slices.Contains(allowed, v):
				fail("render.blocks[%d]: unknown variant %q for block %q (want %s)", i, v, b.Name, strings.Join(allowed, ", "))
			case dups[v]:
				fail("render.blocks[%d]: variant %q is listed more than once in block %q", i, v, b.Name)
			}
			dups[v] = true
		}
	}

	c := r.Languages.Colors
	if c.Custom != nil {
		for _, name := range sortedKeys(c.Custom) {
			if !validHex(c.Custom[name]) {
				fail("render.languages.colors[%q]: must be \"#rrggbb\", got %q", name, c.Custom[name])
			}
		}
	} else if !slices.Contains(palettes, c.Name) {
		fail("render.languages.colors: unknown palette %q (want %s or a map)", c.Name, strings.Join(palettes, ", "))
	}

	if !slices.Contains(presets, r.Theme.Preset) {
		fail("render.theme.preset: unknown preset %q (want %s)", r.Theme.Preset, strings.Join(presets, ", "))
	}
	a := r.Theme.Accent
	if a.Name != "" {
		if !slices.Contains(accents, a.Name) {
			fail("render.theme.accent: unknown accent %q (want %s or {dark, light})", a.Name, strings.Join(accents, ", "))
		}
	} else {
		if a.Dark == "" || a.Light == "" {
			fail("render.theme.accent: a custom accent needs both dark and light")
		}
		for key, v := range map[string]string{"dark": a.Dark, "light": a.Light} {
			if v != "" && !validHex(v) {
				fail("render.theme.accent.%s: must be \"#rrggbb\", got %q", key, v)
			}
		}
	}
	for _, key := range sortedKeys(r.Theme.Overrides) {
		v := r.Theme.Overrides[key]
		switch {
		case !slices.Contains(overrideKeys, key):
			fail("render.theme.overrides: unknown key %q (want %s)", key, strings.Join(overrideKeys, ", "))
		case key == "bg" && v == OverrideNone:
		case !validHex(v):
			fail("render.theme.overrides[%q]: must be \"#rrggbb\"%s, got %q", key, bgNone(key), v)
		}
	}
}

func bgNone(key string) string {
	if key == "bg" {
		return " or \"none\""
	}
	return ""
}

func validHex(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}

func (b Block) MarshalYAML() (any, error) {
	return map[string][]string{b.Name: b.Variants}, nil
}

func (p Palette) MarshalYAML() (any, error) {
	if p.Custom != nil {
		return p.Custom, nil
	}
	return p.Name, nil
}

func (a Accent) MarshalYAML() (any, error) {
	if a.Name != "" {
		return a.Name, nil
	}
	return map[string]string{"dark": a.Dark, "light": a.Light}, nil
}

func BlockVariants() []Block {
	out := make([]Block, 0, len(blockOrder))
	for _, name := range blockOrder {
		out = append(out, Block{Name: name, Variants: slices.Clone(blockVariants[name])})
	}
	return out
}

func Presets() []string { return slices.Clone(presets) }

func Accents() []string { return slices.Clone(accents) }

func Palettes() []string { return slices.Clone(palettes) }

func Layouts() []string { return []string{LayoutCard, LayoutWide} }
