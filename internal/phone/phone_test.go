package phone

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testHost = "my-mac.tail1234.ts.net"

// fakeScript writes an executable shell script that logs its arguments to log
// and then runs body.
func fakeScript(t *testing.T, dir, name, log, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"" + log + "\"\n" + body + "\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

type fixture struct {
	m       *Manager
	out     *bytes.Buffer
	dir     string
	tsLog   string
	lcLog   string
	state   string // file holding the fake `serve status --json` output
	running bool   // fake: Punisher is answering on the port
}

type opts struct {
	statusJSON string
	serveJSON  string
	serveFails bool
	// serve is applied, then the command fails (for example it was killed)
	serveFailsAfterApply bool
	serveStatusFails     bool
}

const testLogin = "brother@example.com"

func statusJSON(dns, selfExtra string) string {
	return `{"BackendState":"Running","Self":{"DNSName":"` + dns + `","UserID":7` + selfExtra + `},"User":{"7":{"LoginName":"` + testLogin + `"}}}`
}

var runningStatus = statusJSON(testHost+".", "")

func ourServe(funnel bool) string {
	s := `{"TCP":{"443":{"HTTPS":true}},"Web":{"` + testHost + `:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:4710"}}}}`
	if funnel {
		s += `,"AllowFunnel":{"` + testHost + `:443":true}`
	}
	return s + "}"
}

func newFixture(t *testing.T, o opts) *fixture {
	t.Helper()
	if o.statusJSON == "" {
		o.statusJSON = runningStatus
	}
	if o.serveJSON == "" {
		o.serveJSON = "{}"
	}
	dir := t.TempDir()
	f := &fixture{out: &bytes.Buffer{}, dir: dir, tsLog: filepath.Join(dir, "ts.log"), lcLog: filepath.Join(dir, "lc.log"), state: filepath.Join(dir, "serve.json")}
	if err := os.WriteFile(f.state, []byte(o.serveJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	statusFile := filepath.Join(dir, "status.json")
	if err := os.WriteFile(statusFile, []byte(o.statusJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	ourFile := filepath.Join(dir, "ours.json")
	if err := os.WriteFile(ourFile, []byte(ourServe(false)), 0o644); err != nil {
		t.Fatal(err)
	}
	serveAction := `cp "` + ourFile + `" "` + f.state + `"; echo served; exit 0`
	if o.serveFails {
		serveAction = `echo nope >&2; exit 1`
	}
	if o.serveFailsAfterApply {
		serveAction = `cp "` + ourFile + `" "` + f.state + `"; exit 1`
	}
	serveStatus := `cat "` + f.state + `"; exit 0`
	if o.serveStatusFails {
		serveStatus = `echo down >&2; exit 1`
	}
	ts := fakeScript(t, dir, "tailscale", f.tsLog, `
case "$1 $2" in
  "status --json") cat "`+statusFile+`"; exit 0;;
  "serve status") `+serveStatus+`;;
  "serve --bg") `+serveAction+`;;
  "serve --https=443") echo '{}' > "`+f.state+`"; exit 0;;
esac
exit 9`)
	lc := fakeScript(t, dir, "launchctl", f.lcLog, `
case "$1" in
  print) [ -f "`+dir+`/loaded" ] && exit 0; exit 3;;
  bootstrap) touch "`+dir+`/loaded"; exit 0;;
  bootout) rm -f "`+dir+`/loaded"; exit 0;;
esac
exit 9`)
	f.m = &Manager{
		DataDir: filepath.Join(dir, "data"), Executable: "/Users/me/bin/punisher", Home: filepath.Join(dir, "home"),
		UID: 501, Port: 4710, Env: []string{"PATH=/usr/bin:/a&b"}, Tailscale: ts, Launchctl: lc,
		Out: f.out, In: strings.NewReader(""),
		IsPunisher: func(int) bool { return f.running || fileExists(filepath.Join(dir, "loaded")) },
		PortFree:   func(int) bool { return !f.running && !fileExists(filepath.Join(dir, "loaded")) },
		Wait:       50 * time.Millisecond, Poll: time.Millisecond,
	}
	return f
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func read(p string) string {
	b, _ := os.ReadFile(p)
	return string(b)
}

func (f *fixture) assertNoChanges(t *testing.T) {
	t.Helper()
	if fileExists(f.m.plistPath()) || fileExists(filepath.Join(f.m.DataDir, settingsFile)) {
		t.Fatal("state was changed")
	}
	if strings.Contains(read(f.lcLog), "bootstrap") || strings.Contains(read(f.tsLog), "serve --bg") {
		t.Fatalf("must not act: launchctl=%q tailscale=%q", read(f.lcLog), read(f.tsLog))
	}
}

func TestOnPublishesAndSavesHost(t *testing.T) {
	f := newFixture(t, opts{})
	if err := f.m.On(context.Background()); err != nil {
		t.Fatalf("on: %v", err)
	}
	access, err := LoadAccess(f.m.DataDir)
	if err != nil || access != (Access{Host: testHost, Login: testLogin}) {
		t.Fatalf("access = %+v %v", access, err)
	}
	if !strings.Contains(read(f.tsLog), "serve --bg --https=443 4710") {
		t.Fatalf("tailscale calls:\n%s", read(f.tsLog))
	}
	if strings.Contains(strings.ToLower(read(f.tsLog)), "funnel") {
		t.Fatalf("funnel must never be used:\n%s", read(f.tsLog))
	}
	if !strings.Contains(read(f.lcLog), "bootstrap gui/501 "+f.m.plistPath()) {
		t.Fatalf("launchctl calls:\n%s", read(f.lcLog))
	}
	if fi, err := os.Stat(f.m.plistPath()); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("plist mode: %v %v", fi, err)
	}
	out := f.out.String()
	for _, want := range []string{"https://" + testHost + "/", "잠들면", "punisher phone off", "\x1b[30;47m"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestPlist(t *testing.T) {
	f := newFixture(t, opts{})
	f.m.CustomDataDir = true
	p := filepath.Join(t.TempDir(), "x.plist")
	body := f.m.plist()
	if err := os.WriteFile(p, body, 0o644); err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{"<string>" + Label + "</string>", "<string>/Users/me/bin/punisher</string>", "<string>4710</string>", "<string>--no-browser</string>", "<string>--data-dir</string>", "/usr/bin:/a&amp;b", "<key>RunAtLoad</key>"} {
		if !strings.Contains(s, want) {
			t.Errorf("plist missing %q:\n%s", want, s)
		}
	}
	if plutil, err := exec.LookPath("plutil"); err == nil {
		if out, err := exec.Command(plutil, "-lint", p).CombinedOutput(); err != nil {
			t.Fatalf("plutil: %v %s", err, out)
		}
	}
}

func TestOnRefuses(t *testing.T) {
	otherServe := `{"Web":{"` + testHost + `:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:9000"}}}}}`
	cases := []struct {
		name string
		o    opts
		want string
	}{
		{"funnel on", opts{serveJSON: ourServe(true)}, "funnel --https=443 off"},
		{"funnel in a foreground session", opts{serveJSON: `{"Foreground":{"abc":{"AllowFunnel":{"` + testHost + `:443":true}}}}`}, "Funnel"},
		{"other service", opts{serveJSON: otherServe}, "다른 프로그램"},
		{"file handler at root", opts{serveJSON: `{"Web":{"` + testHost + `:443":{"Handlers":{"/":{"Path":"/tmp/site"}}}}}`}, "다른 프로그램"},
		{"another path beside ours", opts{serveJSON: `{"Web":{"` + testHost + `:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:4710"},"/x":{"Proxy":"http://127.0.0.1:9000"}}}}}`}, "다른 프로그램"},
		{"foreground session", opts{serveJSON: `{"Foreground":{"abc":` + otherServe + `}}`}, "다른 프로그램"},
		{"tcp forward", opts{serveJSON: `{"TCP":{"443":{"TCPForward":"127.0.0.1:22"}}}`}, "다른 프로그램"},
		{"not logged in", opts{statusJSON: `{"BackendState":"NeedsLogin","Self":{"DNSName":""}}`}, "로그인"},
		{"no magicdns", opts{statusJSON: `{"BackendState":"Running","Self":{"DNSName":""}}`}, "MagicDNS"},
		{"not a tailnet name", opts{statusJSON: `{"BackendState":"Running","Self":{"DNSName":"evil.example."}}`}, "MagicDNS"},
		{"tagged device", opts{statusJSON: statusJSON(testHost+".", `,"Tags":["tag:server"]`)}, "태그"},
		{"no owner", opts{statusJSON: `{"BackendState":"Running","Self":{"DNSName":"` + testHost + `.","UserID":7}}`}, "태그"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, tc.o)
			err := f.m.On(context.Background())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			f.assertNoChanges(t)
		})
	}
}

func TestOnRefusesWhilePortBusy(t *testing.T) {
	f := newFixture(t, opts{})
	f.running = true
	if err := f.m.On(context.Background()); err == nil || !strings.Contains(err.Error(), "Ctrl+C") {
		t.Fatalf("err = %v", err)
	}
	f.assertNoChanges(t)

	f = newFixture(t, opts{})
	f.m.PortFree = func(int) bool { return false }
	f.m.IsPunisher = func(int) bool { return false }
	if err := f.m.On(context.Background()); err == nil || !strings.Contains(err.Error(), "다른 프로그램") {
		t.Fatalf("err = %v", err)
	}
	f.assertNoChanges(t)
}

func TestOnRollsBackWhenServeFails(t *testing.T) {
	f := newFixture(t, opts{serveFails: true})
	if err := f.m.On(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if fileExists(f.m.plistPath()) || fileExists(filepath.Join(f.m.DataDir, settingsFile)) || fileExists(filepath.Join(f.dir, "loaded")) {
		t.Fatal("rollback left state behind")
	}
}

func TestOnCleansUpServeLeftByFailedCommand(t *testing.T) {
	f := newFixture(t, opts{serveFailsAfterApply: true})
	if err := f.m.On(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(read(f.tsLog), "serve --https=443 off") {
		t.Fatalf("serve left behind:\n%s", read(f.tsLog))
	}
	if fileExists(f.m.plistPath()) {
		t.Fatal("agent left behind")
	}
}

func TestOnCancelledLeavesPhoneAccessOff(t *testing.T) {
	f := newFixture(t, opts{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.m.IsPunisher = func(int) bool { cancel(); return false }
	f.m.Wait = time.Minute
	if err := f.m.On(ctx); err == nil || !strings.Contains(err.Error(), "취소") {
		t.Fatalf("err = %v", err)
	}
	if fileExists(f.m.plistPath()) || fileExists(filepath.Join(f.m.DataDir, settingsFile)) || fileExists(filepath.Join(f.dir, "loaded")) {
		t.Fatal("cancel left state behind")
	}
}

func TestOnAfterFailureDuringRepublishEndsOff(t *testing.T) {
	f := newFixture(t, opts{})
	ctx := context.Background()
	if err := f.m.On(ctx); err != nil {
		t.Fatal(err)
	}
	f.m.IsPunisher = func(int) bool { return false }
	if err := f.m.On(ctx); err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(read(f.tsLog), "serve --https=443 off") || fileExists(filepath.Join(f.m.DataDir, settingsFile)) {
		t.Fatalf("not fully off:\n%s", read(f.tsLog))
	}
}

func TestOnRollsBackWhenAgentDoesNotStart(t *testing.T) {
	f := newFixture(t, opts{})
	f.m.IsPunisher = func(int) bool { return false }
	if err := f.m.On(context.Background()); err == nil || !strings.Contains(err.Error(), "punisher.log") {
		t.Fatalf("err = %v", err)
	}
	if fileExists(f.m.plistPath()) || strings.Contains(read(f.tsLog), "serve --bg") {
		t.Fatal("must not publish an app that did not start")
	}
}

func TestOnAgainReplacesAgent(t *testing.T) {
	f := newFixture(t, opts{})
	ctx := context.Background()
	if err := f.m.On(ctx); err != nil {
		t.Fatal(err)
	}
	f.out.Reset()
	if err := f.m.On(ctx); err != nil {
		t.Fatalf("second on: %v", err)
	}
	if strings.Count(read(f.lcLog), "bootout") != 1 || strings.Count(read(f.lcLog), "bootstrap") != 2 {
		t.Fatalf("launchctl calls:\n%s", read(f.lcLog))
	}
}

func TestOffUndoesOn(t *testing.T) {
	f := newFixture(t, opts{})
	ctx := context.Background()
	if err := f.m.On(ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.m.Off(ctx); err != nil {
		t.Fatalf("off: %v", err)
	}
	if fileExists(f.m.plistPath()) || fileExists(filepath.Join(f.m.DataDir, settingsFile)) || fileExists(filepath.Join(f.dir, "loaded")) {
		t.Fatal("off left state behind")
	}
	if !strings.Contains(read(f.tsLog), "serve --https=443 off") {
		t.Fatalf("tailscale calls:\n%s", read(f.tsLog))
	}
	if access, err := LoadAccess(f.m.DataDir); err != nil || access != (Access{}) {
		t.Fatalf("access after off = %+v %v", access, err)
	}
	if err := f.m.Off(ctx); err != nil {
		t.Fatalf("second off must be harmless: %v", err)
	}
}

func TestOffLeavesOtherServiceAlone(t *testing.T) {
	f := newFixture(t, opts{serveJSON: `{"Web":{"` + testHost + `:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:9000"}}}}}`})
	if err := f.m.Off(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(read(f.tsLog), " off") {
		t.Fatalf("must not turn off another service:\n%s", read(f.tsLog))
	}
}

func TestOffWorksWhileLoggedOut(t *testing.T) {
	f := newFixture(t, opts{statusJSON: `{"BackendState":"NeedsLogin"}`, serveJSON: ourServe(false)})
	if err := f.m.Off(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read(f.tsLog), "serve --https=443 off") {
		t.Fatalf("tailscale calls:\n%s", read(f.tsLog))
	}
}

func TestOffReportsWhatRemains(t *testing.T) {
	f := newFixture(t, opts{serveStatusFails: true})
	if err := saveAccess(f.m.DataDir, Access{Host: testHost, Login: testLogin}); err != nil {
		t.Fatal(err)
	}
	err := f.m.Off(context.Background())
	if err == nil || !strings.Contains(err.Error(), "한 번 더") {
		t.Fatalf("err = %v", err)
	}
	if fileExists(filepath.Join(f.m.DataDir, settingsFile)) {
		t.Fatal("access must be removed even when tailscale is unreachable")
	}
}

func TestOffLeavesSharedAddressAlone(t *testing.T) {
	f := newFixture(t, opts{serveJSON: `{"Web":{"` + testHost + `:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:4710"},"/x":{"Proxy":"http://127.0.0.1:9000"}}}}}`})
	if err := f.m.Off(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(read(f.tsLog), " off") || !strings.Contains(f.out.String(), "그대로") {
		t.Fatalf("tailscale calls:\n%s\noutput: %s", read(f.tsLog), f.out.String())
	}
}

func TestOffFinishesAfterCancel(t *testing.T) {
	f := newFixture(t, opts{})
	if err := f.m.On(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := f.m.Off(ctx); err != nil {
		t.Fatal(err)
	}
	if fileExists(f.m.plistPath()) || fileExists(filepath.Join(f.m.DataDir, settingsFile)) || fileExists(filepath.Join(f.dir, "loaded")) {
		t.Fatal("cancel left state behind")
	}
}

func TestOffKeepsBothErrors(t *testing.T) {
	f := newFixture(t, opts{serveStatusFails: true})
	f.m.Launchctl = fakeScript(t, f.dir, "launchctl-broken", f.lcLog, `[ "$1" = print ] && exit 0; echo broken >&2; exit 1`)
	err := f.m.Off(context.Background())
	if err == nil || !strings.Contains(err.Error(), "bootout") || !strings.Contains(err.Error(), "Tailscale 연결 설정") {
		t.Fatalf("err = %v", err)
	}
}

func TestOffWithoutTailscaleStillCleansUp(t *testing.T) {
	f := newFixture(t, opts{})
	ctx := context.Background()
	if err := f.m.On(ctx); err != nil {
		t.Fatal(err)
	}
	f.m.Tailscale = ""
	f.m.AppPaths = nil
	t.Setenv("PATH", t.TempDir())
	if err := f.m.Off(ctx); err != nil {
		t.Fatal(err)
	}
	if fileExists(f.m.plistPath()) || fileExists(filepath.Join(f.m.DataDir, settingsFile)) {
		t.Fatal("off left state behind")
	}
}

func TestFindTailscale(t *testing.T) {
	dir := t.TempDir()
	app := fakeScript(t, dir, "Tailscale", filepath.Join(dir, "log"), "exit 0")
	t.Setenv("PATH", t.TempDir())
	m := &Manager{AppPaths: []string{filepath.Join(dir, "missing"), app}}
	if got, err := m.findTailscale(); err != nil || got != app {
		t.Fatalf("app path: %q %v", got, err)
	}
	m.AppPaths = nil
	if _, err := m.findTailscale(); err == nil || !strings.Contains(err.Error(), "tailscale.com/download") {
		t.Fatalf("missing cli: %v", err)
	}
}

func TestValidHost(t *testing.T) {
	for h, want := range map[string]bool{
		testHost: true, "a.b.ts.net": true,
		"": false, "ts.net": false, "my-mac.ts.net.evil.example": false, "evil.example": false,
		"My-Mac.tail1.ts.net": false, testHost + ":443": false, "*." + testHost: false, "-a.b.ts.net": false,
	} {
		if got := ValidHost(h); got != want {
			t.Errorf("ValidHost(%q) = %v", h, got)
		}
	}
}

func TestValidLogin(t *testing.T) {
	for l, want := range map[string]bool{
		testLogin: true, "동생@example.com": true,
		"": false, taggedLogin: false, "a b@example.com": false, "a\n@example.com": false,
	} {
		if got := ValidLogin(l); got != want {
			t.Errorf("ValidLogin(%q) = %v", l, got)
		}
	}
}

func TestSettingsFile(t *testing.T) {
	dir := t.TempDir()
	good := Access{Host: testHost, Login: testLogin}
	if a, err := LoadAccess(dir); a != (Access{}) || err != nil {
		t.Fatalf("absent: %+v %v", a, err)
	}
	if err := saveAccess(dir, Access{Host: "evil.example", Login: testLogin}); err == nil {
		t.Fatal("must refuse a non-tailnet host")
	}
	if err := saveAccess(dir, Access{Host: testHost}); err == nil {
		t.Fatal("must refuse a missing owner")
	}
	if err := saveAccess(dir, good); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dir, settingsFile))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm %v %v", fi.Mode(), err)
	}
	for _, bad := range []string{`{`, `{"allowedHost":"evil.example","ownerLogin":"a@b"}`, `{"allowedHost":""}`, `{"allowedHost":"` + testHost + `"}`} {
		if err := os.WriteFile(filepath.Join(dir, settingsFile), []byte(bad), 0o600); err != nil {
			t.Fatal(err)
		}
		if a, err := LoadAccess(dir); a != (Access{}) || err == nil {
			t.Fatalf("%s: %+v %v", bad, a, err)
		}
	}
}

func TestIsPunisherAt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"service": "punisher"})
	}))
	defer srv.Close()
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"service": "something-else"})
	}))
	defer other.Close()
	if !isPunisherAt(srv.URL) || isPunisherAt(other.URL) {
		t.Fatal("wrong identification")
	}
}

func TestWriteQR(t *testing.T) {
	var b bytes.Buffer
	if err := writeQR(&b, "https://"+testHost+"/"); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	if len(lines) < 12 {
		t.Fatalf("too few lines: %d", len(lines))
	}
	width := len([]rune(lines[0]))
	for _, l := range lines {
		if len([]rune(l)) != width || !strings.HasPrefix(l, "\x1b[30;47m") {
			t.Fatalf("bad line %q", l)
		}
	}
}
