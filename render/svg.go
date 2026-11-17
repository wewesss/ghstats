package render

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/report"
)

const (
	width    = 495.0
	prefix   = "gs"
	sansFont = "-apple-system, BlinkMacSystemFont, 'Segoe UI', Inter, Helvetica, Arial, sans-serif"
	monoFont = "ui-monospace, 'SF Mono', 'Cascadia Mono', 'JetBrains Mono', Consolas, monospace"
)

var slotSeconds = map[string]int{"stats": 6, "languages": 5, "history": 7}

type animSpec struct {
	key  string
	d    float64
	ease string
	from string
	to   string
	base string
}

var animSpecs = []animSpec{
	{"fade", .7, "cubic-bezier(.2,.7,.2,1)", "opacity:0;transform:translateY(4px)", "opacity:1;transform:none", ""},
	{"fadeslow", 1.2, "ease-out", "opacity:0", "opacity:1", ""},
	{"grow", 1, "cubic-bezier(.2,.7,.2,1)", "transform:scaleX(0)", "transform:scaleX(1)", "transform-box:fill-box;transform-origin:0 50%"},
	{"growfast", .4, "cubic-bezier(.2,.7,.2,1)", "transform:scaleX(0)", "transform:scaleX(1)", "transform-box:fill-box;transform-origin:0 50%"},
	{"draw", 1.8, "cubic-bezier(.4,0,.2,1)", "stroke-dashoffset:1", "stroke-dashoffset:0", "stroke-dasharray:1"},
	{"drop", .7, "cubic-bezier(.2,.7,.2,1)", "opacity:0;transform:translateY(-22px)", "opacity:1;transform:none", ""},
	{"on", .02, "steps(1,end)", "opacity:0", "opacity:1", ""},
}

func specFor(key string) (animSpec, bool) {
	for _, a := range animSpecs {
		if a.key == key {
			return a, true
		}
	}
	return animSpec{}, false
}

func num(v float64) string {
	s := strconv.FormatFloat(v, 'f', 1, 64)
	s = strings.TrimSuffix(s, ".0")
	if s == "-0" {
		return "0"
	}
	return s
}

func esc(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func commas(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var out []byte
	for i := range len(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func anim(cls string, d float64) string {
	return fmt.Sprintf(` class="%s" style="animation-delay:%.3fs"`, cls, d)
}

type topt struct {
	fill   string
	size   float64
	weight int
	mono   bool
	anchor string
	ls     float64
	cls    string
	delay  float64
}

func txt(x, y float64, content string, o topt) string {
	size, weight, anchor, font := o.size, o.weight, o.anchor, sansFont
	if size == 0 {
		size = 12
	}
	if weight == 0 {
		weight = 400
	}
	if anchor == "" {
		anchor = "start"
	}
	if o.mono {
		font = monoFont
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<text x="%s" y="%s" font-family="%s" font-size="%g" font-weight="%d" fill="%s" text-anchor="%s"`,
		num(x), num(y), font, size, weight, o.fill, anchor)
	if o.ls != 0 {
		fmt.Fprintf(&b, ` letter-spacing="%g"`, o.ls)
	}
	if o.cls != "" {
		b.WriteString(anim(o.cls, o.delay))
	}
	fmt.Fprintf(&b, ">%s</text>", content)
	return b.String()
}

func label(y float64, s string, th theme, right string) string {
	out := txt(24, y, s, topt{fill: th.muted, size: 10, mono: true, ls: 1.4})
	if right != "" {
		out += txt(width-24, y, right, topt{fill: acc, size: 10, mono: true, anchor: "end", ls: 1})
	}
	return out
}

func SVG(w io.Writer, r report.Report, cfg config.Render, mode Mode) error {
	if mode != Dark && mode != Light {
		return fmt.Errorf("render: unknown mode %q", mode)
	}
	c := config.Default()
	c.Render = cfg
	if err := c.Validate(); err != nil {
		return err
	}
	rd := newRenderer(r, cfg, mode)
	if _, err := io.WriteString(w, rd.card()+"\n"); err != nil {
		return fmt.Errorf("render: write svg: %w", err)
	}
	return nil
}

func blockCSS(p string, cycle, n int) string {
	var out strings.Builder
	for _, a := range animSpecs {
		base := ""
		if a.base != "" {
			base = a.base + ";"
		}
		if cycle > 0 {
			fmt.Fprintf(&out, ".%s .%s{%sanimation:%ds linear infinite backwards}", p, a.key, base, cycle)
		} else {
			fmt.Fprintf(&out, ".%s .%s{%sanimation:%s%s %gs %s backwards}", p, a.key, base, p, a.key, a.d, a.ease)
			fmt.Fprintf(&out, "@keyframes %s%s{from{%s}}", p, a.key, a.from)
		}
	}
	if cycle > 0 {
		s := 100 / float64(n)
		fmt.Fprintf(&out, ".%s .slide{animation:%sslide %ds linear infinite}", p, p, cycle)
		fmt.Fprintf(&out, "@keyframes %sslide{0%%{opacity:0;transform:translateX(8px)}%.2f%%{opacity:1;transform:none}%.2f%%{opacity:1;transform:none}%.2f%%{opacity:0;transform:translateX(-8px)}100%%{opacity:0}}",
			p, s*.06, s*.94, s)
		fmt.Fprintf(&out, ".%s .dot{animation:%sdot %ds linear infinite}", p, p, cycle)
		fmt.Fprintf(&out, "@keyframes %sdot{0%%,%.2f%%{opacity:1}%.2f%%,100%%{opacity:.25}}", p, s, s+.01)
	}
	return out.String()
}

var slotAnimRe = regexp.MustCompile(`class="(\w+)" style="animation-delay:([\d.]+)s"`)

type frameSet struct {
	seen  map[string]bool
	order []string
}

func alignToSlot(svg, bp string, start, cycle int, frames *frameSet) string {
	return slotAnimRe.ReplaceAllStringFunc(svg, func(m string) string {
		sub := slotAnimRe.FindStringSubmatch(m)
		k := sub[1]
		a, ok := specFor(k)
		if !ok {
			return m
		}
		d, _ := strconv.ParseFloat(sub[2], 64)
		name := fmt.Sprintf("%s%s_%d", bp, k, int(jsRound(d*1000)))
		if !frames.seen[name] {
			frames.seen[name] = true
			p1 := d / float64(cycle) * 100
			p2 := math.Min((d+a.d)/float64(cycle)*100, 99.9)
			frames.order = append(frames.order, fmt.Sprintf("@keyframes %s{0%%{%s}%.3f%%{%s;animation-timing-function:%s}%.3f%%{%s}100%%{%s}}",
				name, a.from, p1, a.from, a.ease, p2, a.to, a.to))
		}
		return fmt.Sprintf(`class="%s" style="animation-name:%s;animation-delay:%ds"`, k, name, start)
	})
}

var accentAttrRe = regexp.MustCompile(`<(\w+)([^<>]*?)\s(fill|stroke|stop-color)="__ACC__"([^<>]*?)(/?)>`)

func accentize(svg string, th theme, cfg config.Render, mode Mode) string {
	if cfg.Theme.Accent.Name != config.AccentCycle || !cfg.Motion {
		return strings.ReplaceAll(svg, acc, th.accent)
	}
	idx := 0
	if mode == Light {
		idx = 1
	}
	list := make([]string, len(accentOrder))
	for i, name := range accentOrder {
		list[i] = accentTable[name][idx]
	}
	var vals []string
	for _, c := range list {
		vals = append(vals, c, c)
	}
	vals = append(vals, list[0])
	n := float64(len(list))
	kt := []string{"0"}
	for i := range list {
		kt = append(kt, fmt.Sprintf("%.3f", (float64(i)+.85)/n), fmt.Sprintf("%.3f", float64(i+1)/n))
	}
	values, keyTimes := strings.Join(vals, ";"), strings.Join(kt, ";")
	animate := func(attr string) string {
		return fmt.Sprintf(`<animate attributeName="%s" values="%s" keyTimes="%s" dur="40s" repeatCount="indefinite"/>`, attr, values, keyTimes)
	}
	for strings.Contains(svg, `="__ACC__"`) {
		next := accentAttrRe.ReplaceAllStringFunc(svg, func(m string) string {
			sub := accentAttrRe.FindStringSubmatch(m)
			tag, a, attr, b, self := sub[1], sub[2], sub[3], sub[4], sub[5]
			open := fmt.Sprintf(`<%s%s %s="%s"%s>`, tag, a, attr, list[0], b)
			if self != "" {
				return open + animate(attr) + "</" + tag + ">"
			}
			return open + animate(attr)
		})
		if next == svg {
			break
		}
		svg = next
	}
	return strings.ReplaceAll(svg, acc, list[0])
}

func (rd *renderer) card() string {
	th, cfg := rd.th, rd.cfg
	login := rd.rep.Login
	var body, styles []string

	body = append(body, txt(24, 31, esc(login), topt{fill: th.fg, size: 14, weight: 600}))
	body = append(body, txt(float64(23+9*utf8.RuneCountInString(login)), 31, "/ ghstats", topt{fill: th.muted, size: 12, mono: true}))
	body = append(body, fmt.Sprintf(`<circle cx="%s" cy="27" r="3" fill="%s"/><circle class="pulse" cx="%s" cy="27" r="3" fill="%s"/>`,
		num(width-132), acc, num(width-132), acc))
	synced := "never"
	if !rd.rep.CollectedAt.IsZero() {
		synced = rd.rep.CollectedAt.UTC().Format("2 Jan")
	}
	body = append(body, txt(width-24, 31, "synced · "+esc(synced), topt{fill: th.muted, size: 11, mono: true, anchor: "end"}))
	body = append(body, fmt.Sprintf(`<line x1="24" x2="%s" y1="46" y2="46" stroke="%s"/>`, num(width-24), th.border))
	if cfg.Motion {
		styles = append(styles, fmt.Sprintf(".%s .blink{animation:%sblink 1.1s steps(1,end) infinite}@keyframes %sblink{50%%{opacity:0}}", prefix, prefix, prefix))
		styles = append(styles, fmt.Sprintf(".%s .pulse{transform-box:fill-box;transform-origin:center;animation:%spulse 2.4s ease-out infinite}@keyframes %spulse{0%%{transform:scale(1);opacity:.55}100%%{transform:scale(3.2);opacity:0}}", prefix, prefix, prefix))
	}

	y := 56
	for bi, blk := range cfg.Blocks {
		n := len(blk.Variants)
		bp := fmt.Sprintf("%sb%d", prefix, bi)
		rotate := cfg.Motion && n > 1
		slot := slotSeconds[blk.Name]
		cycle := 0
		if rotate {
			cycle = n * slot
		}
		h := 0
		for _, v := range blk.Variants {
			h = max(h, rd.blockHeight(blk.Name, v))
		}
		if bi > 0 {
			body = append(body, fmt.Sprintf(`<line x1="24" x2="%s" y1="%d" y2="%d" stroke="%s"/>`, num(width-24), y, y, th.border))
			y += 8
		}
		if cfg.Motion {
			styles = append(styles, blockCSS(bp, cycle, n))
		}
		frames := &frameSet{seen: map[string]bool{}}
		var inner strings.Builder
		for i, v := range blk.Variants {
			svg := variants[blk.Name][v](rd, fmt.Sprintf("%sv%d", bp, i))
			if !rotate {
				if i == 0 {
					inner.WriteString(svg)
				}
				continue
			}
			start := i * slot
			opacity := 0
			if i == 0 {
				opacity = 1
			}
			fmt.Fprintf(&inner, `<g class="slide" style="animation-delay:%ds" opacity="%d">%s</g>`, start, opacity, alignToSlot(svg, bp, start, cycle, frames))
		}
		if len(frames.order) > 0 {
			styles = append(styles, strings.Join(frames.order, ""))
		}
		if rotate {
			for i := range blk.Variants {
				x := width - 24 - float64(n-1-i)*10
				opacity := ".25"
				if i == 0 {
					opacity = "1"
				}
				fmt.Fprintf(&inner, `<circle class="dot" style="animation-delay:%ds" cx="%s" cy="%d" r="2" fill="%s" opacity="%s"/>`,
					i*slot, num(x-2), h-4, th.muted, opacity)
			}
		}
		body = append(body, fmt.Sprintf(`<g class="%s" transform="translate(0 %d)">%s</g>`, bp, y, inner.String()))
		y += h + 6
	}
	height := y + 8

	grid := ""
	if cfg.Theme.Preset != "transparent" {
		grid = fmt.Sprintf(`<defs><pattern id="%sg" width="16" height="16" patternUnits="userSpaceOnUse"><circle cx="1" cy="1" r=".8" fill="%s" opacity=".07"/></pattern></defs><rect x="1" y="1" width="%s" height="%d" rx="10" fill="url(#%sg)"/>`,
			prefix, th.grid, num(width-2), height-2, prefix)
	}
	reduce := ""
	if cfg.Motion {
		reduce = fmt.Sprintf("@media (prefers-reduced-motion: reduce){.%s *{animation:none!important}}", prefix)
	}
	svg := fmt.Sprintf(`<svg class="%s" viewBox="0 0 %s %d" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="%s"><style>%s%s</style>
    <rect x=".5" y=".5" width="%s" height="%d" rx="10" fill="%s" stroke="%s"/>%s%s</svg>`,
		prefix, num(width), height, esc("ghstats card for "+login+" ("+string(rd.mode)+")"), strings.Join(styles, ""), reduce,
		num(width-1), height-1, th.bg, th.border, grid, strings.Join(body, ""))
	return accentize(svg, th, cfg, rd.mode)
}
