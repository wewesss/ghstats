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
	for _, c := range cfg.Render.CardList() {
		sb.Reset()
		rc := cfg.Render
		rc.Blocks = c.Blocks
		rc.Layout = effectiveLayout(c)
		if err := svg(&sb, r, rc, mode, c.HasHeader()); err != nil {
			return nil, err
		}
		files[c.Name+"-"+string(mode)+".svg"] = sb.String()
	}
	if cfg.Profile.HasIdentity() {
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
