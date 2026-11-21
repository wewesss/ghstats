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
}

func (it snippetItem) picture(base, width string) string {
	return picture(base, it.name, it.alt, width)
}

func oneLine(pic string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSuffix(pic, "\n"), "\n  ", ""), "\n", "")
}

func linkBlock(base string, links []config.ResolvedLink, align string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<p align=\"%s\">\n", align)
	for _, l := range links {
		pic := oneLine(picture(base, "links/"+l.Kind, l.Kind, ""))
		fmt.Fprintf(&b, "<a href=\"%s\">%s</a>\n", html.EscapeString(l.URL), pic)
	}
	b.WriteString("</p>\n")
	return b.String()
}

func Snippet(cfg config.Config) string {
	base := cfg.Render.BasePath
	var items []snippetItem
	if cfg.HasIdentity() {
		it := snippetItem{name: config.ItemIdentity, natural: 1000, solo: "100%", alt: "profile"}
		if cfg.Profile.Identity.Layout == config.LayoutCard {
			it.natural, it.solo = 495, "495"
		}
		items = append(items, it)
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

	links := cfg.Profile.ResolvedLinks()
	single := map[string]bool{}
	for _, row := range cfg.Render.Rows {
		for _, name := range row.Items {
			if kind, ok := config.LinkItemKind(name); ok {
				single[kind] = true
			}
		}
	}
	rest := func() []config.ResolvedLink {
		var out []config.ResolvedLink
		for _, l := range links {
			if !single[l.Kind] {
				out = append(out, l)
			}
		}
		return out
	}
	rowLinks := func(row config.Row) []config.ResolvedLink {
		var out []config.ResolvedLink
		for _, name := range row.Items {
			if name == config.ItemLinks {
				out = append(out, rest()...)
				continue
			}
			kind, ok := config.LinkItemKind(name)
			if !ok {
				continue
			}
			for _, l := range links {
				if l.Kind == kind {
					out = append(out, l)
				}
			}
		}
		return out
	}

	var parts []string
	placed := map[string]bool{}
	restPlaced := false
	for _, row := range cfg.Render.Rows {
		align := row.Align
		if align == "" {
			align = config.AlignCenter
		}
		if len(row.Items) > 0 && config.IsLinkItem(row.Items[0]) {
			for _, name := range row.Items {
				if name == config.ItemLinks {
					restPlaced = true
				}
			}
			if rl := rowLinks(row); len(rl) > 0 {
				parts = append(parts, linkBlock(base, rl, align))
			}
			continue
		}
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
		sum := 0
		for _, it := range inRow {
			sum += it.natural
		}
		pics := make([]string, len(inRow))
		for i, it := range inRow {
			w := it.solo
			if len(inRow) > 1 {
				w = fmt.Sprintf("%d%%", 98*it.natural/sum)
			} else if w == "100%" {
				w = "98%"
			}
			pics[i] = oneLine(it.picture(base, w))
		}
		parts = append(parts, fmt.Sprintf("<p align=\"%s\">\n  %s\n</p>\n", align, strings.Join(pics, " ")))
	}
	for _, it := range items {
		if !placed[it.name] {
			parts = append(parts, it.picture(base, it.solo))
		}
	}
	if !restPlaced {
		if rl := rest(); len(rl) > 0 {
			parts = append(parts, linkBlock(base, rl, config.AlignCenter))
		}
	}
	return strings.Join(parts, "\n")
}
