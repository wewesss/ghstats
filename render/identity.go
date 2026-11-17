package render

import (
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/report"
)

const (
	identityWidth = 1000.0
	identitySlot  = 3
)

func IdentitySVG(w io.Writer, r report.Report, cfg config.Config, mode Mode) error {
	if mode != Dark && mode != Light {
		return fmt.Errorf("render: unknown mode %q", mode)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	if _, err := io.WriteString(w, identityCard(r, cfg, mode)+"\n"); err != nil {
		return fmt.Errorf("render: write svg: %w", err)
	}
	return nil
}

func identityHeight(focus int) int {
	if focus == 0 {
		return 288
	}
	return max(300, 198+34*(focus-1))
}

func identityCard(r report.Report, cfg config.Config, mode Mode) string {
	p, rc := cfg.Profile, cfg.Render
	th := newTheme(rc.Theme, mode)
	const W = identityWidth
	login := r.Login
	name := p.Name
	if name == "" {
		name = login
	}
	h := identityHeight(len(p.Focus))
	focus := p.Focus

	var words []string
	var starts []int
	for _, f := range focus {
		starts = append(starts, len(words))
		words = append(words, f.Building...)
	}
	cycle := len(words) * identitySlot

	var styles []string
	styles = append(styles, ".gs .word{opacity:0}.gs .word0{opacity:1}.gs .dom{opacity:.32}.gs .dom0{opacity:1}.gs .domdot{opacity:0}.gs .domdot0{opacity:1}")
	if rc.Motion {
		styles = append(styles, motionCSS()...)
		styles = append(styles, fmt.Sprintf(".%s .fade{animation:%sifade .8s cubic-bezier(.2,.7,.2,1) backwards}@keyframes %sifade{from{opacity:0;transform:translateY(5px)}}", prefix, prefix, prefix))
		if len(words) > 0 {
			s := 100 / float64(len(words))
			styles = append(styles, fmt.Sprintf(".%s .word{animation:%siw %ds linear infinite backwards}@keyframes %siw{0%%{opacity:0;transform:translateY(6px)}%.2f%%{opacity:1;transform:none}%.2f%%{opacity:1;transform:none}%.2f%%{opacity:0;transform:translateY(-6px)}100%%{opacity:0}}",
				prefix, prefix, cycle, prefix, s*.08, s*.9, s))
			seen := map[int]bool{}
			for _, f := range focus {
				n := len(f.Building)
				if seen[n] {
					continue
				}
				seen[n] = true
				d := 100 * float64(n) / float64(len(words))
				styles = append(styles, fmt.Sprintf(".%s .dom%d{animation:%sidm%d %ds linear infinite backwards}@keyframes %sidm%d{0%%,%.2f%%{opacity:1}%.2f%%,100%%{opacity:.32}}",
					prefix, n, prefix, n, cycle, prefix, n, d-.01, d))
				styles = append(styles, fmt.Sprintf(".%s .domdot%d{animation:%sidd%d %ds linear infinite backwards}@keyframes %sidd%d{0%%,%.2f%%{opacity:1}%.2f%%,100%%{opacity:0}}",
					prefix, n, prefix, n, cycle, prefix, n, d-.01, d))
			}
		}
	}

	var b []string
	b = append(b, txt(28, 37, esc(name), topt{fill: th.fg, size: 14, weight: 600}))
	b = append(b, txt(float64(29+9*utf8.RuneCountInString(name)), 37, "/ profile", topt{fill: th.muted, size: 12, mono: true}))
	dotX := W - 31
	if p.Location != "" {
		dotX = W - 150
	}
	b = append(b, fmt.Sprintf(`<circle cx="%s" cy="33" r="3" fill="%s"/><circle class="pulse" cx="%s" cy="33" r="3" fill="%s"/>`, num(dotX), acc, num(dotX), acc))
	if p.Location != "" {
		b = append(b, txt(W-28, 37, esc(p.Location), topt{fill: th.muted, size: 11, mono: true, anchor: "end"}))
	}
	b = append(b, fmt.Sprintf(`<line x1="28" x2="%s" y1="54" y2="54" stroke="%s"/>`, num(W-28), th.border))
	if len(focus) > 0 {
		b = append(b, fmt.Sprintf(`<line x1="640" x2="640" y1="74" y2="%d" stroke="%s"/>`, h-64, th.border))
	}

	if p.Tagline != "" {
		b = append(b, txt(28, 98, esc(p.Tagline), topt{fill: th.muted, size: 11, mono: true, ls: 2, cls: "fade"}))
	}
	b = append(b, txt(24, 172, esc(name), topt{fill: th.fg, size: 76, weight: 700, ls: -3, cls: "fade", delay: .1}))
	noteY := 216.0
	if len(focus) > 0 {
		b = append(b, txt(28, 216, fmt.Sprintf(`<tspan fill="%s">$</tspan> building`, acc), topt{fill: th.fg, size: 18, mono: true, cls: "fade", delay: .3}))
		for i, wd := range words {
			cls := "word"
			if i == 0 {
				cls = "word word0"
			}
			b = append(b, txt(156, 216, esc(wd), topt{fill: acc, size: 18, mono: true, cls: cls, delay: float64(i * identitySlot)}))
		}
		b = append(b, fmt.Sprintf(`<rect class="blink" x="28" y="236" width="9" height="3" fill="%s"/>`, acc))
		noteY = 252
	}
	if p.Note != "" {
		b = append(b, txt(28, noteY, esc(p.Note), topt{fill: th.muted, size: 12, mono: true}))
	}

	if len(focus) > 0 {
		b = append(b, txt(672, 98, "FOCUS", topt{fill: th.muted, size: 11, mono: true, ls: 2}))
		for i, f := range focus {
			y := 132 + i*34
			n := len(f.Building)
			cls, dotCls := "dom", "domdot"
			if i == 0 {
				cls, dotCls = "dom dom0", "domdot domdot0"
			}
			style := ""
			if rc.Motion {
				cls += fmt.Sprintf(" dom%d", n)
				dotCls += fmt.Sprintf(" domdot%d", n)
				delay := 0
				if starts[i] > 0 {
					delay = starts[i]*identitySlot - cycle
				}
				style = fmt.Sprintf(` style="animation-delay:%ds"`, delay)
			}
			b = append(b, fmt.Sprintf(`<g class="%s"%s>`, cls, style))
			b = append(b, txt(672, float64(y), fmt.Sprintf("%02d", i+1), topt{fill: th.muted, size: 11, mono: true}))
			b = append(b, txt(700, float64(y), esc(f.Name), topt{fill: th.fg, size: 13, weight: 500, mono: true, ls: 1.2}))
			b = append(b, `</g>`)
			b = append(b, fmt.Sprintf(`<circle class="%s"%s cx="%s" cy="%d" r="3" fill="%s"/>`, dotCls, style, num(W-36), y-4, acc))
		}
	}

	b = append(b, fmt.Sprintf(`<line x1="28" x2="%s" y1="%d" y2="%d" stroke="%s"/>`, num(W-28), h-44, h-44, th.border))
	b = append(b, txt(28, float64(h-20), esc("github.com/"+login), topt{fill: th.muted, size: 11, mono: true}))

	svg := shell(W, h, "ghstats profile for "+name+" ("+string(mode)+")", th, rc, styles, b)
	return accentize(svg, th, rc, mode)
}
