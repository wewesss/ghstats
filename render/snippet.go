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

func Snippet(cfg config.Config) string {
	base := cfg.Render.BasePath
	var parts []string
	if cfg.Profile.HasIdentity() {
		parts = append(parts, picture(base, "identity", "profile", "100%"))
	}
	for _, c := range cfg.Render.CardList() {
		width := "495"
		if c.Layout == config.LayoutWide {
			width = "100%"
		}
		alt := "GitHub stats"
		if c.Name != "stats" {
			alt += ": " + c.Name
		}
		parts = append(parts, picture(base, c.Name, alt, width))
	}
	if links := cfg.Profile.ResolvedLinks(); len(links) > 0 {
		var b strings.Builder
		b.WriteString("<p align=\"center\">\n")
		for _, l := range links {
			pic := picture(base, "links/"+l.Kind, l.Kind, "")
			pic = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSuffix(pic, "\n"), "\n  ", ""), "\n", "")
			fmt.Fprintf(&b, "<a href=\"%s\">%s</a>\n", html.EscapeString(l.URL), pic)
		}
		b.WriteString("</p>\n")
		parts = append(parts, b.String())
	}
	return strings.Join(parts, "\n")
}
