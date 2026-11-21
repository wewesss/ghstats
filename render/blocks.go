package render

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/report"
)

const (
	acc      = "__ACC__"
	maxRows  = 7
	minWeeks = 4
)

type langRow struct {
	name  string
	v     float64
	raw   float64
	mult  string
	other bool
}

type renderer struct {
	th      theme
	mode    Mode
	cfg     config.Render
	rep     report.Report
	rows    []langRow
	maxV    float64
	private bool
	header  bool

	minHeight int
	yshift    int
}

type variantFn func(rd *renderer, p string) string

var variants = map[string]map[string]variantFn{
	"stats": {
		"totals":       (*renderer).totals,
		"distribution": (*renderer).distribution,
	},
	"languages": {
		"bars":     (*renderer).bars,
		"iso":      (*renderer).iso,
		"cells":    (*renderer).cells,
		"treemap":  (*renderer).treemap,
		"shift":    (*renderer).shift,
		"terminal": (*renderer).terminal,
	},
	"history": {
		"views":  func(rd *renderer, p string) string { return rd.history(p, report.ChannelViews, "VIEWS") },
		"clones": func(rd *renderer, p string) string { return rd.history(p, report.ChannelClones, "CLONES") },
	},
}

func (rd *renderer) blockHeight(name, variant string) int {
	switch name {
	case "stats":
		if variant == "totals" && rd.private {
			return 92
		}
		if variant == "totals" {
			return 66
		}
		return 92
	case "languages":
		if variant == "bars" && !rd.cfg.Languages.ShowWeighting() && len(rd.rows) > 0 {
			return 48 + (len(rd.rows)-1)*22 + 10
		}
		return 250
	}
	return 104
}

func newRenderer(r report.Report, cfg config.Render, mode Mode) *renderer {
	rd := &renderer{th: newTheme(cfg.Theme, mode), mode: mode, cfg: cfg, rep: r}
	rd.rows = buildRows(r.Languages, cfg.Languages.ShowWeighting())
	for _, row := range rd.rows {
		rd.maxV = max(rd.maxV, row.v)
	}
	rd.private = cfg.PrivateBar && r.Totals.PrivateRepos > 0
	return rd
}

func otherName(l report.Languages, tail []string) string {
	if len(l.Raw) > 0 {
		real := make(map[string]bool, len(l.Raw))
		for _, s := range l.Raw {
			real[s.Name] = true
		}
		for _, n := range tail {
			if !real[n] {
				return n
			}
		}
	}
	return config.Default().Languages.OtherLabel
}

func buildRows(l report.Languages, weighting bool) []langRow {
	type entry struct {
		name  string
		share float64
	}
	var entries []entry
	for _, d := range l.Display {
		if d.Share > 0 && !math.IsInf(d.Share, 0) && !math.IsNaN(d.Share) {
			entries = append(entries, entry{d.Name, d.Share * 100})
		}
	}
	if len(entries) > maxRows {
		tail := make([]string, 0, len(entries)-maxRows+1)
		for _, e := range entries[maxRows-1:] {
			tail = append(tail, e.name)
		}
		folded := entry{name: otherName(l, tail)}
		for _, e := range entries[maxRows-1:] {
			folded.share += e.share
		}
		entries = append(entries[:maxRows-1], folded)
	}

	rawBy := make(map[string]float64, len(l.Raw))
	for _, s := range l.Raw {
		rawBy[s.Name] = s.Share * 100
	}
	kept := make(map[string]bool, len(entries))
	for _, e := range entries {
		kept[e.name] = true
	}
	otherIdx := -1
	for i, e := range entries {
		if _, real := rawBy[e.name]; !real {
			otherIdx = i
		}
	}

	rows := make([]langRow, len(entries))
	var hiddenRaw, sumRaw float64
	for name, v := range rawBy {
		if !kept[name] {
			hiddenRaw += v
		}
	}
	for i, e := range entries {
		rows[i] = langRow{name: e.name, v: e.share, other: i == otherIdx}
		if v, ok := rawBy[e.name]; ok {
			rows[i].raw = v
		}
		if w, ok := l.Multipliers[e.name]; ok && weighting {
			rows[i].mult = fmt.Sprintf("×%g", w)
		}
	}
	if otherIdx >= 0 {
		rows[otherIdx].raw += hiddenRaw
	}
	for _, row := range rows {
		sumRaw += row.raw
	}
	if otherIdx < 0 && sumRaw > 0 {
		for i := range rows {
			rows[i].raw = rows[i].raw * 100 / sumRaw
		}
	}
	return rows
}

func (rd *renderer) nameFill(r langRow) string {
	if r.other {
		return rd.th.muted
	}
	return rd.th.fg
}

func pct(v float64) string { return fmt.Sprintf("%.1f%%", v) }

func (rd *renderer) langColors() []string {
	th := rd.th
	pick := func(list []string, i int) string { return list[min(i, len(list)-1)] }
	secondName := "mint"
	if rd.cfg.Theme.Accent.Name == "mint" {
		secondName = "violet"
	}
	modeIdx := 0
	if rd.mode == Light {
		modeIdx = 1
	}
	second := hex2rgb(accentTable[secondName][modeIdx])
	pal := rd.cfg.Languages.Colors

	out := make([]string, len(rd.rows))
	for i, row := range rd.rows {
		switch {
		case row.other:
			out[i] = otherColor[rd.mode]
		case pal.Custom != nil:
			if c, ok := customColor(pal.Custom, row.name); ok {
				out[i] = c
			} else {
				out[i] = readable(githubColor(row.name), ground[rd.mode])
			}
		default:
			switch pal.Name {
			case config.PaletteGitHub:
				out[i] = readable(githubColor(row.name), ground[rd.mode])
			case config.PaletteMono:
				out[i] = pick(grays[rd.mode], i)
			case config.PaletteSpectrum:
				out[i] = pick(spectrum[rd.mode], i)
			case config.PaletteDuotone:
				out[i] = mixTo(th.accent, second, min(float64(i)/5, 1))
			default:
				if i == 0 {
					out[i] = acc
				} else {
					out[i] = pick(grays[rd.mode], i)
				}
			}
		}
	}
	return out
}

func customColor(m map[string]string, name string) (string, bool) {
	if c, ok := m[name]; ok {
		return strings.ToLower(c), true
	}
	for k, c := range m {
		if strings.EqualFold(k, name) {
			return strings.ToLower(c), true
		}
	}
	return "", false
}

func (rd *renderer) modeTag() string {
	if !rd.cfg.Languages.ShowWeighting() {
		return ""
	}
	return esc(rd.rep.Languages.Mode)
}

func (rd *renderer) emptyLanguages(title string) string {
	return label(22, title, rd.th, rd.modeTag()) +
		txt(width/2, 130, "no language data yet", topt{fill: rd.th.muted, size: 11, mono: true, anchor: "middle"})
}

func (rd *renderer) langLabel(title string) string {
	return label(22, title, rd.th, rd.modeTag())
}

func (rd *renderer) metric(channel, window string) string {
	for _, m := range rd.rep.Distribution {
		if m.Channel == channel && (m.Window == window || window == report.WindowSincePrefix && strings.HasPrefix(m.Window, window)) {
			if m.Count == nil || m.Reliability == report.ReliabilityUnavailable {
				return "n/a"
			}
			return commas(*m.Count)
		}
	}
	return "n/a"
}

func (rd *renderer) statRow(items [4][2]string) string {
	th := rd.th
	var s strings.Builder
	for i, it := range items {
		x := float64(24 + i*112)
		fill := th.fg
		if it[0] == "n/a" {
			fill = th.muted
		}
		s.WriteString(txt(x, 34, esc(it[0]), topt{fill: fill, size: 22, weight: 600, cls: "fade", delay: .05 * float64(i)}))
		s.WriteString(txt(x, 52, it[1], topt{fill: th.muted, size: 10, mono: true, ls: 1.4, cls: "fade", delay: .05*float64(i) + .05}))
	}
	return s.String()
}

func (rd *renderer) totals(p string) string {
	th, t := rd.th, rd.rep.Totals
	s := rd.statRow([4][2]string{
		{commas(int64(t.Stars)), "STARS"},
		{commas(int64(t.Repos)), "REPOS"},
		{commas(int64(t.Forks)), "FORKS"},
		{commas(int64(t.Followers)), "FOLLOWERS"},
	})
	if rd.private {
		s += txt(24, 80, fmt.Sprintf("PRIVATE %d", t.PrivateRepos), topt{fill: th.muted, size: 10, mono: true, ls: 1.2})
		for i, b := range [][2]float64{{104, 58}, {168, 44}, {218, 76}, {300, 38}} {
			s += fmt.Sprintf(`<rect%s x="%s" y="71" width="%s" height="11" rx="1" fill="%s" opacity=".9"/>`,
				anim("growfast", .5+float64(i)*.15), num(b[0]), num(b[1]), th.fg)
		}
		s += txt(width-24, 80, "names hidden", topt{fill: th.muted, size: 10, mono: true, anchor: "end"})
	}
	return s
}

func (rd *renderer) distribution(p string) string {
	th := rd.th
	s := rd.statRow([4][2]string{
		{rd.metric(report.ChannelReleaseAssets, report.WindowAllTime), "RELEASES"},
		{rd.metric(report.ChannelClones, report.WindowLast14Days), "CLONES 14D"},
		{rd.metric(report.ChannelViews, report.WindowLast14Days), "VIEWS 14D"},
		{rd.metric(report.ChannelViews, report.WindowSincePrefix), "VIEWS TOTAL"},
	})
	return s + txt(24, 80, "release assets all time · traffic summed across repositories", topt{fill: th.muted, size: 10, mono: true, cls: "fade", delay: .3})
}

func (rd *renderer) stacked(y float64, shares []float64, cols []string, d float64) string {
	x, total := 64.0, width-24-64
	var s strings.Builder
	for i, v := range shares {
		w := total * v / 100
		fmt.Fprintf(&s, `<rect%s x="%s" y="%s" width="%s" height="6" rx="1" fill="%s"/>`,
			anim("grow", d+float64(i)*.05), num(x), num(y), num(math.Max(w-2, 1)), cols[i])
		x += w
	}
	return s.String()
}

func (rd *renderer) bars(p string) string {
	th := rd.th
	if len(rd.rows) == 0 {
		return rd.emptyLanguages("LANGUAGES · BARS")
	}
	cols := rd.langColors()
	var s strings.Builder
	s.WriteString(rd.langLabel("LANGUAGES · BARS"))
	for i, r := range rd.rows {
		y := float64(48 + i*22)
		d := .2 + float64(i)*.07
		w := 250 * r.v / rd.maxV
		s.WriteString(txt(24, y, esc(r.name), topt{fill: rd.nameFill(r), cls: "fade", delay: d}))
		fmt.Fprintf(&s, `<rect x="112" y="%s" width="250" height="3" rx="1.5" fill="%s"/>`, num(y-6), th.faint)
		fmt.Fprintf(&s, `<rect%s x="112" y="%s" width="%s" height="3" rx="1.5" fill="%s"/>`, anim("grow", d), num(y-6), num(w), cols[i])
		if r.mult != "" {
			s.WriteString(txt(112+w+8, y-1, r.mult, topt{fill: acc, size: 10, mono: true, cls: "fade", delay: d + .6}))
		}
		s.WriteString(txt(width-24, y, pct(r.v), topt{fill: th.muted, size: 11, mono: true, anchor: "end", cls: "fade", delay: d}))
	}
	if !rd.cfg.Languages.ShowWeighting() {
		return s.String()
	}
	fmt.Fprintf(&s, `<line x1="24" x2="%s" y1="198" y2="198" stroke="%s" stroke-dasharray="2 4"/>`, num(width-24), th.border)
	raw := make([]float64, len(rd.rows))
	adj := make([]float64, len(rd.rows))
	for i, r := range rd.rows {
		raw[i], adj[i] = r.raw, r.v
	}
	s.WriteString(txt(24, 220, "RAW", topt{fill: th.muted, size: 10, mono: true, ls: 1.2, cls: "fade", delay: .8}))
	s.WriteString(rd.stacked(213, raw, cols, .8))
	s.WriteString(txt(24, 240, "ADJ", topt{fill: th.fg, size: 10, mono: true, ls: 1.2, cls: "fade", delay: 1}))
	s.WriteString(rd.stacked(233, adj, cols, 1))
	return s.String()
}

func squarify(vals []float64, x, y, w, h float64) [][4]float64 {
	var total float64
	for _, v := range vals {
		total += v
	}
	areas := make([]float64, len(vals))
	for i, v := range vals {
		areas[i] = v * w * h / total
	}
	var out [][4]float64
	rx, ry, rw, rh := x, y, w, h
	var row []float64
	sum := func(r []float64) float64 {
		var s float64
		for _, a := range r {
			s += a
		}
		return s
	}
	worst := func(r []float64, side float64) float64 {
		s := sum(r)
		m := math.Inf(-1)
		for _, a := range r {
			m = math.Max(m, math.Max(side*side*a/(s*s), s*s/(side*side*a)))
		}
		return m
	}
	lay := func(r []float64) {
		s := sum(r)
		if rw >= rh {
			cw := s / rh
			cy := ry
			for _, a := range r {
				out = append(out, [4]float64{rx, cy, cw, a / cw})
				cy += a / cw
			}
			rx += cw
			rw -= cw
		} else {
			ch := s / rw
			cx := rx
			for _, a := range r {
				out = append(out, [4]float64{cx, ry, a / ch, ch})
				cx += a / ch
			}
			ry += ch
			rh -= ch
		}
	}
	for i := 0; i < len(areas); {
		side := math.Min(rw, rh)
		if len(row) == 0 || worst(append(slices.Clone(row), areas[i]), side) <= worst(row, side) {
			row = append(row, areas[i])
			i++
		} else {
			lay(row)
			row = nil
		}
	}
	if len(row) > 0 {
		lay(row)
	}
	return out
}

func (rd *renderer) treemap(p string) string {
	th := rd.th
	if len(rd.rows) == 0 {
		return rd.emptyLanguages("LANGUAGES · AREA")
	}
	cols := rd.langColors()
	var s strings.Builder
	s.WriteString(rd.langLabel("LANGUAGES · AREA"))
	vals := make([]float64, len(rd.rows))
	for i, r := range rd.rows {
		vals[i] = r.v
	}
	for i, q := range squarify(vals, 24, 36, width-48, 206) {
		x, y, w, h := q[0], q[1], q[2], q[3]
		r := rd.rows[i]
		d := .15 + float64(i)*.09
		c := cols[i]
		var g strings.Builder
		fmt.Fprintf(&g, `<rect x="%s" y="%s" width="%s" height="%s" rx="4" fill="%s" fill-opacity=".14" stroke="%s" stroke-opacity=".55"/>`,
			num(x+1.5), num(y+1.5), num(w-3), num(h-3), c, c)
		fmt.Fprintf(&g, `<rect x="%s" y="%s" width="%s" height="2" fill="%s"/>`, num(x+1.5), num(y+1.5), num(w-3), c)
		switch {
		case w > 64 && h > 34:
			g.WriteString(txt(x+10, y+22, esc(r.name), topt{fill: rd.nameFill(r), size: 12, weight: 500}))
			content := pct(r.v)
			if r.mult != "" {
				content += fmt.Sprintf(`  <tspan fill="%s">%s</tspan>`, th.accent, r.mult)
			}
			g.WriteString(txt(x+10, y+38, content, topt{fill: th.muted, size: 10, mono: true}))
		case w > 30 && h > 18:
			g.WriteString(txt(x+w/2, y+h/2+4, fmt.Sprintf("%.0f%%", r.v), topt{fill: th.muted, size: 10, mono: true, anchor: "middle"}))
		}
		fmt.Fprintf(&s, `<g%s>%s</g>`, anim("fade", d), g.String())
	}
	return s.String()
}

func (rd *renderer) shiftUseful() bool {
	for _, r := range rd.rows {
		if math.Abs(r.raw-r.v) >= .05 {
			return true
		}
	}
	return false
}

func rank(arr []float64) []int {
	idx := make([]int, len(arr))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int {
		switch {
		case arr[a] > arr[b]:
			return -1
		case arr[a] < arr[b]:
			return 1
		}
		return a - b
	})
	return idx
}

func (rd *renderer) shift(p string) string {
	th := rd.th
	if len(rd.rows) == 0 {
		return rd.emptyLanguages("LANGUAGES · RAW → ADJUSTED")
	}
	cols := rd.langColors()
	var s strings.Builder
	s.WriteString(label(22, "LANGUAGES · RAW → ADJUSTED", th, ""))
	const xl, xr, top, step = 168.0, 327.0, 52.0, 29.0
	raw := make([]float64, len(rd.rows))
	adj := make([]float64, len(rd.rows))
	for i, r := range rd.rows {
		raw[i], adj[i] = r.raw, r.v
	}
	rawOrder, adjOrder := rank(raw), rank(adj)
	yOf := func(order []int, i int) float64 { return top + float64(slices.Index(order, i))*step }
	s.WriteString(txt(xl, 38, "RAW", topt{fill: th.muted, size: 10, mono: true, anchor: "middle", ls: 1.2}))
	s.WriteString(txt(xr, 38, "ADJUSTED", topt{fill: th.muted, size: 10, mono: true, anchor: "middle", ls: 1.2}))
	for i, r := range rd.rows {
		y1, y2 := yOf(rawOrder, i), yOf(adjOrder, i)
		d := .2 + float64(slices.Index(adjOrder, i))*.1
		moved := y1 != y2
		c := cols[i]
		opacity := ".4"
		switch {
		case r.mult != "":
			c, opacity = acc, "1"
		case moved:
			opacity = ".7"
		}
		fmt.Fprintf(&s, `<path%s d="M%s %s C%s %s %s %s %s %s" pathLength="1" fill="none" stroke="%s" stroke-opacity="%s" stroke-width="%.1f" stroke-linecap="round"/>`,
			anim("draw", d), num(xl+4), num(y1-4), num(xl+80), num(y1-4), num(xr-80), num(y2-4), num(xr-4), num(y2-4), c, opacity, math.Min(1+r.v/8, 4))
		fmt.Fprintf(&s, `<circle%s cx="%s" cy="%s" r="2.5" fill="%s"/>`, anim("fade", d), num(xl), num(y1-4), c)
		fmt.Fprintf(&s, `<circle%s cx="%s" cy="%s" r="2.5" fill="%s"/>`, anim("fade", d+.5), num(xr), num(y2-4), c)
		s.WriteString(txt(xl-12, y1, fmt.Sprintf(`%s <tspan fill="%s" font-family="%s" font-size="10">%.1f</tspan>`, esc(r.name), th.muted, monoFont, r.raw),
			topt{fill: rd.nameFill(r), anchor: "end", cls: "fade", delay: d}))
		right := fmt.Sprintf(`<tspan font-family="%s" font-size="10" fill="%s">%.1f</tspan> %s`, monoFont, th.muted, r.v, esc(r.name))
		if r.mult != "" {
			right += fmt.Sprintf(` <tspan fill="%s" font-family="%s" font-size="10">%s</tspan>`, th.accent, monoFont, r.mult)
		}
		s.WriteString(txt(xr+12, y2, right, topt{fill: rd.nameFill(r), cls: "fade", delay: d + .5}))
	}
	return s.String()
}

func (rd *renderer) terminal(p string) string {
	th := rd.th
	if len(rd.rows) == 0 {
		return rd.emptyLanguages("LANGUAGES · TERMINAL")
	}
	cols := rd.langColors()
	var s strings.Builder
	command := "ghstats languages"
	if rd.cfg.Languages.ShowWeighting() {
		command += " --" + esc(rd.rep.Languages.Mode)
	}
	s.WriteString(txt(24, 22, fmt.Sprintf(`<tspan fill="%s">$</tspan> %s`, acc, command),
		topt{fill: th.fg, size: 12, mono: true, cls: "on", delay: .1}))
	for i, r := range rd.rows {
		y := float64(52 + i*24)
		d := .45 + float64(i)*.22
		blocks := int(jsRound(r.v / rd.maxV * 22))
		bar := strings.Repeat("█", blocks) + fmt.Sprintf(`<tspan fill="%s">%s</tspan>`, th.faint, strings.Repeat("░", 22-blocks))
		s.WriteString(txt(24, y, esc(r.name), topt{fill: rd.nameFill(r), size: 12, mono: true, cls: "on", delay: d}))
		s.WriteString(txt(116, y, bar, topt{fill: cols[i], size: 12, mono: true, cls: "on", delay: d + .08}))
		content := pct(r.v)
		if r.mult != "" {
			content += fmt.Sprintf(` <tspan fill="%s">%s</tspan>`, th.accent, r.mult)
		}
		s.WriteString(txt(width-24, y, content, topt{fill: th.muted, size: 11, mono: true, anchor: "end", cls: "on", delay: d + .12}))
	}
	s.WriteString(txt(24, 236, fmt.Sprintf(`<tspan fill="%s">$</tspan>`, acc), topt{fill: th.fg, size: 12, mono: true, cls: "on", delay: 2.1}))
	fmt.Fprintf(&s, `<rect class="blink" x="36" y="226" width="7" height="13" fill="%s"/>`, th.fg)
	return s.String()
}

func (rd *renderer) cells(p string) string {
	th := rd.th
	if len(rd.rows) == 0 {
		return rd.emptyLanguages("LANGUAGES · CELLS")
	}
	cols := rd.langColors()
	var s strings.Builder
	s.WriteString(rd.langLabel("LANGUAGES · CELLS"))
	for i, r := range rd.rows {
		y := float64(54 + i*29)
		d := .2 + float64(i)*.07
		on := int(math.Max(1, jsRound(r.v/rd.maxV*16)))
		s.WriteString(txt(24, y, esc(r.name), topt{fill: rd.nameFill(r), cls: "fade", delay: d}))
		for c := range 20 {
			fill, extra := th.faint, ""
			if c < on {
				fill, extra = cols[i], anim("on", .4+float64(i)*.08+float64(c)*.035)
			}
			fmt.Fprintf(&s, `<rect%s x="%d" y="%s" width="8" height="16" rx="1.5" fill="%s"/>`, extra, 112+c*10, num(y-13), fill)
		}
		if r.mult != "" {
			s.WriteString(txt(320, y, r.mult, topt{fill: acc, size: 10, mono: true, cls: "fade", delay: d + .6}))
		}
		s.WriteString(txt(width-24, y, pct(r.v), topt{fill: th.muted, size: 11, mono: true, anchor: "end", cls: "fade", delay: d}))
	}
	return s.String()
}

func (rd *renderer) iso(p string) string {
	th := rd.th
	if len(rd.rows) == 0 {
		return rd.emptyLanguages("LANGUAGES · STACK")
	}
	cols := rd.langColors()
	var s strings.Builder
	s.WriteString(rd.langLabel("LANGUAGES · STACK"))
	const cx, cy, dx, dy, total, gap = 104.0, 196.0, 58.0, 33.0, 124.0, 2.0
	type slab struct{ z0, z1 float64 }
	slabs := make([]slab, len(rd.rows))
	z := 0.0
	for i, r := range rd.rows {
		hh := total * r.v / 100
		slabs[i] = slab{z, z + hh - gap}
		z += hh
	}
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	join := func(pts ...string) string { return strings.Join(pts, " ") }
	for i, sl := range slabs {
		c := cols[i]
		top := join(pt(cx, cy-dy-sl.z1), pt(cx+dx, cy-sl.z1), pt(cx, cy+dy-sl.z1), pt(cx-dx, cy-sl.z1))
		left := join(pt(cx-dx, cy-sl.z1), pt(cx, cy+dy-sl.z1), pt(cx, cy+dy-sl.z0), pt(cx-dx, cy-sl.z0))
		right := join(pt(cx, cy+dy-sl.z1), pt(cx+dx, cy-sl.z1), pt(cx+dx, cy-sl.z0), pt(cx, cy+dy-sl.z0))
		face := func(pts, shade string, op float64) string {
			return fmt.Sprintf(`<polygon points="%s" fill="%s"/><polygon points="%s" fill="%s" fill-opacity="%g"/>`, pts, c, pts, shade, op)
		}
		fmt.Fprintf(&s, `<g%s>%s%s%s</g>`, anim("drop", .15+float64(i)*.12),
			face(left, "#000000", .12), face(right, "#000000", .32), face(top, "#ffffff", .14))
	}
	const lx = 248.0
	for k := range rd.rows {
		i := len(rd.rows) - 1 - k
		r, sl := rd.rows[i], slabs[i]
		ly := float64(48 + k*20)
		ay := cy - (sl.z0+sl.z1)/2
		d := .5 + float64(i)*.12
		fmt.Fprintf(&s, `<polyline%s points="%s" fill="none" stroke="%s"/>`,
			anim("fade", d), join(pt(cx+dx+4, ay), pt(lx-26, ay), pt(lx-10, ly-4)), th.border)
		fmt.Fprintf(&s, `<circle%s cx="%s" cy="%s" r="3" fill="%s"/>`, anim("fade", d), num(lx-4), num(ly-4), cols[i])
		name := esc(r.name)
		if r.mult != "" {
			name += fmt.Sprintf(` <tspan fill="%s" font-family="%s" font-size="10">%s</tspan>`, th.accent, monoFont, r.mult)
		}
		s.WriteString(txt(lx+6, ly, name, topt{fill: rd.nameFill(r), cls: "fade", delay: d}))
		s.WriteString(txt(width-24, ly, pct(r.v), topt{fill: th.muted, size: 11, mono: true, anchor: "end", cls: "fade", delay: d}))
	}
	return s.String()
}

type plot struct {
	label string
	data  []int64
}

func shortDate(day string) string {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return day
	}
	return strings.ToUpper(t.Format("2 Jan"))
}

func firstNonZero(vals []int64) int {
	for i, v := range vals {
		if v > 0 {
			return i
		}
	}
	return -1
}

func (rd *renderer) trend(channel, name string) *plot {
	for _, t := range rd.rep.Trends {
		if t.Channel != channel {
			continue
		}
		weeks := make([]int64, len(t.Weeks))
		for i, w := range t.Weeks {
			weeks[i] = w.Count
		}
		fw := firstNonZero(weeks)
		if fw < 0 {
			return nil
		}
		if len(weeks)-fw >= minWeeks {
			label := fmt.Sprintf("%s · %d WEEKS", name, len(weeks))
			if fw > 0 {
				label = name + " · SINCE " + shortDate(t.Weeks[fw].Start)
			}
			return &plot{label, weeks[fw:]}
		}
		days := make([]int64, len(t.Days))
		for i, d := range t.Days {
			days[i] = d.Count
		}
		start := firstNonZero(days)
		if start < 0 || len(days)-start < 2 {
			start = len(days) - 2
		}
		if start < 0 {
			return nil
		}
		return &plot{name + " · DAILY · SINCE " + shortDate(t.Days[start].Day), days[start:]}
	}
	return nil
}

func (rd *renderer) history(p, channel, name string) string {
	th := rd.th
	var s strings.Builder
	pl := rd.trend(channel, name)
	title := name + " · 26 WEEKS"
	if pl != nil {
		title = pl.label
	}
	s.WriteString(label(22, title, th, ""))
	s.WriteString(txt(width-24, 22, fmt.Sprintf(`%s <tspan fill="%s">last 14d</tspan>`, esc(rd.metric(channel, report.WindowLast14Days)), th.muted),
		topt{fill: th.fg, size: 11, mono: true, anchor: "end"}))
	if pl == nil {
		s.WriteString(txt(width/2, 68, "no traffic data yet", topt{fill: th.muted, size: 11, mono: true, anchor: "middle"}))
		return s.String()
	}
	data := pl.data
	const x0, x1, y0, y1 = 24.0, width - 24, 38.0, 92.0
	var vmax int64
	for _, v := range data {
		vmax = max(vmax, v)
	}
	pts := make([][2]float64, len(data))
	var line strings.Builder
	for i, v := range data {
		pts[i] = [2]float64{x0 + (x1-x0)*float64(i)/float64(len(data)-1), y1 - (y1-y0)*float64(v)/float64(vmax)}
		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		fmt.Fprintf(&line, "%s%.1f %.1f ", cmd, pts[i][0], pts[i][1])
	}
	d := strings.TrimSpace(line.String())
	gid := p + channel
	fmt.Fprintf(&s, `<defs><linearGradient id="%s" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="%s" stop-opacity=".22"/><stop offset="1" stop-color="%s" stop-opacity="0"/></linearGradient></defs>`, gid, acc, acc)
	fmt.Fprintf(&s, `<path%s d="%s L%s %s L%s %s Z" fill="url(#%s)"/>`, anim("fadeslow", 1.1), d, num(x1), num(y1), num(x0), num(y1), gid)
	fmt.Fprintf(&s, `<path%s d="%s" pathLength="1" fill="none" stroke="%s" stroke-width="1.6" stroke-linejoin="round" stroke-linecap="round"/>`, anim("draw", .4), d, acc)
	e := pts[len(pts)-1]
	fmt.Fprintf(&s, `<circle%s cx="%s" cy="%s" r="3" fill="%s"/>`, anim("fade", 2.1), num(e[0]), num(e[1]), acc)
	return s.String()
}
