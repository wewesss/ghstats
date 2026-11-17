// Package render turns a report into a JSON document and a configurable SVG card.
package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/wewesss/ghstats/report"
)

func JSON(w io.Writer, r report.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r); err != nil {
		return fmt.Errorf("render: encode json: %w", err)
	}
	return nil
}
