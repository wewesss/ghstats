// Command showcase regenerates the static SVG gallery from the sample report.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/sample"
)

func main() {
	out := flag.String("out", "showcase", "directory receiving the gallery")
	flag.Parse()
	if err := generate(*out); err != nil {
		fmt.Fprintln(os.Stderr, "showcase:", err)
		os.Exit(1)
	}
}

func generate(dir string) error {
	if err := clean(dir); err != nil {
		return err
	}
	files, err := build()
	if err != nil {
		return err
	}
	for name, data := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func clean(dir string) error {
	old, err := filepath.Glob(filepath.Join(dir, "*.svg"))
	if err != nil {
		return err
	}
	for _, p := range old {
		if err := os.Remove(p); err != nil {
			return err
		}
	}
	return os.RemoveAll(filepath.Join(dir, "badges"))
}

func build() (map[string]string, error) {
	files := map[string]string{}
	rep := sample.Report()
	profile := sample.Profile()
	profile.Render.Motion = false

	for _, mode := range []render.Mode{render.Dark, render.Light} {
		for _, blk := range sample.AllBlocks() {
			for _, v := range blk.Variants {
				cfg := sample.FullRender()
				cfg.Motion = false
				cfg.Blocks = []config.Block{{Name: blk.Name, Variants: []string{v}}}
				svg, err := svgString(func(sb *strings.Builder) error { return render.SVG(sb, rep, cfg, mode) })
				if err != nil {
					return nil, err
				}
				files[fmt.Sprintf("%s-%s-%s.svg", blk.Name, v, mode)] = svg
			}
		}

		wide := sample.WideRender()
		wide.Motion = false
		svg, err := svgString(func(sb *strings.Builder) error { return render.SVG(sb, rep, wide, mode) })
		if err != nil {
			return nil, err
		}
		files[fmt.Sprintf("wide-%s.svg", mode)] = svg

		svg, err = svgString(func(sb *strings.Builder) error { return render.IdentitySVG(sb, rep, profile, mode) })
		if err != nil {
			return nil, err
		}
		files[fmt.Sprintf("identity-%s.svg", mode)] = svg

		for _, style := range []string{config.LinkStyleFull, config.LinkStyleIcon, config.LinkStyleText} {
			for _, l := range resolve(profile, sample.AllLinks(), style) {
				name := fmt.Sprintf("badges/%s-%s.svg", l.Kind, mode)
				if style != config.LinkStyleFull {
					if !slices.Contains(styleKinds, l.Kind) {
						continue
					}
					name = fmt.Sprintf("badges/styles/%s-%s-%s.svg", l.Kind, style, mode)
				}
				if err := addBadge(files, name, l, profile.Render.Theme, mode); err != nil {
					return nil, err
				}
			}
		}

		variants := sample.CustomLinks()
		for _, variant := range []string{"named", "path", "none"} {
			for _, l := range resolve(profile, []config.Link{variants[variant]}, config.LinkStyleFull) {
				if err := addBadge(files, fmt.Sprintf("badges/custom/%s-%s.svg", variant, mode), l, profile.Render.Theme, mode); err != nil {
					return nil, err
				}
			}
		}
	}
	return files, nil
}

var styleKinds = []string{"youtube", "discord", "github_sponsors"}

func resolve(cfg config.Config, links []config.Link, style string) []config.ResolvedLink {
	p := cfg.Profile
	p.Links = links
	p.LinkStyle = style
	return p.ResolvedLinks()
}

func addBadge(files map[string]string, name string, l config.ResolvedLink, th config.Theme, mode render.Mode) error {
	svg, err := svgString(func(sb *strings.Builder) error { return render.LinkBadgeSVG(sb, l, th, mode) })
	if err != nil {
		return err
	}
	files[name] = svg
	return nil
}

func svgString(f func(sb *strings.Builder) error) (string, error) {
	var sb strings.Builder
	if err := f(&sb); err != nil {
		return "", err
	}
	return sb.String(), nil
}
