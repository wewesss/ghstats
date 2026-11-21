package png_test

import (
	"bytes"
	"context"
	"image/png"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/wewesss/ghstats/config"
	ghpng "github.com/wewesss/ghstats/png"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/sample"
)

var viewBox = regexp.MustCompile(`viewBox="0 0 (\d+) (\d+)"`)

var magic = []byte("\x89PNG\r\n\x1a\n")

const tiny = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 20"><rect width="40" height="20" fill="#123456"/><text x="2" y="14" font-size="10" fill="#fff">ab</text></svg>`

func decode(t *testing.T, data []byte) (w, h int) {
	t.Helper()
	if !bytes.HasPrefix(data, magic) {
		t.Fatalf("missing PNG header: % x", data[:min(8, len(data))])
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	b := img.Bounds()
	return b.Dx(), b.Dy()
}

func TestRenderSmallSVG(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		data, err := ghpng.Render(context.Background(), []byte(tiny), scale)
		if err != nil {
			t.Fatal(err)
		}
		w, h := decode(t, data)
		if w != int(40*scale) || h != int(20*scale) {
			t.Fatalf("scale %v: got %dx%d", scale, w, h)
		}
	}
}

func TestRenderCards(t *testing.T) {
	cfg := config.Default()
	cfg.Render = sample.FullRender()
	cfg.Render.Motion = false
	cfg.Render.Blocks = sample.AllBlocks()
	cards, err := render.Cards(sample.Report(), cfg, render.Dark)
	if err != nil {
		t.Fatal(err)
	}
	svg, ok := cards["stats-dark.svg"]
	if !ok {
		t.Fatalf("no stats-dark.svg in %d cards", len(cards))
	}
	data, err := ghpng.Render(context.Background(), []byte(svg), 2)
	if err != nil {
		t.Fatal(err)
	}
	w, h := decode(t, data)
	m := viewBox.FindStringSubmatch(svg)
	if m == nil {
		t.Fatal("no viewBox")
	}
	vw, _ := strconv.Atoi(m[1])
	vh, _ := strconv.Atoi(m[2])
	if w != vw*2 || h != vh*2 {
		t.Fatalf("got %dx%d, want %dx%d", w, h, vw*2, vh*2)
	}
}

func TestRenderInvalidSVG(t *testing.T) {
	for _, in := range []string{"", "not an svg", "<svg"} {
		if _, err := ghpng.Render(context.Background(), []byte(in), 2); err == nil {
			t.Fatalf("%q: want an error", in)
		}
	}
	if _, err := ghpng.Render(context.Background(), []byte(tiny), 0); err == nil {
		t.Fatal("scale 0: want an error")
	}
}

func TestRenderRecoversAfterError(t *testing.T) {
	if _, err := ghpng.Render(context.Background(), []byte("nope"), 1); err == nil {
		t.Fatal("want an error")
	}
	if _, err := ghpng.Render(context.Background(), []byte(tiny), 1); err != nil {
		t.Fatal(err)
	}
}

func TestRenderConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := ghpng.Render(context.Background(), []byte(strings.Clone(tiny)), 1)
			if err != nil || !bytes.HasPrefix(data, magic) {
				t.Errorf("render: %v", err)
			}
		}()
	}
	wg.Wait()
}
