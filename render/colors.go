package render

import (
	"hash/fnv"
	"math"
	"strings"
)

var linguist = map[string]string{
	"Ada":               "#02f88c",
	"Agda":              "#315665",
	"Apex":              "#1797c0",
	"Arduino":           "#bd79d1",
	"Assembly":          "#6e4c13",
	"Astro":             "#ff5a03",
	"Awk":               "#c30e9b",
	"Batchfile":         "#c1f12e",
	"C":                 "#555555",
	"C#":                "#178600",
	"C++":               "#f34b7d",
	"CMake":             "#da3434",
	"CSS":               "#663399",
	"Clojure":           "#db5855",
	"CoffeeScript":      "#244776",
	"Common Lisp":       "#3fb68b",
	"Crystal":           "#000100",
	"Cuda":              "#3a4e3a",
	"D":                 "#ba595e",
	"Dart":              "#00b4ab",
	"Dockerfile":        "#384d54",
	"EJS":               "#a91e50",
	"Elixir":            "#6e4a7e",
	"Elm":               "#60b5cc",
	"Emacs Lisp":        "#c065db",
	"Erlang":            "#b83998",
	"F#":                "#b845fc",
	"Fortran":           "#4d41b1",
	"GDScript":          "#355570",
	"GLSL":              "#5686a5",
	"Gherkin":           "#5b2063",
	"Gleam":             "#ffaff3",
	"Go":                "#00add8",
	"Groovy":            "#4298b8",
	"HCL":               "#844fba",
	"HLSL":              "#aace60",
	"HTML":              "#e34c26",
	"Hack":              "#878787",
	"Handlebars":        "#f7931e",
	"Haskell":           "#5e5086",
	"Java":              "#b07219",
	"JavaScript":        "#f1e05a",
	"Jinja":             "#a52a22",
	"Julia":             "#a270ba",
	"Jupyter Notebook":  "#da5b0b",
	"Kotlin":            "#a97bff",
	"Less":              "#1d365d",
	"Liquid":            "#67b8de",
	"Lua":               "#000080",
	"MATLAB":            "#e16737",
	"Makefile":          "#427819",
	"Markdown":          "#083fa1",
	"Mathematica":       "#dd1100",
	"Nim":               "#ffc200",
	"Nix":               "#7e7eff",
	"OCaml":             "#ef7a08",
	"Objective-C":       "#438eff",
	"Objective-C++":     "#6866fb",
	"Odin":              "#60affe",
	"PHP":               "#4f5d95",
	"PLpgSQL":           "#336790",
	"Pascal":            "#e3f171",
	"Perl":              "#0298c3",
	"PowerShell":        "#012456",
	"Prolog":            "#74283c",
	"Pug":               "#a86454",
	"Puppet":            "#302b6d",
	"Python":            "#3572a5",
	"R":                 "#198ce7",
	"Racket":            "#3c5caa",
	"Raku":              "#0000fb",
	"Ruby":              "#701516",
	"Rust":              "#dea584",
	"SCSS":              "#c6538c",
	"Sass":              "#a53b70",
	"Scala":             "#c22d40",
	"Scheme":            "#1e4aec",
	"Shell":             "#89e051",
	"Solidity":          "#aa6746",
	"Starlark":          "#76d275",
	"Svelte":            "#ff3e00",
	"Swift":             "#f05138",
	"TeX":               "#3d6117",
	"Tcl":               "#e4cc98",
	"TypeScript":        "#3178c6",
	"V":                 "#4f87c4",
	"VHDL":              "#adb2cb",
	"Vala":              "#a56de2",
	"Verilog":           "#b2b7f8",
	"Vim Script":        "#199f4b",
	"Visual Basic .NET": "#945db7",
	"Vue":               "#41b883",
	"WebAssembly":       "#04133b",
	"Zig":               "#ec915c",
}

var linguistFolded = func() map[string]string {
	out := make(map[string]string, len(linguist))
	for k, v := range linguist {
		out[strings.ToLower(k)] = v
	}
	return out
}()

func githubColor(name string) string {
	if c, ok := linguistFolded[strings.ToLower(name)]; ok {
		return c
	}
	h := fnv.New32a()
	h.Write([]byte(strings.ToLower(name)))
	return hslHex(float64(h.Sum32()%360), .55, .55)
}

func hslHex(h, s, l float64) string {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	const digits = "0123456789abcdef"
	out := []byte{'#'}
	for _, v := range []float64{r + m, g + m, b + m} {
		n := int(jsRound(v * 255))
		out = append(out, digits[n>>4&15], digits[n&15])
	}
	return string(out)
}
