//go:build js && wasm

// Command wasm exposes the config parser and the SVG renderer to the browser builder.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/sample"
)

func main() {
	api := map[string]any{
		"render":   js.FuncOf(renderFunc),
		"defaults": js.FuncOf(defaultsFunc),
		"options":  js.FuncOf(optionsFunc),
	}
	js.Global().Set("ghstats", js.ValueOf(api))
	select {}
}

func failure(msg string) any {
	return map[string]any{
		"cards":   map[string]any{},
		"snippet": "",
		"yaml":    "",
		"error":   msg,
	}
}

func renderFunc(_ js.Value, args []js.Value) any {
	if len(args) != 2 {
		return failure("render: want (yaml, mode)")
	}
	mode := render.Mode(args[1].String())
	if mode != render.Dark && mode != render.Light {
		return failure(fmt.Sprintf("render: mode must be %q or %q, got %q", render.Dark, render.Light, mode))
	}
	cfg, err := config.Parse(strings.NewReader(args[0].String()))
	if err != nil {
		return failure(err.Error())
	}
	cards, err := render.Cards(sample.Report(), cfg, mode)
	if err != nil {
		return failure(err.Error())
	}
	var normalized bytes.Buffer
	if err := config.Encode(&normalized, cfg); err != nil {
		return failure(err.Error())
	}
	out := make(map[string]any, len(cards))
	for name, svg := range cards {
		out[name] = svg
	}
	return map[string]any{
		"cards":   out,
		"snippet": render.Snippet(cfg),
		"yaml":    normalized.String(),
		"error":   "",
	}
}

func defaultsFunc(js.Value, []js.Value) any {
	var buf bytes.Buffer
	if err := config.Encode(&buf, sample.Profile()); err != nil {
		return ""
	}
	return buf.String()
}

type blockOption struct {
	Name     string   `json:"name"`
	Variants []string `json:"variants"`
}

type linkOption struct {
	Kind    string `json:"kind"`
	URLOnly bool   `json:"url_only"`
	MapOnly bool   `json:"map_only"`
}

type options struct {
	Blocks     []blockOption `json:"blocks"`
	Presets    []string      `json:"presets"`
	Accents    []string      `json:"accents"`
	Palettes   []string      `json:"palettes"`
	Layouts    []string      `json:"layouts"`
	LinkKinds  []linkOption  `json:"link_kinds"`
	LinkStyles []string      `json:"link_styles"`
	Limits     struct {
		Focus    int `json:"focus"`
		Building int `json:"building"`
	} `json:"limits"`
}

func optionsFunc(js.Value, []js.Value) any {
	o := options{
		Presets:    config.Presets(),
		Accents:    config.Accents(),
		Palettes:   config.Palettes(),
		Layouts:    config.Layouts(),
		LinkStyles: config.LinkStyles(),
	}
	for _, b := range config.BlockVariants() {
		o.Blocks = append(o.Blocks, blockOption{Name: b.Name, Variants: b.Variants})
	}
	for _, k := range config.LinkKinds() {
		o.LinkKinds = append(o.LinkKinds, linkOption{Kind: k, URLOnly: config.LinkKindIsURLOnly(k), MapOnly: config.LinkKindIsMapOnly(k)})
	}
	o.Limits.Focus, o.Limits.Building = config.FocusLimits()
	data, err := json.Marshal(o)
	if err != nil {
		return ""
	}
	return string(data)
}
