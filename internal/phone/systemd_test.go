package phone

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// linuxFixture swaps launchctl for a fake systemctl (state = a file "active").
func linuxFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t, opts{})
	sc := fakeScript(t, f.dir, "systemctl", f.lcLog, `
case "$2" in
  is-active) [ -f "`+f.dir+`/loaded" ] && exit 0; exit 3;;
  daemon-reload) exit 0;;
  enable) touch "`+f.dir+`/loaded"; exit 0;;
  disable) rm -f "`+f.dir+`/loaded"; exit 0;;
esac
exit 9`)
	f.m.GOOS, f.m.Systemctl, f.m.Launchctl = "linux", sc, "/nonexistent"
	return f
}

func TestLinuxOnInstallsUserService(t *testing.T) {
	f := linuxFixture(t)
	if err := f.m.On(context.Background()); err != nil {
		t.Fatalf("on: %v", err)
	}
	calls := read(f.lcLog)
	for _, want := range []string{"--user daemon-reload", "--user enable --now punisher.service"} {
		if !strings.Contains(calls, want) {
			t.Fatalf("systemctl calls miss %q:\n%s", want, calls)
		}
	}
	unit := read(f.m.unitPath())
	for _, want := range []string{`ExecStart="/Users/me/bin/punisher" "--port" "4710" "--no-browser"`, `Environment="PATH=/usr/bin:/a&b"`,
		"Restart=on-failure", "WantedBy=default.target"} {
		if !strings.Contains(unit, want) {
			t.Fatalf("unit misses %q:\n%s", want, unit)
		}
	}
	if fi, err := os.Stat(f.m.unitPath()); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("unit mode: %v %v", fi, err)
	}
	if !strings.Contains(f.out.String(), "loginctl enable-linger") {
		t.Fatalf("missing linger hint:\n%s", f.out.String())
	}
	if err := f.m.Off(context.Background()); err != nil {
		t.Fatalf("off: %v", err)
	}
	if !strings.Contains(read(f.lcLog), "--user disable --now punisher.service") || fileExists(f.m.unitPath()) {
		t.Fatalf("off must disable and remove the unit:\n%s", read(f.lcLog))
	}
}

func TestSystemdQuote(t *testing.T) {
	if got := systemdQuote(`a b"c%d$e\f`); got != `"a b\"c%%d$$e\\f"` {
		t.Fatalf("quote = %s", got)
	}
	m := &Manager{Home: "/h", Executable: "/x", Port: 1, CustomDataDir: true, DataDir: "/d d"}
	if !strings.Contains(string(m.unit()), `"--data-dir" "/d d"`) || filepath.Dir(m.unitPath()) != "/h/.config/systemd/user" {
		t.Fatalf("unit:\n%s", m.unit())
	}
}
