// Package png rasterizes SVG cards to PNG with resvg compiled to WebAssembly and run by wazero.
package png

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed resvg.wasm
var resvgWasm []byte

type engine struct {
	mu       sync.Mutex
	mod      api.Module
	alloc    api.Function
	dealloc  api.Function
	render   api.Function
	resPtr   api.Function
	resLen   api.Function
	compiled wazero.CompiledModule
	runtime  wazero.Runtime
}

var (
	once    sync.Once
	shared  *engine
	loadErr error
)

func load() (*engine, error) {
	once.Do(func() {
		ctx := context.Background()
		rt := wazero.NewRuntime(ctx)
		compiled, err := rt.CompileModule(ctx, resvgWasm)
		if err != nil {
			loadErr = fmt.Errorf("png: compile resvg: %w", err)
			return
		}
		mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(""))
		if err != nil {
			loadErr = fmt.Errorf("png: instantiate resvg: %w", err)
			return
		}
		e := &engine{runtime: rt, compiled: compiled, mod: mod}
		for name, dst := range map[string]*api.Function{
			"alloc": &e.alloc, "dealloc": &e.dealloc, "render": &e.render,
			"result_ptr": &e.resPtr, "result_len": &e.resLen,
		} {
			if *dst = mod.ExportedFunction(name); *dst == nil {
				loadErr = fmt.Errorf("png: resvg module does not export %q", name)
				return
			}
		}
		shared = e
	})
	return shared, loadErr
}

// Render rasterizes svg to a PNG whose size is the SVG size multiplied by scale.
func Render(ctx context.Context, svg []byte, scale float64) ([]byte, error) {
	if len(svg) == 0 {
		return nil, errors.New("png: empty svg")
	}
	if !(scale > 0) || math.IsInf(scale, 0) {
		return nil, fmt.Errorf("png: invalid scale %v", scale)
	}
	e, err := load()
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	res, err := e.alloc.Call(ctx, uint64(len(svg)))
	if err != nil {
		return nil, fmt.Errorf("png: alloc: %w", err)
	}
	ptr := uint32(res[0])
	defer e.dealloc.Call(ctx, uint64(ptr), uint64(len(svg)))
	if !e.mod.Memory().Write(ptr, svg) {
		return nil, errors.New("png: write svg out of memory range")
	}
	res, err = e.render.Call(ctx, uint64(ptr), uint64(len(svg)), api.EncodeF32(float32(scale)))
	if err != nil {
		return nil, fmt.Errorf("png: render: %w", err)
	}
	code := api.DecodeI32(res[0])
	out, err := e.result(ctx)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, fmt.Errorf("png: %s", out)
	}
	return out, nil
}

func (e *engine) result(ctx context.Context) ([]byte, error) {
	p, err := e.resPtr.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("png: result_ptr: %w", err)
	}
	n, err := e.resLen.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("png: result_len: %w", err)
	}
	b, ok := e.mod.Memory().Read(uint32(p[0]), uint32(n[0]))
	if !ok {
		return nil, errors.New("png: read result out of memory range")
	}
	return append([]byte(nil), b...), nil
}
