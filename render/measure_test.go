package render

import "testing"

func TestStatsSlashX(t *testing.T) {
	cases := []struct {
		login string
		want  float64
	}{
		{"wewesss", 86},
		{"abcdefghijklmnopqrst", 188},
	}
	for _, c := range cases {
		if got := statsSlashX(c.login); got != c.want {
			t.Errorf("statsSlashX(%q) = %v, want %v", c.login, got, c.want)
		}
	}
}
