package render

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"unicode/utf8"

	"github.com/wewesss/ghstats/config"
)

//go:embed icons/simple-icons.json
var simpleIconsJSON []byte

var iconSlugs = map[string]string{
	"itchio":          "itchdotio",
	"github_sponsors": "githubsponsors",
	"open_collective": "opencollective",
	"buy_me_a_coffee": "buymeacoffee",
	"ko_fi":           "kofi",
}

var simpleIcons = sync.OnceValue(func() map[string]string {
	m := map[string]string{}
	if err := json.Unmarshal(simpleIconsJSON, &m); err != nil {
		panic(fmt.Sprintf("render: simple-icons.json: %v", err))
	}
	return m
})

func iconPath(kind string) (string, bool) {
	slug := kind
	if s, ok := iconSlugs[kind]; ok {
		slug = s
	}
	p, ok := simpleIcons()[slug]
	return p, ok
}

func LinkBadgeSVG(w io.Writer, link config.ResolvedLink, t config.Theme, mode Mode) error {
	if mode != Dark && mode != Light {
		return fmt.Errorf("render: unknown mode %q", mode)
	}
	th := newTheme(t, mode)
	var icon string
	if link.Kind == "website" {
		icon = fmt.Sprintf(`<g transform="translate(15 11)" fill="none" stroke="%s" stroke-width="1.4"><circle cx="9" cy="9" r="8"/><path d="M1 9h16M9 1c2.6 2.4 2.6 13.6 0 16M9 1c-2.6 2.4-2.6 13.6 0 16"/></g>`, th.fg)
	} else {
		p, ok := iconPath(link.Kind)
		if !ok {
			return fmt.Errorf("render: no icon for link kind %q", link.Kind)
		}
		icon = fmt.Sprintf(`<g transform="translate(15 11) scale(.75)"><path d="%s" fill="%s"/></g>`, p, th.fg)
	}
	bw := int(jsRound(48 + float64(utf8.RuneCountInString(link.Label))*7.3 + 14))
	const bh = 40
	svg := fmt.Sprintf(`<svg viewBox="0 0 %d %d" width="%d" height="%d" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="%s">
    <rect x=".5" y=".5" width="%d" height="%d" rx="8" fill="%s" stroke="%s"/>%s
    <line x1="42" x2="42" y1="10" y2="30" stroke="%s"/>
    %s</svg>
`, bw, bh, bw, bh, esc(link.Kind), bw-1, bh-1, th.bg, th.border, icon, th.border,
		txt(52, 25, esc(link.Label), topt{fill: th.fg, size: 12, mono: true}))
	if _, err := io.WriteString(w, svg); err != nil {
		return fmt.Errorf("render: write svg: %w", err)
	}
	return nil
}
