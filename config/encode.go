package config

import (
	"io"

	"go.yaml.in/yaml/v3"
)

func Encode(w io.Writer, c Config) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if len(c.Render.Cards) > 0 {
		c.Render.Blocks = nil
	}
	if err := enc.Encode(c); err != nil {
		return err
	}
	return enc.Close()
}
