package claudecli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// Request is one CLI call. The prompt goes to stdin.
type Request struct {
	System   string
	Prompt   string
	JSONMode bool
}

// disallowedTools is a belt-and-braces deny list in addition to `--tools ""`
// (older CLIs without --tools still honour --disallowedTools).
var disallowedTools = []string{
	"Bash", "BashOutput", "KillShell", "Edit", "MultiEdit", "Write", "Read", "Glob", "Grep",
	"NotebookEdit", "WebFetch", "WebSearch", "Task", "Agent", "TodoWrite", "SlashCommand", "Skill",
}

// optionalFlags are dropped when `claude --help` does not list them, so the
// app keeps working on older CLI builds.
var optionalFlags = []string{
	"--system-prompt", "--tools", "--setting-sources", "--strict-mcp-config", "--mcp-config",
	"--no-session-persistence", "--include-partial-messages",
}

const jsonModeInstruction = "Respond with a single valid JSON value only. No prose, no markdown code fences."

var modelPattern = regexp.MustCompile(`^(sonnet|haiku|opus|claude-[a-z0-9.\-]{1,60})$`)

// Runner executes the claude CLI in an isolated, empty working directory.
type Runner struct {
	Bin      string
	TempBase string   // per-call temp dirs are created under it
	Env      []string // full child environment (already filtered)
	// Supported records which optionalFlags the installed CLI accepts.
	Supported map[string]bool
}

// NewRunner builds the child environment and probes optional flags. bin
// must be an executable path; its directory is put first on the child PATH
// so npm-installed CLIs (#!/usr/bin/env node) find their node.
func NewRunner(ctx context.Context, bin, tempBase string) (*Runner, error) {
	if err := os.MkdirAll(tempBase, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir temp base: %w", err)
	}
	r := &Runner{Bin: bin, TempBase: tempBase, Env: childEnv(filepath.Dir(bin))}
	help, err := r.output(ctx, 15*time.Second, "--help")
	r.Supported = map[string]bool{}
	for _, f := range optionalFlags {
		// If help cannot be read, assume support; the CLI will report bad flags.
		r.Supported[f] = err != nil || strings.Contains(help, f)
	}
	return r, nil
}

// childEnv keeps only what the CLI needs (HOME, PATH, LANG, USER, LOGNAME,
// CLAUDE_CONFIG_DIR) and turns off auto-update / non-essential traffic for
// unattended runs. USER/LOGNAME are needed on macOS where the OAuth token
// lives in the login keychain under the account name.
func childEnv(binDir string) []string {
	env := []string{"DISABLE_AUTOUPDATER=1", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1", "DISABLE_TELEMETRY=1"}
	path := os.Getenv("PATH")
	if binDir != "" && binDir != "." {
		path = binDir + string(os.PathListSeparator) + path
	}
	env = append(env, "PATH="+path)
	for _, key := range []string{"HOME", "LANG", "USER", "LOGNAME", "CLAUDE_CONFIG_DIR", "TMPDIR"} {
		if v, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+v)
		}
	}
	return env
}

// Version runs `claude --version`.
func (r *Runner) Version(ctx context.Context) (string, error) {
	out, err := r.output(ctx, 10*time.Second, "--version")
	return strings.TrimSpace(out), err
}

func (r *Runner) output(ctx context.Context, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.Bin, args...)
	cmd.Env = r.Env
	cmd.Dir = r.TempBase
	configureProcessGroup(cmd)
	out, err := cmd.Output()
	return string(out), err
}

// Args builds the CLI argument list (the prompt itself goes to stdin).
func (r *Runner) Args(req Request, model string, stream bool) []string {
	args := []string{"-p", "--model", model, "--max-turns", "1"}
	if stream {
		args = append(args, "--output-format", "stream-json", "--verbose")
		if r.supports("--include-partial-messages") {
			args = append(args, "--include-partial-messages")
		}
	} else {
		args = append(args, "--output-format", "json")
	}
	system := req.System
	if req.JSONMode {
		system = strings.TrimSpace(system + "\n\n" + jsonModeInstruction)
	}
	if system != "" {
		if r.supports("--system-prompt") {
			args = append(args, "--system-prompt", system)
		} else {
			args = append(args, "--append-system-prompt", system)
		}
	}
	if r.supports("--tools") {
		args = append(args, "--tools", "")
	}
	args = append(args, "--disallowedTools", strings.Join(disallowedTools, ","))
	if r.supports("--setting-sources") {
		args = append(args, "--setting-sources", "")
	}
	if r.supports("--strict-mcp-config") && r.supports("--mcp-config") {
		args = append(args, "--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`)
	}
	if r.supports("--no-session-persistence") {
		args = append(args, "--no-session-persistence")
	}
	return args
}

func (r *Runner) supports(flag string) bool {
	if r.Supported == nil {
		return true
	}
	return r.Supported[flag]
}

// Run starts one CLI process with the prompt on stdin. onLine receives each
// stdout line when non-nil; otherwise stdout is buffered and returned. The
// whole process group is killed when ctx ends (timeout or client disconnect).
func (r *Runner) Run(ctx context.Context, args []string, prompt string, onLine func([]byte) error) (stdout []byte, stderr string, err error) {
	workDir, err := os.MkdirTemp(r.TempBase, "call-*")
	if err != nil {
		return nil, "", fmt.Errorf("mkdir call dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	cmd := exec.CommandContext(ctx, r.Bin, args...)
	cmd.Dir = workDir
	cmd.Env = r.Env
	cmd.Stdin = strings.NewReader(prompt)
	var errBuf bytes.Buffer
	cmd.Stderr = &limitedWriter{w: &errBuf, n: 64 << 10}
	configureProcessGroup(cmd)

	if onLine == nil {
		var outBuf bytes.Buffer
		cmd.Stdout = &outBuf
		err = cmd.Run()
		return outBuf.Bytes(), errBuf.String(), err
	}

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", err
	}
	if err := cmd.Start(); err != nil {
		return nil, "", err
	}
	var tail bytes.Buffer
	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 0, 64<<10), 16<<20)
	var cbErr error
	for scanner.Scan() {
		line := scanner.Bytes()
		tail.Reset()
		tail.Write(line)
		if cbErr == nil {
			if cbErr = onLine(line); cbErr != nil {
				killGroup(cmd)
			}
		}
	}
	_, _ = io.Copy(io.Discard, pipe)
	waitErr := cmd.Wait()
	if cbErr != nil {
		return tail.Bytes(), errBuf.String(), cbErr
	}
	return tail.Bytes(), errBuf.String(), waitErr
}

// configureProcessGroup puts the child in its own process group so a
// timeout kills grandchildren too (they would otherwise keep stdout open).
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killGroup(cmd) }
	cmd.WaitDelay = 5 * time.Second
}

func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return cmd.Process.Kill()
	}
	return nil
}

type limitedWriter struct {
	w io.Writer
	n int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.n <= 0 {
		return len(p), nil
	}
	chunk := p
	if len(chunk) > l.n {
		chunk = chunk[:l.n]
	}
	l.n -= len(chunk)
	if _, err := l.w.Write(chunk); err != nil {
		return 0, err
	}
	return len(p), nil
}
