package render_test

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/sample"
)

func ExampleCards() {
	cards, err := render.Cards(sample.Report(), config.Default(), render.Dark)
	if err != nil {
		fmt.Println(err)
		return
	}
	names := make([]string, 0, len(cards))
	for name := range cards {
		names = append(names, name)
	}
	slices.Sort(names)
	fmt.Println(names)
	// Output: [stats-dark.svg]
}

func ExampleSnippet() {
	fmt.Print(render.Snippet(config.Default()))
	// Output:
	// <picture>
	//   <source media="(prefers-color-scheme: dark)" srcset="ghstats/stats-dark.svg">
	//   <source media="(prefers-color-scheme: light)" srcset="ghstats/stats-light.svg">
	//   <img alt="GitHub stats" src="ghstats/stats-dark.svg" width="495">
	// </picture>
}

func ExampleJSON() {
	var buf bytes.Buffer
	if err := render.JSON(&buf, sample.Report()); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(bytes.Contains(buf.Bytes(), []byte(`"schema_version": 1`)))
	// Output: true
}
