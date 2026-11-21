package render

import (
	"fmt"
	"io"
	"math"
	"unicode/utf8"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/report"
)

const (
	identityWidth     = 1000.0
	identityCardWidth = 495.0
	identitySlot      = 3
	monoAdvance       = 0.6
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

func identityStyles(rc config.Render, focus []config.Focus, words []string, cycle int, withWords, withDom bool) []string {
	styles := []string{".gs .word{opacity:0}.gs .word0{opacity:1}.gs .dom{opacity:.5}.gs .dom0{opacity:1}.gs .domdot{opacity:0}.gs .domdot0{opacity:1}"}
	if !rc.Motion {
		return styles
	}
	styles = append(styles, motionCSS()...)
	styles = append(styles, fmt.Sprintf(".%s .fade{animation:%sifade .8s cubic-bezier(.2,.7,.2,1) backwards}@keyframes %sifade{from{opacity:0;transform:translateY(5px)}}", prefix, prefix, prefix))
	if len(words) == 0 {
		return styles
	}
	if withWords {
		s := 100 / float64(len(words))
		styles = append(styles, fmt.Sprintf(".%s .word{animation:%siw %ds linear infinite backwards}@keyframes %siw{0%%{opacity:0;transform:translateY(6px)}%.2f%%{opacity:1;transform:none}%.2f%%{opacity:1;transform:none}%.2f%%{opacity:0;transform:translateY(-6px)}100%%{opacity:0}}",
			prefix, prefix, cycle, prefix, s*.08, s*.9, s))
	}
	if withDom {
		seen := map[int]bool{}
		for _, f := range focus {
			n := len(f.Building)
			if seen[n] {
				continue
			}
			seen[n] = true
			d := 100 * float64(n) / float64(len(words))
			styles = append(styles, fmt.Sprintf(".%s .dom%d{animation:%sidm%d %ds linear infinite backwards}@keyframes %sidm%d{0%%,%.2f%%{opacity:1}%.2f%%,100%%{opacity:.5}}",
				prefix, n, prefix, n, cycle, prefix, n, d-.01, d))
			styles = append(styles, fmt.Sprintf(".%s .domdot%d{animation:%sidd%d %ds linear infinite backwards}@keyframes %sidd%d{0%%,%.2f%%{opacity:1}%.2f%%,100%%{opacity:0}}",
				prefix, n, prefix, n, cycle, prefix, n, d-.01, d))
		}
	}
	return styles
}

func buildingLines(x, y float64, words []string, th theme) []string {
	out := []string{txt(x, y, fmt.Sprintf(`<tspan fill="%s">$</tspan> building`, acc), topt{fill: th.fg, size: 18, mono: true, cls: "fade", delay: .3})}
	for i, wd := range words {
		cls := "word"
		if i == 0 {
			cls = "word word0"
		}
		line := `<tspan fill-opacity="0">$ building </tspan>` + esc(wd) + `<tspan class="blink">_</tspan>`
		out = append(out, txt(x, y, line, topt{fill: acc, size: 18, mono: true, cls: cls, delay: float64(i * identitySlot)}))
	}
	return out
}

func focusEntries(focus []config.Focus, starts []int, cycle int, rc config.Render, th theme, numX, nameX, dotX float64, y0, step int) []string {
	var out []string
	for i, f := range focus {
		y := y0 + i*step
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
		out = append(out, fmt.Sprintf(`<g class="%s"%s>`, cls, style))
		out = append(out, txt(numX, float64(y), fmt.Sprintf("%02d", i+1), topt{fill: th.muted, size: 11, mono: true}))
		out = append(out, txt(nameX, float64(y), esc(f.Name), topt{fill: th.fg, size: 13, weight: 500, mono: true, ls: 1.2}))
		out = append(out, `</g>`)
		out = append(out, fmt.Sprintf(`<circle class="%s"%s cx="%s" cy="%d" r="3" fill="%s"/>`, dotCls, style, num(dotX), y-4, acc))
	}
	return out
}

func focusWords(focus []config.Focus) (words []string, starts []int) {
	for _, f := range focus {
		starts = append(starts, len(words))
		words = append(words, f.Building...)
	}
	return words, starts
}

func identityCard(r report.Report, cfg config.Config, mode Mode) string {
	p := cfg.Profile
	if len(p.Identity.Show) > 0 || p.Identity.Layout == config.LayoutCard {
		return identityFlow(r, cfg, mode)
	}
	rc := cfg.Render
	th := newTheme(rc.Theme, mode)
	const W = identityWidth
	login := r.Login
	name := p.Name
	if name == "" {
		name = login
	}
	h := identityHeight(len(p.Focus))
	focus := p.Focus

	words, starts := focusWords(focus)
	cycle := len(words) * identitySlot
	styles := identityStyles(rc, focus, words, cycle, true, true)

	var b []string
	b = append(b, txt(28, 37, esc(name), topt{fill: th.fg, size: 14, weight: 600}))
	b = append(b, txt(headerSlashX(28, name), 37, "/ profile", topt{fill: th.muted, size: 12, mono: true}))
	dotX := W - 31
	if p.Location != "" {
		dotX = W - 28 - 11*monoAdvance*float64(utf8.RuneCountInString(p.Location)) - 12
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
		b = append(b, buildingLines(28, 216, words, th)...)
		noteY = 242
	}
	if p.Note != "" {
		b = append(b, txt(28, noteY, esc(p.Note), topt{fill: th.muted, size: 12, mono: true}))
	}

	if len(focus) > 0 {
		b = append(b, txt(672, 98, "FOCUS", topt{fill: th.muted, size: 11, mono: true, ls: 2}))
		b = append(b, focusEntries(focus, starts, cycle, rc, th, 672, 700, 658, 132, 34)...)
	}

	b = append(b, fmt.Sprintf(`<line x1="28" x2="%s" y1="%d" y2="%d" stroke="%s"/>`, num(W-28), h-44, h-44, th.border))
	b = append(b, txt(28, float64(h-20), esc("github.com/"+login), topt{fill: th.muted, size: 11, mono: true}))

	svg := shell(W, h, "ghstats profile for "+name+" ("+string(mode)+")", th, rc, styles, b)
	return accentize(svg, th, rc, mode)
}

func headerSlashX(mx float64, name string) float64 {
	return math.Round(mx + sansWidth(name, 14) + 9)
}

func fitNameSize(name string, avail float64) (size, spacing float64) {
	size = 76
	if em := sansWidth(name, 1); em > 0 {
		size = math.Max(16, math.Min(76, math.Floor(avail/em)))
	}
	return size, math.Round(-3*size/76*10) / 10
}

func identityFlow(r report.Report, cfg config.Config, mode Mode) string {
	p, rc := cfg.Profile, cfg.Render
	th := newTheme(rc.Theme, mode)
	show := p.Identity.Shows
	stacked := p.Identity.Layout == config.LayoutCard
	W, mx := identityWidth, 28.0
	if stacked {
		W, mx = identityCardWidth, 24.0
	}
	login := r.Login
	name := p.Name
	if name == "" {
		name = login
	}
	focus := p.Focus
	words, starts := focusWords(focus)
	cycle := len(words) * identitySlot
	showHeader := show(config.PartHeader)
	showName := show(config.PartName)
	showTagline := show(config.PartTagline) && p.Tagline != ""
	showLocation := show(config.PartLocation) && p.Location != ""
	showNote := show(config.PartNote) && p.Note != ""
	showBuilding := show(config.PartBuilding) && len(focus) > 0
	showFocus := show(config.PartFocus) && len(focus) > 0
	showFooter := show(config.PartFooter)

	styles := identityStyles(rc, focus, words, cycle, showBuilding, showFocus)

	var b []string
	top := 20
	if showHeader {
		top = 74
		b = append(b, txt(mx, 37, esc(name), topt{fill: th.fg, size: 14, weight: 600}))
		b = append(b, txt(headerSlashX(mx, name), 37, "/ profile", topt{fill: th.muted, size: 12, mono: true}))
		dotX := W - mx - 3
		if showLocation {
			dotX = W - mx - 11*monoAdvance*float64(utf8.RuneCountInString(p.Location)) - 12
		}
		b = append(b, fmt.Sprintf(`<circle cx="%s" cy="33" r="3" fill="%s"/><circle class="pulse" cx="%s" cy="33" r="3" fill="%s"/>`, num(dotX), acc, num(dotX), acc))
		if showLocation {
			b = append(b, txt(W-mx, 37, esc(p.Location), topt{fill: th.muted, size: 11, mono: true, anchor: "end"}))
		}
		b = append(b, fmt.Sprintf(`<line x1="%s" x2="%s" y1="54" y2="54" stroke="%s"/>`, num(mx), num(W-mx), th.border))
	}

	edge := top + 15
	place := func(asc, desc int) float64 {
		bl := edge + asc
		edge = bl + desc + 14
		return float64(bl)
	}
	placeLines := func(n, asc, desc, lead int) []float64 {
		ys := make([]float64, n)
		first := edge + asc
		for i := range ys {
			ys[i] = float64(first + i*lead)
		}
		edge = first + (n-1)*lead + desc + 14
		return ys
	}
	textEnd := func() int { return edge - 14 }

	hoist := showLocation && !showHeader
	firstPart := ""
	for _, c := range []struct {
		part string
		on   bool
	}{{"tagline", showTagline}, {"name", showName}, {"building", showBuilding}, {"note", showNote}} {
		if c.on {
			firstPart = c.part
			break
		}
	}
	reserve := func(part string) float64 {
		if hoist && part == firstPart {
			return 11*monoAdvance*float64(utf8.RuneCountInString(p.Location)) + 16
		}
		return 0
	}
	wrapAt := func(part string) float64 {
		if !stacked {
			return 0
		}
		return W - 2*mx - reserve(part)
	}
	locationDone := !hoist
	putLocation := func(y float64) {
		if !locationDone {
			b = append(b, txt(W-mx, y, esc(p.Location), topt{fill: th.muted, size: 11, mono: true, anchor: "end"}))
			locationDone = true
		}
	}

	availName := W - 2*mx
	if showFocus && !stacked {
		availName = 640 - 2*mx
	}
	if showTagline {
		lines := wrapMono(p.Tagline, wrapAt("tagline"), 11, 2)
		ys := placeLines(len(lines), 9, 3, 15)
		for i, l := range lines {
			b = append(b, txt(mx, ys[i], esc(l), topt{fill: th.muted, size: 11, mono: true, ls: 2, cls: "fade"}))
		}
		putLocation(ys[0])
	}
	if showName {
		size, ls := fitNameSize(name, availName-reserve("name"))
		putLocation(float64(edge + 9))
		b = append(b, txt(mx-4, place(57, 16), esc(name), topt{fill: th.fg, size: size, weight: 700, ls: ls, cls: "fade", delay: .1}))
	}
	if showBuilding {
		shown := words
		if limit := wrapAt("building"); limit > 0 {
			cols := monoCols(limit, 18, 0) - len("$ building ") - 1
			shown = make([]string, len(words))
			for i, wd := range words {
				shown[i] = clipRunes(wd, max(cols, 1))
			}
		}
		y := place(14, 3)
		putLocation(y)
		b = append(b, buildingLines(mx, y, shown, th)...)
	}
	if showNote {
		lines := wrapMono(p.Note, wrapAt("note"), 12, 0)
		ys := placeLines(len(lines), 9, 3, 17)
		for i, l := range lines {
			b = append(b, txt(mx, ys[i], esc(l), topt{fill: th.muted, size: 12, mono: true}))
		}
		putLocation(ys[0])
	}
	if !locationDone {
		putLocation(place(9, 3))
	}

	leftEnd := textEnd() + 25
	rightEnd := 0
	if showFocus {
		if stacked {
			rule := textEnd() + 18
			b = append(b, fmt.Sprintf(`<line x1="%s" x2="%s" y1="%d" y2="%d" stroke="%s"/>`, num(mx), num(W-mx), rule, rule, th.border))
			b = append(b, txt(mx, float64(rule+26), "FOCUS", topt{fill: th.muted, size: 11, mono: true, ls: 2}))
			first := rule + 56
			b = append(b, focusEntries(focus, starts, cycle, rc, th, mx, mx+28, mx-14, first, 30)...)
			leftEnd = first + 30*(len(focus)-1) + 28
		} else {
			b = append(b, txt(672, float64(top+24), "FOCUS", topt{fill: th.muted, size: 11, mono: true, ls: 2}))
			first := top + 58
			b = append(b, focusEntries(focus, starts, cycle, rc, th, 672, 700, 658, first, 34)...)
			rightEnd = first + 34*(len(focus)-1) + 22
		}
	}

	bodyEnd := max(leftEnd, rightEnd)
	if showFocus && !stacked {
		b = append(b, fmt.Sprintf(`<line x1="640" x2="640" y1="%d" y2="%d" stroke="%s"/>`, top, bodyEnd-20, th.border))
	}
	h := bodyEnd
	if showFooter {
		b = append(b, fmt.Sprintf(`<line x1="%s" x2="%s" y1="%d" y2="%d" stroke="%s"/>`, num(mx), num(W-mx), bodyEnd, bodyEnd, th.border))
		b = append(b, txt(mx, float64(bodyEnd+24), esc(clipMono("github.com/"+login, wrapAt("footer"), 11, 0)), topt{fill: th.muted, size: 11, mono: true}))
		h = bodyEnd + 44
	}

	svg := shell(W, h, "ghstats profile for "+name+" ("+string(mode)+")", th, rc, styles, b)
	return accentize(svg, th, rc, mode)
}
