package render_test

import (
	"fmt"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
)

func ExampleSnippet() {
	fmt.Print(render.Snippet(config.Default()))
	// Output:
	// <picture>
	//   <source media="(prefers-color-scheme: dark)" srcset="ghstats/stats-dark.svg">
	//   <source media="(prefers-color-scheme: light)" srcset="ghstats/stats-light.svg">
	//   <img alt="GitHub stats" src="ghstats/stats-dark.svg" width="495">
	// </picture>
}
