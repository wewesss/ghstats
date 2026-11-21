package render

import (
	"fmt"
	"html"
	"strings"

	"github.com/wewesss/ghstats/config"
)

func picture(base, name, alt, width string) string {
	attr := ""
	if width != "" {
		attr = fmt.Sprintf(` width="%s"`, width)
	}
	return fmt.Sprintf(`<picture>
  <source media="(prefers-color-scheme: dark)" srcset="%[1]s%[2]s-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="%[1]s%[2]s-light.svg">
  <img alt="%[3]s" src="%[1]s%[2]s-dark.svg"%[4]s>
</picture>
`, html.EscapeString(base), name, html.EscapeString(alt), attr)
}

type snippetItem struct {
	name    string
	natural int
	solo    string
	alt     string
	links   bool
}

func (it snippetItem) picture(base, width string) string {
	return picture(base, it.name, it.alt, width)
}

func oneLine(pic string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSuffix(pic, "\n"), "\n  ", ""), "\n", "")
}

func linkBlock(cfg config.Config, align string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<p align=\"%s\">\n", align)
	for _, l := range cfg.Profile.ResolvedLinks() {
		pic := oneLine(picture(cfg.Render.BasePath, "links/"+l.Kind, l.Kind, ""))
		fmt.Fprintf(&b, "<a href=\"%s\">%s</a>\n", html.EscapeString(l.URL), pic)
	}
	b.WriteString("</p>\n")
	return b.String()
}

func Snippet(cfg config.Config) string {
	base := cfg.Render.BasePath
	var items []snippetItem
	if cfg.Profile.HasIdentity() {
		items = append(items, snippetItem{name: config.ItemIdentity, natural: 1000, solo: "100%", alt: "profile"})
	}
	for _, c := range cfg.Render.CardList() {
		natural, solo := 495, "495"
		if effectiveLayout(c) == config.LayoutWide {
			natural, solo = 1000, "100%"
		}
		alt := "GitHub stats"
		if c.Name != "stats" {
			alt += ": " + c.Name
		}
		items = append(items, snippetItem{name: c.Name, natural: natural, solo: solo, alt: alt})
	}
	if len(cfg.Profile.ResolvedLinks()) > 0 {
		items = append(items, snippetItem{name: config.ItemLinks, links: true})
	}

	var parts []string
	placed := map[string]bool{}
	for _, row := range cfg.Render.Rows {
		var inRow []snippetItem
		for _, name := range row.Items {
			for _, it := range items {
				if it.name == name {
					inRow = append(inRow, it)
					placed[name] = true
				}
			}
		}
		if len(inRow) == 0 {
			continue
		}
		align := row.Align
		if align == "" {
			align = config.AlignCenter
		}
		if inRow[0].links {
			parts = append(parts, linkBlock(cfg, align))
			continue
		}
		sum := 0
		for _, it := range inRow {
			sum += it.natural
		}
		pics := make([]string, len(inRow))
		for i, it := range inRow {
			w := it.solo
			if len(inRow) > 1 {
				w = fmt.Sprintf("%d%%", 98*it.natural/sum)
			}
			pics[i] = oneLine(it.picture(base, w))
		}
		parts = append(parts, fmt.Sprintf("<p align=\"%s\">\n  %s\n</p>\n", align, strings.Join(pics, " ")))
	}
	for _, it := range items {
		if placed[it.name] {
			continue
		}
		if it.links {
			parts = append(parts, linkBlock(cfg, config.AlignCenter))
		} else {
			parts = append(parts, it.picture(base, it.solo))
		}
	}
	return strings.Join(parts, "\n")
}
