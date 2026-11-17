package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/wewesss/ghstats/config"
)

type Mode string

const (
	Dark  Mode = "dark"
	Light Mode = "light"
)

type colors struct {
	bg, border, fg, muted, faint, grid string
}

type theme struct {
	colors
	accent string
}

var accentOrder = []string{"ice", "mint", "amber", "silver", "violet", "rose", "lime", "coral"}

var accentTable = map[string][2]string{
	"ice":    {"#9ec5ff", "#2457c5"},
	"mint":   {"#7ee0b8", "#0b7a55"},
	"amber":  {"#f5c46b", "#a85a07"},
	"silver": {"#d6dae0", "#3b424b"},
	"violet": {"#c3a6ff", "#6d3fd1"},
	"rose":   {"#ff9ec4", "#b8285f"},
	"lime":   {"#c6ee6b", "#4d7a0c"},
	"coral":  {"#ff9f87", "#b5412a"},
}

var themeTable = map[string]map[Mode]colors{
	"carbon": {
		Dark:  {bg: "#000000", border: "#1a1a1a", fg: "#f2f2f2", muted: "#7a7a7a", faint: "#1c1c1c", grid: "#ffffff"},
		Light: {bg: "#fafafa", border: "#e5e5e5", fg: "#0a0a0a", muted: "#737373", faint: "#ececec", grid: "#000000"},
	},
	"midnight": {
		Dark:  {bg: "#0b1020", border: "#1c2440", fg: "#e3e8f5", muted: "#7c87a6", faint: "#1a2238", grid: "#ffffff"},
		Light: {bg: "#f5f7fc", border: "#dde3f0", fg: "#101830", muted: "#5d6a8a", faint: "#e6ebf5", grid: "#000000"},
	},
	"slate": {
		Dark:  {bg: "#121a1f", border: "#23313a", fg: "#e4ecef", muted: "#82939c", faint: "#1f2b33", grid: "#ffffff"},
		Light: {bg: "#f3f6f7", border: "#d9e2e6", fg: "#13202a", muted: "#5c6e78", faint: "#e3eaed", grid: "#000000"},
	},
	"dune": {
		Dark:  {bg: "#16130f", border: "#2c261e", fg: "#ece6dc", muted: "#948a7c", faint: "#2a241c", grid: "#ffffff"},
		Light: {bg: "#fbf8f3", border: "#e8e0d4", fg: "#1d1914", muted: "#7a7062", faint: "#efe8dd", grid: "#000000"},
	},
	"noir": {
		Dark:  {bg: "#0b0c0f", border: "#1f2329", fg: "#e6e8eb", muted: "#7b838d", faint: "#22262d", grid: "#ffffff"},
		Light: {bg: "#ffffff", border: "#e3e6ea", fg: "#101317", muted: "#6a727c", faint: "#edf0f2", grid: "#000000"},
	},
	"graphite": {
		Dark:  {bg: "#15171b", border: "#2a2e35", fg: "#e9ebee", muted: "#8a929c", faint: "#2a2e35", grid: "#ffffff"},
		Light: {bg: "#f6f7f9", border: "#dfe3e8", fg: "#15181c", muted: "#646c76", faint: "#e6e9ed", grid: "#000000"},
	},
	"transparent": {
		Dark:  {bg: "none", border: "#30363d", fg: "#e6edf3", muted: "#8b949e", faint: "#21262d", grid: "#ffffff"},
		Light: {bg: "none", border: "#d0d7de", fg: "#1f2328", muted: "#656d76", faint: "#eaeef2", grid: "#000000"},
	},
}

var spectrum = map[Mode][]string{
	Dark:  {"#8ab4ff", "#6fd6c5", "#f1c56e", "#f4927f", "#c4a2ff", "#9fd479"},
	Light: {"#2f62c9", "#0f8a77", "#a96d06", "#c2452d", "#7446d0", "#4f8a1c"},
}

var grays = map[Mode][]string{
	Dark:  {"#e6e8eb", "#c8cdd4", "#a3a9b2", "#7f868f", "#5f656e", "#454a52"},
	Light: {"#1b1f24", "#2b3036", "#4a5058", "#6b727b", "#8d939b", "#b0b5bb"},
}

var (
	ground     = map[Mode]string{Dark: "#0d1117", Light: "#ffffff"}
	otherColor = map[Mode]string{Dark: "#4a4f57", Light: "#b8bdc3"}
)

func accentFor(a config.Accent, mode Mode) string {
	idx := 0
	if mode == Light {
		idx = 1
	}
	switch {
	case a.Name == "" && mode == Dark:
		return strings.ToLower(a.Dark)
	case a.Name == "":
		return strings.ToLower(a.Light)
	case a.Name == config.AccentCycle:
		return accentTable["ice"][idx]
	}
	return accentTable[a.Name][idx]
}

func newTheme(t config.Theme, mode Mode) theme {
	th := theme{colors: themeTable[t.Preset][mode], accent: accentFor(t.Accent, mode)}
	for key, v := range t.Overrides {
		v = strings.ToLower(v)
		switch key {
		case "bg":
			th.bg = v
		case "border":
			th.border = v
		case "fg":
			th.fg = v
		case "muted":
			th.muted = v
		case "faint":
			th.faint = v
		}
	}
	return th
}

func jsRound(v float64) float64 { return math.Floor(v + .5) }

func hex2rgb(h string) [3]float64 {
	var out [3]float64
	for i := range out {
		n, _ := strconv.ParseUint(h[1+2*i:3+2*i], 16, 8)
		out[i] = float64(n)
	}
	return out
}

func mixTo(h string, to [3]float64, t float64) string {
	c := hex2rgb(h)
	var sb strings.Builder
	sb.WriteByte('#')
	for i := range c {
		v := int(jsRound(c[i] + (to[i]-c[i])*t))
		fmt.Fprintf(&sb, "%02x", min(max(v, 0), 255))
	}
	return sb.String()
}

func lum(h string) float64 {
	c := hex2rgb(h)
	for i, v := range c {
		v /= 255
		if v <= .03928 {
			c[i] = v / 12.92
		} else {
			c[i] = math.Pow((v+.055)/1.055, 2.4)
		}
	}
	return .2126*c[0] + .7152*c[1] + .0722*c[2]
}

func contrast(a, b string) float64 {
	x, y := lum(a), lum(b)
	if y > x {
		x, y = y, x
	}
	return (x + .05) / (y + .05)
}

func readable(c, ground string) string {
	to := [3]float64{0, 0, 0}
	if lum(ground) < .5 {
		to = [3]float64{255, 255, 255}
	}
	out, k := c, 0.0
	for contrast(out, ground) < 3 && k < 1 {
		k += .05
		out = mixTo(c, to, k)
	}
	return out
}
