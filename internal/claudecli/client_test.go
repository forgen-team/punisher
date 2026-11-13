package claudecli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wooo-jin/punisher/internal/llm"
)

// fakeClaude stands in for the claude CLI. Behaviour is selected by the
// prompt on stdin; `auth status` reads __AUTH__ (true/false/missing).
const fakeClaude = `#!/bin/sh
case "$1" in
  --help) echo "--system-prompt --tools --setting-sources --strict-mcp-config --mcp-config --no-session-persistence --include-partial-messages"; exit 0;;
  --version) echo "9.9.9 (Claude Code)"; exit 0;;
  auth)
    if [ -f "__AUTH__" ]; then echo "{\"loggedIn\": $(cat "__AUTH__")}"; exit 0; fi
    echo "unknown command" >&2; exit 1;;
esac
printf '%s\n' "$@" > "__ARGS__"
pwd > "__PWD__"
env > "__ENV__"
prompt=$(cat)
printf '%s' "$prompt" > "__STDIN__"
case "$prompt" in
  *MODE_SLEEP*) sleep 30; exit 0;;
  *MODE_AUTH*) echo '{"type":"result","subtype":"success","is_error":true,"result":"Invalid API key · Please run /login"}'; exit 1;;
  *MODE_RATE*) echo '{"type":"result","subtype":"success","is_error":true,"result":"Claude AI usage limit reached|1762400000"}'; exit 1;;
  *MODE_TRUNC*) echo '{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"par"}}}'; exit 0;;
esac
for a in "$@"; do
  if [ "$a" = "stream-json" ]; then
    echo '{"type":"system","subtype":"init"}'
    echo '{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hel"}}}'
    echo '{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"lo"}}}'
    echo '{"type":"assistant","message":{"model":"claude-haiku-x","content":[{"type":"text","text":"Hello"}]}}'
    echo '{"type":"result","subtype":"success","is_error":false,"result":"Hello","usage":{"input_tokens":5,"output_tokens":2},"modelUsage":{"claude-haiku-x":{}}}'
    exit 0
  fi
done
echo '{"type":"result","subtype":"success","is_error":false,"result":"ok","usage":{"input_tokens":10,"output_tokens":3,"cache_read_input_tokens":4},"modelUsage":{"claude-sonnet-x":{}}}'
`

type fixture struct {
	client                           *Client
	dir, args, pwd, env, stdin, auth string
	tempBase                         string
}

// newFixture writes the fake CLI as "claude" into a temp dir that is put
// first on PATH, so the client finds it by auto-detection like a real install.
func newFixture(t *testing.T, mutate func(*Config)) *fixture {
	t.Helper()
	dir := t.TempDir()
	f := &fixture{dir: dir, args: filepath.Join(dir, "args"), pwd: filepath.Join(dir, "pwd"), env: filepath.Join(dir, "env"),
		stdin: filepath.Join(dir, "stdin"), auth: filepath.Join(dir, "auth"), tempBase: filepath.Join(dir, "tmp")}
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := strings.NewReplacer("__ARGS__", f.args, "__PWD__", f.pwd, "__ENV__", f.env,
		"__STDIN__", f.stdin, "__AUTH__", f.auth).Replace(fakeClaude)
	if err := os.WriteFile(filepath.Join(binDir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	cfg := Config{TempBase: f.tempBase, Concurrency: 1, DefaultTimeout: 10 * time.Second}
	if mutate != nil {
		mutate(&cfg)
	}
	f.client = New(cfg)
	return f
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCreateRunsCLIWithToolsDisabledInEmptyDir(t *testing.T) {
	f := newFixture(t, nil)
	res, err := f.client.Create(context.Background(), "ignored", llm.MessagesRequest{
		Model: "haiku", System: "SYS", User: "hello prompt"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if res.Text != "ok" || res.Model != "claude-sonnet-x" || res.Usage.InputTokens != 14 || res.Usage.OutputTokens != 3 {
		t.Fatalf("result = %+v", res)
	}
	args := read(t, f.args)
	for _, want := range []string{"-p\n", "--model\nhaiku\n", "--output-format\njson\n", "--system-prompt\nSYS\n", "--tools\n\n", "--max-turns\n1\n", "--disallowedTools\n"} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q:\n%s", want, args)
		}
	}
	if got := read(t, f.stdin); got != "hello prompt" {
		t.Fatalf("prompt via stdin = %q", got)
	}
	pwd := strings.TrimPrefix(strings.TrimSpace(read(t, f.pwd)), "/private")
	base := strings.TrimPrefix(f.tempBase, "/private")
	if !strings.HasPrefix(pwd, base+string(os.PathSeparator)+"call-") {
		t.Fatalf("cwd %q is not an isolated call dir under %q", pwd, base)
	}
	if _, err := os.Stat(pwd); !os.IsNotExist(err) {
		t.Fatalf("call dir not removed: %v", err)
	}
	env := read(t, f.env)
	if !strings.Contains(env, "DISABLE_AUTOUPDATER=1") || !strings.Contains(env, "PATH="+filepath.Join(f.dir, "bin")) {
		t.Fatalf("child env missing autoupdate off / bin dir on PATH:\n%s", env)
	}
}

func TestStreamDeliversDeltasInOrder(t *testing.T) {
	f := newFixture(t, nil)
	var got []string
	res, err := f.client.Stream(context.Background(), "", llm.MessagesRequest{User: "stream please"}, func(d string) error {
		got = append(got, d)
		return nil
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if strings.Join(got, "|") != "Hel|lo" || res.Text != "Hello" || res.Model != "claude-haiku-x" {
		t.Fatalf("deltas=%v result=%+v", got, res)
	}
	if args := read(t, f.args); !strings.Contains(args, "stream-json") || !strings.Contains(args, "--include-partial-messages") {
		t.Fatalf("stream args:\n%s", args)
	}
}

func TestStreamWithoutResultIsError(t *testing.T) {
	f := newFixture(t, nil)
	_, err := f.client.Stream(context.Background(), "", llm.MessagesRequest{User: "MODE_TRUNC"}, nil)
	if err == nil {
		t.Fatal("truncated stream must fail")
	}
}

func TestErrorMapping(t *testing.T) {
	f := newFixture(t, nil)
	ctx := context.Background()
	_, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "MODE_AUTH"})
	if !errors.Is(err, llm.ErrGenerationUnavailable) || ErrorCode(err) != CodeNotLoggedIn {
		t.Fatalf("auth failure = %v (code %s)", err, ErrorCode(err))
	}
	if st := f.client.Status(ctx); st.Code != CodeNotLoggedIn || st.Ready {
		t.Fatalf("status after auth failure = %+v", st)
	}
	_, err = f.client.Create(ctx, "", llm.MessagesRequest{User: "MODE_RATE"})
	if !errors.Is(err, llm.ErrRateLimited) || ErrorCode(err) != CodeRateLimited {
		t.Fatalf("rate failure = %v", err)
	}
	if _, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "x", Model: "gpt-4"}); !errors.Is(err, llm.ErrInvalidRequest) {
		t.Fatalf("bad model = %v", err)
	}
	if _, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "  "}); !errors.Is(err, llm.ErrInvalidRequest) {
		t.Fatalf("empty prompt = %v", err)
	}
	if _, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "fine"}); err != nil {
		t.Fatalf("recovery call: %v", err)
	}
	if st := f.client.Status(ctx); st.LastFailure != nil {
		t.Fatalf("success must clear last failure: %+v", st.LastFailure)
	}
}

func TestTimeoutKillsProcessGroup(t *testing.T) {
	f := newFixture(t, nil)
	start := time.Now()
	_, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "MODE_SLEEP", Timeout: time.Second})
	if !errors.Is(err, llm.ErrTimeout) {
		t.Fatalf("err = %v, want timeout", err)
	}
	if elapsed := time.Since(start); elapsed > 8*time.Second {
		t.Fatalf("timeout took %s (process not killed)", elapsed)
	}
}

func TestMissingCLIIsUnavailable(t *testing.T) {
	c := New(Config{Bin: filepath.Join(t.TempDir(), "no-claude-here"), TempBase: t.TempDir()})
	_, err := c.Create(context.Background(), "", llm.MessagesRequest{User: "hi"})
	if !errors.Is(err, llm.ErrGenerationUnavailable) || ErrorCode(err) != CodeCLIMissing {
		t.Fatalf("missing CLI = %v (code %s)", err, ErrorCode(err))
	}
	if st := c.Status(context.Background()); st.Installed || st.Code != CodeCLIMissing || st.Ready {
		t.Fatalf("status = %+v", st)
	}
}

func TestStatusReadsAuthStatus(t *testing.T) {
	f := newFixture(t, nil)
	ctx := context.Background()
	st := f.client.Status(ctx)
	if !st.Installed || st.Version != "9.9.9 (Claude Code)" || st.LoggedIn != nil || !st.Ready {
		t.Fatalf("status without auth command = %+v", st)
	}

	f2 := newFixture(t, nil)
	if err := os.WriteFile(f2.auth, []byte("false"), 0o644); err != nil {
		t.Fatal(err)
	}
	st = f2.client.Status(ctx)
	if st.LoggedIn == nil || *st.LoggedIn || st.Code != CodeNotLoggedIn || st.Ready {
		t.Fatalf("logged-out status = %+v", st)
	}

	f3 := newFixture(t, nil)
	if err := os.WriteFile(f3.auth, []byte("true"), 0o644); err != nil {
		t.Fatal(err)
	}
	st = f3.client.Status(ctx)
	if st.LoggedIn == nil || !*st.LoggedIn || st.Code != "" || !st.Ready {
		t.Fatalf("logged-in status = %+v", st)
	}
}

func TestDailyLimit(t *testing.T) {
	f := newFixture(t, func(c *Config) { c.DailyLimit = 1 })
	ctx := context.Background()
	if _, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "one"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "two"}); !errors.Is(err, llm.ErrRateLimited) || ErrorCode(err) != CodeDailyCap {
		t.Fatalf("second call = %v", err)
	}
}

func TestBackgroundWaitsBehindInteractive(t *testing.T) {
	f := newFixture(t, func(c *Config) { c.Concurrency = 2; c.BackgroundConcurrency = 2 })
	bg := llm.WithBackground(context.Background())
	// Background may never take the last free slot: with one interactive call
	// running, a background call must wait until it finishes.
	var wg sync.WaitGroup
	wg.Add(1)
	interactiveDone := make(chan time.Time, 1)
	go func() {
		defer wg.Done()
		_, _ = f.client.Create(context.Background(), "", llm.MessagesRequest{User: "MODE_SLEEP", Timeout: 1500 * time.Millisecond})
		interactiveDone <- time.Now()
	}()
	waitFor(t, func() bool { a, _ := f.client.limiter.Stats(); return a == 1 })
	if _, err := f.client.Create(bg, "", llm.MessagesRequest{User: "bg"}); err != nil {
		t.Fatalf("background call: %v", err)
	}
	finished := time.Now()
	wg.Wait()
	if done := <-interactiveDone; done.After(finished) {
		t.Fatal("background ran before the interactive call released its slot")
	}
}

func TestLimiterLanePolicy(t *testing.T) {
	l := NewLimiter(LimiterConfig{Total: 2, BackgroundMax: 1, InteractiveQueue: 4, BackgroundQueue: 4})
	ctx := context.Background()
	a, _ := l.Acquire(ctx, LaneInteractive)
	b, _ := l.Acquire(ctx, LaneInteractive)

	bgDone := make(chan func(), 1)
	go func() { r, _ := l.Acquire(ctx, LaneBackground); bgDone <- r }()
	waitFor(t, func() bool { return l.LaneStats()[LaneBackground].Queued == 1 })
	icDone := make(chan func(), 1)
	go func() { r, _ := l.Acquire(ctx, LaneInteractive); icDone <- r }()
	waitFor(t, func() bool { return l.LaneStats()[LaneInteractive].Queued == 1 })

	a() // the freed slot goes to the interactive waiter, not the older background one
	c := <-icDone
	b() // one slot free, but background may not take the last free slot
	select {
	case <-bgDone:
		t.Fatal("background took the last free slot")
	case <-time.After(100 * time.Millisecond):
	}
	c()
	bg := <-bgDone
	if _, err := l.Acquire(ctx, LaneInteractive); err != nil {
		t.Fatalf("interactive must get the reserved slot: %v", err)
	}
	bg()
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not reached")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestShortCallNotBehindLongCalls: with the default 3 slots, two long
// generations running leave a slot for a short call (pretest, tutor, Q&A).
func TestShortCallNotBehindLongCalls(t *testing.T) {
	f := newFixture(t, func(c *Config) { c.Concurrency = 0 }) // 0 = default 3
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = f.client.Create(context.Background(), "", llm.MessagesRequest{User: "MODE_SLEEP", Timeout: 1500 * time.Millisecond})
		}()
	}
	waitFor(t, func() bool { return f.client.limiter.LaneStats()[LaneInteractive].Active == 2 })
	start := time.Now()
	if _, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "q", Priority: llm.PriorityShort}); err != nil {
		t.Fatalf("short call: %v", err)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("short call waited %s behind two long calls", d)
	}
	wg.Wait()
}

func TestLimiterShortLane(t *testing.T) {
	l := NewLimiter(LimiterConfig{Total: 3, BackgroundMax: 1, InteractiveQueue: 4, BackgroundQueue: 4})
	ctx := context.Background()
	long1, _ := l.Acquire(ctx, LaneInteractive)
	long2, _ := l.Acquire(ctx, LaneInteractive)
	quick, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if _, err := l.Acquire(quick, LaneInteractive); err == nil {
		t.Fatal("a third long call took the short slot")
	}
	short, err := l.Acquire(ctx, LaneShort)
	if err != nil {
		t.Fatal(err)
	}
	short()
	bgDone := make(chan func(), 1)
	go func() { r, _ := l.Acquire(ctx, LaneBackground); bgDone <- r }()
	waitFor(t, func() bool { return l.LaneStats()[LaneBackground].Queued == 1 })
	long1()
	bg := <-bgDone
	if s, err := l.Acquire(ctx, LaneShort); err != nil {
		t.Fatalf("short slot with long + background running: %v", err)
	} else {
		s()
	}
	if ParseLane("short") != LaneShort || ParseLane("") != LaneInteractive || ParseLane("background") != LaneBackground {
		t.Fatal("ParseLane")
	}
	bg()
	long2()
}
