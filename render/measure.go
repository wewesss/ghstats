package render

import (
	"math"
	"strings"
	"unicode/utf8"
)

const (
	sansAdvance = 0.56
	sansSpace   = 0.28
	ellipsis    = "…"
)

func sansWidth(s string, size float64) float64 {
	spaces := strings.Count(s, " ")
	letters := utf8.RuneCountInString(s) - spaces
	return (sansAdvance*float64(letters) + sansSpace*float64(spaces)) * size
}

func statsSlashX(login string) float64 {
	return math.Round(24 + sansWidth(login, 14) + 7)
}

func monoCols(width, size, ls float64) int {
	if width <= 0 {
		return 0
	}
	return max(1, int(math.Floor(width/(monoAdvance*size+ls))))
}

func clipRunes(s string, cols int) string {
	r := []rune(s)
	if len(r) <= cols {
		return s
	}
	if cols <= 1 {
		return ellipsis
	}
	return string(r[:cols-1]) + ellipsis
}

func clipMono(s string, width, size, ls float64) string {
	cols := monoCols(width, size, ls)
	if cols == 0 {
		return s
	}
	return clipRunes(s, cols)
}

func wrapMono(s string, width, size, ls float64) []string {
	cols := monoCols(width, size, ls)
	words := strings.Fields(s)
	if cols == 0 || len(words) == 0 {
		return []string{s}
	}
	var lines []string
	line := ""
	for _, w := range words {
		w = clipRunes(w, cols)
		switch {
		case line == "":
			line = w
		case utf8.RuneCountInString(line)+1+utf8.RuneCountInString(w) <= cols:
			line += " " + w
		default:
			lines = append(lines, line)
			line = w
		}
	}
	return append(lines, line)
}
