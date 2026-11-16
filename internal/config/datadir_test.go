package config

import "testing"

func TestDefaultDataDir(t *testing.T) {
	none := func(string) bool { return false }
	cases := []struct {
		goos, xdg string
		exists    func(string) bool
		want      string
	}{
		{"darwin", "", none, "/h/Library/Application Support/Punisher"},
		{"linux", "", none, "/h/.local/share/punisher"},
		{"linux", "/x", none, "/x/punisher"},
		{"linux", "rel", none, "/h/.local/share/punisher"},
		{"linux", "/x", func(p string) bool { return p == "/h/.punisher" }, "/h/.punisher"},
	}
	for _, c := range cases {
		if got := defaultDataDir(c.goos, "/h", c.xdg, c.exists); got != c.want {
			t.Errorf("%s xdg=%q: %s, want %s", c.goos, c.xdg, got, c.want)
		}
	}
}
