package render

import (
	"regexp"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
)

func TestSnippetLinkAnchorsHaveNoWhitespace(t *testing.T) {
	cfg, err := config.Parse(strings.NewReader(`version: 1
profile:
  tagline: T
  links:
    - youtube: "@a"
    - website: https://example.com
`))
	if err != nil {
		t.Fatal(err)
	}
	s := Snippet(cfg)
	anchors := regexp.MustCompile(`(?s)<a href=.*?</a>`).FindAllString(s, -1)
	if len(anchors) != 2 {
		t.Fatalf("want 2 anchors, got %d", len(anchors))
	}
	for _, a := range anchors {
		if strings.ContainsAny(a, "\n\t") || strings.Contains(a, "> <") {
			t.Errorf("anchor contains whitespace that GitHub renders as an underlined link: %q", a)
		}
	}
}
