package clicore

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testCodes = CodesFor("fake", "log in")

func shProc(t *testing.T) Proc {
	t.Helper()
	p, err := NewProc("/bin/sh", t.TempDir(), nil, BaseEnvKeys...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func ok() (string, error) { return "ok", nil }

func TestFinishArgTooLongIsInvalidRequest(t *testing.T) {
	p, err := NewProc("/bin/echo", t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// One argument over the OS limit (macOS ARG_MAX 1 MiB, Linux MAX_ARG_STRLEN 128 KiB).
	_, _, runErr := p.Run(context.Background(), []string{strings.Repeat("x", 2<<20)}, "", nil)
	if !ArgTooLong(runErr) {
		t.Fatalf("run error = %v, want E2BIG", runErr)
	}
	ctx := context.Background()
	_, ae := Finish(ctx, ctx, testCodes, runErr, "", ok)
	if ae == nil || ae.Code != CodeInvalid || ae.Status != http.StatusBadRequest {
		t.Fatalf("E2BIG must be invalid_request, not %s_unavailable: %+v", "fake", ae)
	}
}

func TestRunStopsAfterFinalEventAndKillsGroup(t *testing.T) {
	p := shProc(t)
	start := time.Now()
	var lines []string
	// The CLI prints its final event, then it and a child keep stdout open
	// (the shell forks its last sleep right as the group is killed).
	_, _, err := p.Run(context.Background(), []string{"-c", `(sleep 30) & sleep 0.2; echo a; echo final; sleep 30`}, "", func(l []byte) error {
		lines = append(lines, string(l))
		if string(l) == "final" {
			return ErrStop
		}
		return nil
	})
	if err != nil || strings.Join(lines, ",") != "a,final" {
		t.Fatalf("err=%v lines=%q", err, lines)
	}
	if d := time.Since(start); d > 4*time.Second {
		t.Fatalf("Run waited %s after the final event", d)
	}
}

func TestRunReportsScannerError(t *testing.T) {
	p := shProc(t)
	// One 17 MB line: over the 16 MiB scanner limit (bufio.ErrTooLong).
	_, _, runErr := p.Run(context.Background(), []string{"-c", `head -c 17000000 /dev/zero | tr '\000' a; echo; echo '{"type":"result"}'`}, "",
		func([]byte) error { return nil })
	if !errors.Is(runErr, ErrStdoutRead) {
		t.Fatalf("run error = %v, want ErrStdoutRead", runErr)
	}
	ctx := context.Background()
	_, ae := Finish(ctx, ctx, testCodes, runErr, "", ok)
	if ae == nil || ae.Code != testCodes.CLIError {
		t.Fatalf("read failure must be a fixed fake_error: %+v", ae)
	}
}

func TestFinishWaitDelayWithParsedResultIsSuccess(t *testing.T) {
	ctx := context.Background()
	if res, ae := Finish(ctx, ctx, testCodes, exec.ErrWaitDelay, "", ok); ae != nil || res != "ok" {
		t.Fatalf("ErrWaitDelay + parsed result = %q %+v", res, ae)
	}
	fail := func() (string, error) { return "", NewError(502, testCodes.CLIError, "no result") }
	if _, ae := Finish(ctx, ctx, testCodes, exec.ErrWaitDelay, "", fail); ae == nil || ae.Code != testCodes.CLIError {
		t.Fatalf("ErrWaitDelay without a result must fail: %+v", ae)
	}
}

func TestFinishClassifiesStderrNotStdoutError(t *testing.T) {
	ctx := context.Background()
	// The parse error carries generated text mentioning 401; the process failed with empty stderr.
	parse := func() (string, error) {
		return "", NewError(502, testCodes.CLIError, "partial: HTTP 401 vs 429 explained")
	}
	exitErr := exec.Command("/bin/sh", "-c", "exit 1").Run()
	if _, ae := Finish(ctx, ctx, testCodes, exitErr, "", parse); ae == nil || ae.Code != testCodes.CLIError {
		t.Fatalf("empty stderr must keep the fixed error: %+v", ae)
	}
	if _, ae := Finish(ctx, ctx, testCodes, exitErr, "Error: not logged in", parse); ae == nil || ae.Code != testCodes.NotLoggedIn {
		t.Fatalf("stderr must be classified: %+v", ae)
	}
}

func TestBinCacheRetriesIncompleteProbe(t *testing.T) {
	old := incompleteTTL
	incompleteTTL = 50 * time.Millisecond
	t.Cleanup(func() { incompleteTTL = old })
	bin := filepath.Join(t.TempDir(), "cli")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var c BinCache[int]
	builds, complete := 0, false
	get := func() int {
		v, err := c.Get(func() string { return bin }, func(string) (int, bool, error) {
			builds++
			return builds, complete, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	if get() != 1 || get() != 1 {
		t.Fatal("an incomplete probe is reused within its TTL")
	}
	time.Sleep(60 * time.Millisecond)
	complete = true
	if get() != 2 {
		t.Fatal("an incomplete probe must be re-built after its TTL")
	}
	time.Sleep(60 * time.Millisecond)
	if get() != 2 || builds != 2 {
		t.Fatal("a complete probe is cached while the executable exists")
	}
}

func TestProbeHelpReportsFailure(t *testing.T) {
	p := shProc(t)
	if got, err := p.ProbeHelp(context.Background(), []string{"--a", "--b"}, "-c", "echo usage --a"); err != nil || !got["--a"] || got["--b"] {
		t.Fatalf("probe = %v %v", got, err)
	}
	if _, err := p.ProbeHelp(context.Background(), []string{"--a"}, "-c", "exit 3"); err == nil {
		t.Fatal("unreadable help must be reported")
	}
}

func TestChildEnvPassesProxyNotAPIKeys(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://proxy:3128")
	t.Setenv("no_proxy", "localhost")
	t.Setenv("NODE_EXTRA_CA_CERTS", "/ca.pem")
	t.Setenv("OPENAI_API_KEY", "x")
	t.Setenv("ANTHROPIC_API_KEY", "x")
	t.Setenv("GEMINI_API_KEY", "x")
	env := strings.Join(ChildEnv("/bin", nil, BaseEnvKeys...), "\n")
	for _, want := range []string{"HTTPS_PROXY=http://proxy:3128", "no_proxy=localhost", "NODE_EXTRA_CA_CERTS=/ca.pem"} {
		if !strings.Contains(env, want) {
			t.Errorf("env missing %s", want)
		}
	}
	if strings.Contains(env, "API_KEY") {
		t.Errorf("API keys must not reach the child:\n%s", env)
	}
}

func TestStatusCache(t *testing.T) {
	var c StatusCache
	if _, ok := c.Get("b", time.Minute); ok {
		t.Fatal("empty cache hit")
	}
	c.Put("b", StatusProbe{Version: "1"})
	if p, ok := c.Get("b", time.Minute); !ok || p.Version != "1" {
		t.Fatal("cached probe missing")
	}
	if _, ok := c.Get("other", time.Minute); ok {
		t.Fatal("another executable must miss")
	}
	c.Reset()
	if _, ok := c.Get("b", time.Minute); ok {
		t.Fatal("reset must miss")
	}
}
