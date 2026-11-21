package render

import (
	"strings"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/report"
)

func effectiveLayout(c config.Card) string {
	if c.Layout == config.LayoutWide && len(c.Blocks) >= 2 {
		return config.LayoutWide
	}
	return config.LayoutCard
}

func Cards(r report.Report, cfg config.Config, mode Mode) (map[string]string, error) {
	files := map[string]string{}
	var sb strings.Builder
	cards := cfg.Render.CardList()
	draw := func(c config.Card, minHeight int) (int, error) {
		sb.Reset()
		rc := cfg.Render
		rc.Blocks = c.Blocks
		rc.Layout = effectiveLayout(c)
		h, err := svg(&sb, r, rc, mode, cardOpts{header: c.HasHeader(), minHeight: minHeight, hidePartial: cfg.Distribution.HidePartial})
		if err != nil {
			return 0, err
		}
		files[c.Name+"-"+string(mode)+".svg"] = sb.String()
		return h, nil
	}
	heights := map[string]int{}
	for _, c := range cards {
		h, err := draw(c, 0)
		if err != nil {
			return nil, err
		}
		heights[c.Name] = h
	}
	for _, row := range cfg.Render.Rows {
		var inRow []config.Card
		tallest := 0
		for _, name := range row.Items {
			for _, c := range cards {
				if c.Name == name {
					inRow = append(inRow, c)
					tallest = max(tallest, heights[name])
				}
			}
		}
		if len(inRow) < 2 {
			continue
		}
		for _, c := range inRow {
			if heights[c.Name] == tallest {
				continue
			}
			if _, err := draw(c, tallest); err != nil {
				return nil, err
			}
		}
	}
	if cfg.HasIdentity() {
		sb.Reset()
		if err := IdentitySVG(&sb, r, cfg, mode); err != nil {
			return nil, err
		}
		files["identity-"+string(mode)+".svg"] = sb.String()
	}
	for _, l := range cfg.Profile.ResolvedLinks() {
		sb.Reset()
		if err := LinkBadgeSVG(&sb, l, cfg.Render.Theme, mode); err != nil {
			return nil, err
		}
		files["links/"+l.Kind+"-"+string(mode)+".svg"] = sb.String()
	}
	return files, nil
}
