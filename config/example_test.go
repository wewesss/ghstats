package config_test

import (
	"fmt"
	"strings"

	"github.com/wewesss/ghstats/config"
)

func ExampleParse() {
	cfg, err := config.Parse(strings.NewReader("version: 1\nlanguages:\n  max_visible: 3\n"))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(cfg.Version, cfg.Languages.MaxVisible)
	// Output: 1 3
}

func ExampleDefault() {
	cfg := config.Default()
	fmt.Println(cfg.Languages.Mode, cfg.Languages.MaxVisible, cfg.Scope.IncludePrivate)
	// Output: raw 6 true
}
