package agycli

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

// fakeAgy follows the documented headless contract (antigravity.google/docs/cli/headless).
// The prompt is the -p value; its content selects the behaviour. printf, not
// echo: the JSON carries literal \n escapes that sh echo would expand.
const fakeAgy = `#!/bin/sh
case "$1" in
  --help)
    if [ -f "__HELPFAIL__" ]; then exit 1; fi
    echo "-p, --print, --prompt --output-format --model --effort --print-timeout --sandbox --disable-slash-commands"; exit 0;;
  --version) echo v >> "__COUNT__"; echo "agy 1.9.9"; exit 0;;
esac
printf '%s\n' "$@" > "__ARGS__"
prompt="$2"
format=text
prev=""
for a in "$@"; do
  if [ "$prev" = "--output-format" ]; then format="$a"; fi
  prev="$a"
done
case "$prompt" in
  *MODE_SLEEP*) sleep 30; exit 0;;
  *MODE_AUTH*) echo "Error: authentication required. Run agy to sign in." >&2; exit 1;;
  *MODE_QUOTA*) echo '{"conversation_id":"","status":"ERROR","response":"","error":"RESOURCE_EXHAUSTED: quota exceeded"}'; exit 1;;
  *MODE_TRUNC*) echo '{"event":"step_update","step_update":{"step_type":"agent_response","state":"ACTIVE","text_delta":"par"}}'; exit 0;;
  *MODE_LECTURE401*) echo 'A 401 means unauthorized; a 429 means too many requests.'; exit 1;;
  *MODE_PRETTY*) printf '%s\n' 'warming up' '{' '  "status": "SUCCESS",' '  "response": "pretty",' '  "usage": {"input_tokens": 3, "output_tokens": 1}' '}'; exit 0;;
  *MODE_LINGER*)
    (sleep 30) &
    sleep 0.2
    printf '%s\n' '{"event":"result","result":{"status":"SUCCESS","response":"done"}}'
    sleep 30; exit 0;;
esac
if [ "$format" = "stream-json" ]; then
  printf '%s\n' '{"event":"init","conversation_id":"c","init":{"cwd":"/x","tools":["run_command"],"permission_mode":"request-review","model":"gemini-x"}}'
  printf '%s\n' '{"event":"step_update","step_update":{"step_index":0,"state":"DONE","step_type":"user_input"}}'
  printf '%s\n' '{"event":"step_update","step_update":{"step_index":2,"state":"ACTIVE","step_type":"agent_response","text_delta":"Hel"}}'
  printf '%s\n' '{"event":"step_update","step_update":{"step_index":2,"state":"DONE","step_type":"agent_response","text_delta":"lo\n"}}'
  printf '%s\n' '{"event":"result","result":{"conversation_id":"c","status":"SUCCESS","response":"Hello\n","num_turns":1,"usage":{"input_tokens":50,"output_tokens":4,"thinking_tokens":2,"cache_read_tokens":30,"total_tokens":54}}}'
  exit 0
fi
printf '%s\n' '{"conversation_id":"c","status":"SUCCESS","response":"ok\n","duration_seconds":1.2,"num_turns":1,"usage":{"input_tokens":10415,"output_tokens":657,"thinking_tokens":616,"cache_read_tokens":8113,"total_tokens":11072}}'
`

type fixture struct {
	client                *Client
	args, helpFail, count string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	f := &fixture{args: filepath.Join(dir, "args"), helpFail: filepath.Join(dir, "helpfail"), count: filepath.Join(dir, "count")}
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "agy"), []byte(strings.NewReplacer("__ARGS__", f.args, "__HELPFAIL__", f.helpFail, "__COUNT__", f.count).Replace(fakeAgy)), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	f.client = New(Config{TempBase: filepath.Join(dir, "tmp"), Gate: clicore.NewGate(clicore.GateConfig{Concurrency: 1, DefaultTimeout: 10 * time.Second})})
	return f
}

func TestCreateUsesHeadlessJSON(t *testing.T) {
	f := newFixture(t)
	res, err := f.client.Create(context.Background(), "", llm.MessagesRequest{System: "SYS", User: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "ok\n" || res.Model != DefaultModelName || res.Usage.InputTokens != 10415 || res.Usage.OutputTokens != 657 {
		t.Fatalf("result = %+v", res)
	}
	b, _ := os.ReadFile(f.args)
	args := string(b)
	if !strings.HasPrefix(args, "-p\n<system_instructions>\nSYS\n") || !strings.Contains(args, "\nhello\n") {
		t.Fatalf("prompt must be the -p value with the system block first:\n%s", args)
	}
	for _, want := range []string{"--output-format\njson\n", "--print-timeout\n10s\n", "--sandbox\n", "--disable-slash-commands\n"} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q:\n%s", want, args)
		}
	}
	if strings.Contains(args, "--model") {
		t.Errorf("no --model without a configured model:\n%s", args)
	}
	if _, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "hi", Model: "gemini-3.8-flash-medium"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(f.args); !strings.Contains(string(b), "--model\ngemini-3.8-flash-medium\n") {
		t.Fatalf("model flag:\n%s", b)
	}
}

func TestStreamDeliversTextDeltas(t *testing.T) {
	f := newFixture(t)
	var got []string
	res, err := f.client.Stream(context.Background(), "", llm.MessagesRequest{User: "stream"}, func(d string) error {
		got = append(got, d)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, "|") != "Hel|lo\n" || res.Text != "Hello\n" || res.Model != "gemini-x" || res.Usage.InputTokens != 50 {
		t.Fatalf("deltas=%q result=%+v", got, res)
	}
	if b, _ := os.ReadFile(f.args); !strings.Contains(string(b), "--output-format\nstream-json\n") {
		t.Fatalf("stream args:\n%s", b)
	}
}

func TestErrorsAreMapped(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	_, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "MODE_AUTH"})
	if !errors.Is(err, llm.ErrGenerationUnavailable) || clicore.ErrorCode(err) != CodeNotLoggedIn {
		t.Fatalf("auth = %v", err)
	}
	if st := f.client.Status(ctx); st.LoggedIn != nil || st.Code != CodeNotLoggedIn || st.Ready || st.Version != "agy 1.9.9" {
		t.Fatalf("status after auth failure = %+v", st)
	}
	_, err = f.client.Create(ctx, "", llm.MessagesRequest{User: "MODE_QUOTA"})
	if !errors.Is(err, llm.ErrRateLimited) || clicore.ErrorCode(err) != CodeRateLimited {
		t.Fatalf("quota = %v", err)
	}
	if _, err := f.client.Stream(ctx, "", llm.MessagesRequest{User: "MODE_TRUNC"}, nil); err == nil {
		t.Fatal("a stream without a result event must fail")
	}
	_, err = f.client.Create(ctx, "", llm.MessagesRequest{User: "MODE_SLEEP", Timeout: time.Second})
	if !errors.Is(err, llm.ErrTimeout) || clicore.ErrorCode(err) != CodeTimeout {
		t.Fatalf("timeout = %v", err)
	}
	if _, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "fine"}); err != nil {
		t.Fatal(err)
	}
	if st := f.client.Status(ctx); st.Code != "" || !st.Ready || st.LastFailure != nil {
		t.Fatalf("success must clear the failure: %+v", st)
	}
}

func TestMissingCLI(t *testing.T) {
	c := New(Config{Bin: filepath.Join(t.TempDir(), "no-agy"), TempBase: t.TempDir()})
	_, err := c.Create(context.Background(), "", llm.MessagesRequest{User: "hi"})
	if !errors.Is(err, llm.ErrGenerationUnavailable) || clicore.ErrorCode(err) != CodeCLIMissing {
		t.Fatalf("missing = %v", err)
	}
	if st := c.Status(context.Background()); st.Installed || st.Code != CodeCLIMissing {
		t.Fatalf("status = %+v", st)
	}
}

func TestLongPromptIsInvalidRequestBeforeSpawning(t *testing.T) {
	f := newFixture(t)
	_, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: strings.Repeat("x", clicore.MaxArgPromptBytes+1)})
	if !errors.Is(err, llm.ErrInvalidRequest) || clicore.ErrorCode(err) != clicore.CodeInvalid {
		t.Fatalf("long prompt = %v", err)
	}
	if _, statErr := os.Stat(f.args); statErr == nil {
		t.Fatal("agy must not be started with an over-long argv prompt")
	}
}

func TestFailuresAreNotClassifiedFromStdout(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if _, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "MODE_LECTURE401"}); clicore.ErrorCode(err) != CodeCLIError {
		t.Fatalf("stdout text with 401/429 = %v", err)
	}
	if _, err := f.client.Stream(ctx, "", llm.MessagesRequest{User: "MODE_TRUNC"}, nil); clicore.ErrorCode(err) != CodeCLIError {
		t.Fatalf("truncated stream = %v", err)
	}
}

func TestParseJSONWholeOutputThenLastLine(t *testing.T) {
	f := newFixture(t)
	res, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "MODE_PRETTY"})
	if err == nil {
		t.Fatalf("leading log line + pretty JSON is neither one envelope nor a last-line envelope: %+v", res)
	}
	pretty := "{\n  \"status\": \"SUCCESS\",\n  \"response\": \"multi\"\n}\n"
	if r, err := ParseJSON([]byte(pretty)); err != nil || r.Text != "multi" {
		t.Fatalf("pretty-printed envelope: %+v %v", r, err)
	}
	if r, err := ParseJSON([]byte("log line\n{\"status\":\"SUCCESS\",\"response\":\"last\"}")); err != nil || r.Text != "last" {
		t.Fatalf("last-line envelope: %+v %v", r, err)
	}
}

func TestStopsAfterResultEvent(t *testing.T) {
	f := newFixture(t)
	start := time.Now()
	res, err := f.client.Stream(context.Background(), "", llm.MessagesRequest{User: "MODE_LINGER"}, nil)
	if err != nil || res.Text != "done" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("an agy that lingers after its result held the call for %s", d)
	}
}

func TestUnreadableHelpIsRefused(t *testing.T) {
	f := newFixture(t)
	if err := os.WriteFile(f.helpFail, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the first caller is already gone: the probe must not inherit that
	if _, err := f.client.Create(ctx, "", llm.MessagesRequest{User: "hi"}); err == nil {
		t.Fatal("cancelled caller must fail")
	}
	_, err := f.client.Create(context.Background(), "", llm.MessagesRequest{User: "hi"})
	if clicore.ErrorCode(err) != CodeCLIError || !strings.Contains(err.Error(), "--help") {
		t.Fatalf("unreadable help must refuse, not assume every flag: %v", err)
	}
	if _, statErr := os.Stat(f.args); statErr == nil {
		t.Fatal("agy must not run with guessed flags")
	}
}

func TestStatusCachesVersion(t *testing.T) {
	f := newFixture(t)
	for range 3 {
		if st := f.client.Status(context.Background()); st.Version != "agy 1.9.9" {
			t.Fatalf("status = %+v", st)
		}
	}
	if b, _ := os.ReadFile(f.count); string(b) != "v\n" {
		t.Fatalf("--version ran %q times within %s", b, statusTTL)
	}
}

func TestOldCLIWithoutFlagsSkipsThem(t *testing.T) {
	r := &runner{supported: map[string]bool{"--output-format": true}}
	args := strings.Join(r.Args("P", "m", false, time.Minute), " ")
	if args != "-p P --output-format json" {
		t.Fatalf("unknown flags must be skipped: %q", args)
	}
}
