package codexcli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/clicore"
	"github.com/forgen-team/punisher/internal/llm"
)

// fakeCodex stands in for `codex`. Behaviour is selected by the prompt on
// stdin; `login status` exits by __LOGIN__ (missing = logged in); __HELP__
// (when present) replaces the exec help text, "FAIL" making it exit 1.
// --version / login status append to __COUNT__.
const fakeCodex = `#!/bin/sh
if [ "$1" = "--version" ]; then echo v >> "__COUNT__"; echo "codex-cli 9.9.9"; exit 0; fi
if [ "$1" = "login" ] && [ "$2" = "status" ]; then
  echo l >> "__COUNT__"
  if [ -f "__LOGIN__" ]; then echo "Not logged in" >&2; exit 1; fi
  echo "Logged in using ChatGPT"; exit 0
fi
if [ "$1" = "exec" ] && [ "$2" = "--help" ]; then
  if [ -f "__HELP__" ]; then
    if [ "$(cat "__HELP__")" = "FAIL" ]; then exit 1; fi
    cat "__HELP__"; exit 0
  fi
  echo "-c, --config --disable <FEATURE> --json --skip-git-repo-check --sandbox --ephemeral --ignore-user-config --ignore-rules --color"; exit 0
fi
if [ "$1" = "features" ] && [ "$2" = "list" ]; then
  printf 'apps   stable  true\nshell_tool   stable   true\nunified_exec  stable  true\n'; exit 0
fi
printf '%s\n' "$@" > "__ARGS__"
pwd > "__PWD__"
prompt=$(cat)
printf '%s' "$prompt" > "__STDIN__"
case "$prompt" in
  *MODE_SLEEP*) sleep 30; exit 0;;
  *MODE_AUTH*) echo '{"type":"thread.started","thread_id":"t"}'; echo "Error: Not logged in. Run codex login" >&2; exit 1;;
  *MODE_RATE*) echo '{"type":"turn.started"}'; echo '{"type":"turn.failed","error":{"message":"You have hit your usage limit. Try again later."}}'; exit 1;;
  *MODE_TRUNC*) echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"par"}}'; exit 0;;
  *MODE_LECTURE401*) echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"HTTP 401 Unauthorized vs 429 Too Many Requests"}}'; exit 1;;
  *MODE_TOOL*)
    echo '{"type":"turn.started"}'
    echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"I will look at the files first."}}'
    echo '{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"ls","status":"in_progress"}}'
    echo '{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"ls","status":"completed"}}'
    echo '{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"{\"answer\":42}"}}'
    echo '{"type":"turn.completed","usage":{"input_tokens":5,"output_tokens":2}}'
    exit 0;;
  *MODE_LINGER*)
    (sleep 30) &
    sleep 0.2
    echo '{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"done"}}'
    echo '{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1}}'
    sleep 30; exit 0;;
  *MODE_RETRY*)
    echo '{"type":"error","message":"Reconnecting... 1/5"}'
    echo '{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"after retry"}}'
    echo '{"type":"turn.completed","usage":{"input_tokens":3,"output_tokens":1}}'
    exit 0;;
esac
echo '{"type":"thread.started","thread_id":"0199"}'
echo '{"type":"turn.started"}'
echo '{"type":"item.completed","item":{"id":"item_0","type":"reasoning","text":"thinking"}}'
echo '{"type":"item.started","item":{"id":"item_1","type":"agent_message","text":"Hel"}}'
echo '{"type":"item.updated","item":{"id":"item_1","type":"agent_message","text":"Hello"}}'
echo '{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"Hello world"}}'
echo '{"type":"turn.completed","usage":{"input_tokens":120,"cached_input_tokens":100,"output_tokens":7}}'
`

type fixture struct {
	client                                    *Client
	dir, args, pwd, stdin, login, help, count string
	tempBase                                  string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	f := &fixture{dir: dir, args: filepath.Join(dir, "args"), pwd: filepath.Join(dir, "pwd"), stdin: filepath.Join(dir, "stdin"),
		login: filepath.Join(dir, "logged-out"), help: filepath.Join(dir, "help"), count: filepath.Join(dir, "count"), tempBase: filepath.Join(dir, "tmp")}
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := strings.NewReplacer("__ARGS__", f.args, "__PWD__", f.pwd, "__STDIN__", f.stdin, "__LOGIN__", f.login,
		"__HELP__", f.help, "__COUNT__", f.count).Replace(fakeCodex)
	if err := os.WriteFile(filepath.Join(binDir, "codex"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	f.client = New(Config{TempBase: f.tempBase, Gate: clicore.NewGate(clicore.GateConfig{Concurrency: 1, DefaultTimeout: 10 * time.Second})})
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

func TestCreateBuildsReadOnlyExecAndParsesJSONL(t *testing.T) {
	f := newFixture(t)
	res, err := f.client.Create(context.Background(), "ignored", llm.MessagesRequest{System: "SYS", User: "hello prompt"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if res.Text != "Hello world" || res.Model != DefaultModelName || res.Usage.InputTokens != 120 || res.Usage.OutputTokens != 7 {
		t.Fatalf("result = %+v", res)
	}
	args := read(t, f.args)
	for _, want := range []string{"exec\n--json\n", "--skip-git-repo-check\n", "--sandbox\nread-only\n", "--ephemeral\n",
		"--ignore-user-config\n", "--ignore-rules\n", "--color\nnever\n", "--disable\nshell_tool\n", "--disable\nunified_exec\n",
		"-c\nweb_search=\"disabled\"\n"} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q:\n%s", want, args)
		}
	}
	if !strings.HasSuffix(args, "\n-\n") {
		t.Errorf("prompt must come from stdin (last arg -):\n%s", args)
	}
	if strings.Contains(args, "-m\n") || strings.Contains(args, "model_reasoning_effort") {
		t.Errorf("no -m / effort override without a configured model or fast tier:\n%s", args)
	}
	stdin := read(t, f.stdin)
	if !strings.HasPrefix(stdin, "<system_instructions>\nSYS\n\n"+noToolsInstruction+"\n</system_instructions>\n\n") || !strings.HasSuffix(stdin, "hello prompt") {
		t.Fatalf("stdin = %q", stdin)
	}
	pwd := strings.TrimPrefix(strings.TrimSpace(read(t, f.pwd)), "/private")
	if base := strings.TrimPrefix(f.tempBase, "/private"); !strings.HasPrefix(pwd, base+string(os.PathSeparator)+"call-") {
		t.Fatalf("cwd %q is not an isolated call dir under %q", pwd, base)
	}
}

func TestModelFlagOnlyWhenSet(t *testing.T) {
	f := newFixture(t)
	res, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "hi", Model: "gpt-5.5"})
	if err != nil {
		t.Fatal(err)
	}
	if args := read(t, f.args); !strings.Contains(args, "-m\ngpt-5.5\n") || res.Model != "gpt-5.5" {
		t.Fatalf("model: %+v\n%s", res, args)
	}
	if _, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "hi", Model: "bad model;rm"}); !errors.Is(err, llm.ErrInvalidRequest) {
		t.Fatalf("bad model = %v", err)
	}
}

func TestStreamDeliversOnlyTheFinalAgentMessage(t *testing.T) {
	f := newFixture(t)
	var got []string
	res, err := f.client.Stream(context.Background(), "", llm.MessagesRequest{User: "stream"}, func(d string) error {
		got = append(got, d)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, "|") != "Hello world" || res.Text != "Hello world" {
		t.Fatalf("deltas=%q text=%q", got, res.Text)
	}
}

func TestPreToolNarrationDoesNotLeak(t *testing.T) {
	f := newFixture(t)
	var got []string
	res, err := f.client.Stream(context.Background(), "", llm.MessagesRequest{User: "MODE_TOOL"}, func(d string) error {
		got = append(got, d)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != `{"answer":42}` || strings.Join(got, "") != res.Text {
		t.Fatalf("only the last agent_message may be the answer: text=%q deltas=%q", res.Text, got)
	}
	p := &Parser{}
	for _, l := range []string{
		`{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"narration"}}`,
		`{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"ls"}}`,
		`{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"ls"}}`,
		`{"type":"item.completed","item":{"id":"item_2","type":"web_search","query":"x"}}`,
		`{"type":"item.completed","item":{"id":"item_3","type":"agent_message","text":"answer"}}`,
	} {
		if err := p.Feed([]byte(l)); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.Feed([]byte(`{"type":"turn.completed"}`)); !errors.Is(err, clicore.ErrStop) {
		t.Fatalf("turn.completed must stop reading: %v", err)
	}
	if res, err := p.Finish(); err != nil || res.Text != "answer" || strings.Join(p.ToolItems, ",") != "command_execution,web_search" {
		t.Fatalf("res=%+v err=%v tools=%q", res, err, p.ToolItems)
	}
}

func TestFastTierWithoutModelUsesLowEffort(t *testing.T) {
	f := newFixture(t)
	if _, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "hi", Tier: llm.TierFast}); err != nil {
		t.Fatal(err)
	}
	if args := read(t, f.args); !strings.Contains(args, "-c\nmodel_reasoning_effort=\"low\"\n") {
		t.Fatalf("fast tier on the default model must ask for low effort:\n%s", args)
	}
	if _, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "hi", Tier: llm.TierFast, Model: "gpt-mini"}); err != nil {
		t.Fatal(err)
	}
	if args := read(t, f.args); strings.Contains(args, "model_reasoning_effort") {
		t.Fatalf("a configured fast model keeps its own effort:\n%s", args)
	}
}

func TestRequiredSecurityFlagsFailClosed(t *testing.T) {
	for name, help := range map[string]string{
		"no sandbox":  "--json --ignore-user-config --color",
		"no isolate":  "--json --sandbox --color",
		"help failed": "FAIL",
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			if err := os.WriteFile(f.help, []byte(help), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "hi"})
			if clicore.ErrorCode(err) != CodeCLIError || !strings.Contains(err.Error(), "codex") {
				t.Fatalf("err = %v", err)
			}
			if _, statErr := os.Stat(f.args); statErr == nil {
				t.Fatal("codex exec must not run without its isolation flags")
			}
		})
	}
}

func TestTruncatedStreamIsNotClassifiedFromStdout(t *testing.T) {
	f := newFixture(t)
	// Partial generated text mentions 401/429; exit 1 with empty stderr.
	_, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "MODE_LECTURE401"})
	if clicore.ErrorCode(err) != CodeCLIError {
		t.Fatalf("truncated stream = %v (%s), want %s", err, clicore.ErrorCode(err), CodeCLIError)
	}
}

func TestStopsAfterTurnCompleted(t *testing.T) {
	f := newFixture(t)
	start := time.Now()
	res, err := f.client.Stream(context.Background(), "", llm.MessagesRequest{User: "MODE_LINGER"}, nil)
	if err != nil || res.Text != "done" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("a codex that lingers after turn.completed held the call for %s", d)
	}
}

func TestErrorsAreMapped(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	_, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "MODE_AUTH"})
	if !errors.Is(err, llm.ErrGenerationUnavailable) || clicore.ErrorCode(err) != CodeNotLoggedIn {
		t.Fatalf("auth = %v (%s)", err, clicore.ErrorCode(err))
	}
	var le *llm.LLMError
	if !errors.As(err, &le) || le.Provider != "codex" {
		t.Fatalf("provider on error = %+v", le)
	}
	_, err = f.client.Create(ctx, "", llm.MessagesRequest{User: "MODE_RATE"})
	if !errors.Is(err, llm.ErrRateLimited) || clicore.ErrorCode(err) != CodeRateLimited {
		t.Fatalf("rate = %v", err)
	}
	if st := f.client.Status(ctx); st.Code != CodeRateLimited || !st.Ready {
		t.Fatalf("status after rate limit = %+v", st)
	}
	if _, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "MODE_TRUNC"}); err == nil || clicore.ErrorCode(err) != CodeCLIError {
		t.Fatalf("no turn.completed must fail: %v", err)
	}
	if res, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "MODE_RETRY"}); err != nil || res.Text != "after retry" {
		t.Fatalf("a retried error event must not fail a completed turn: %+v %v", res, err)
	}
	start := time.Now()
	_, err = f.client.Create(ctx, "", llm.MessagesRequest{User: "MODE_SLEEP", Timeout: time.Second})
	if !errors.Is(err, llm.ErrTimeout) || clicore.ErrorCode(err) != CodeTimeout || time.Since(start) > 8*time.Second {
		t.Fatalf("timeout = %v after %s", err, time.Since(start))
	}
}

func TestMissingCLI(t *testing.T) {
	c := New(Config{Bin: filepath.Join(t.TempDir(), "no-codex"), TempBase: t.TempDir()})
	_, err := c.Create(context.Background(), "", llm.MessagesRequest{User: "hi"})
	if !errors.Is(err, llm.ErrGenerationUnavailable) || clicore.ErrorCode(err) != CodeCLIMissing {
		t.Fatalf("missing = %v", err)
	}
	if st := c.Status(context.Background()); st.Installed || st.Code != CodeCLIMissing || st.Ready {
		t.Fatalf("status = %+v", st)
	}
}

func TestStatusReadsLoginStatus(t *testing.T) {
	f := newFixture(t)
	st := f.client.Status(context.Background())
	if !st.Installed || st.Version != "codex-cli 9.9.9" || st.LoggedIn == nil || !*st.LoggedIn || !st.Ready || st.Code != "" {
		t.Fatalf("logged-in status = %+v", st)
	}
	if err := os.WriteFile(f.login, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	st = f.client.Status(context.Background())
	if st.LoggedIn == nil || !*st.LoggedIn || read(t, f.count) != "v\nl\n" {
		t.Fatalf("a poll within %s must reuse --version / login status: %+v count=%q", statusTTL, st, read(t, f.count))
	}
	f.client.status.Reset() // as after statusTTL
	st = f.client.Status(context.Background())
	if st.LoggedIn == nil || *st.LoggedIn || st.Code != CodeNotLoggedIn || st.Ready {
		t.Fatalf("logged-out status = %+v", st)
	}
}

func TestLoggedOutCallDropsStatusCache(t *testing.T) {
	f := newFixture(t)
	if st := f.client.Status(context.Background()); st.LoggedIn == nil || !*st.LoggedIn {
		t.Fatalf("status = %+v", st)
	}
	if _, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "MODE_AUTH"}); clicore.ErrorCode(err) != CodeNotLoggedIn {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.login, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if st := f.client.Status(context.Background()); st.LoggedIn == nil || *st.LoggedIn || st.Code != CodeNotLoggedIn {
		t.Fatalf("a logged-out call must re-read login state: %+v", st)
	}
}
