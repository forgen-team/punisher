package claudecli

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/clicore"
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
	clicore.Proc
	// Supported records which optionalFlags the installed CLI accepts.
	Supported map[string]bool
	probeErr  error // help could not be read (Supported is then all true)
}

// NewRunner builds the child environment and probes optional flags. bin
// must be an executable path; its directory is put first on the child PATH
// so npm-installed CLIs (#!/usr/bin/env node) find their node.
func NewRunner(ctx context.Context, bin, tempBase string) (*Runner, error) {
	proc, err := clicore.NewProc(bin, tempBase, childEnvFixed, slices.Concat(clicore.BaseEnvKeys, []string{"CLAUDE_CONFIG_DIR"})...)
	if err != nil {
		return nil, err
	}
	r := &Runner{Proc: proc}
	// Unreadable help keeps the old behaviour (every flag assumed supported)
	// but is not cached forever: the client re-probes after a short while.
	r.Supported, r.probeErr = proc.ProbeHelp(context.WithoutCancel(ctx), optionalFlags, "--help")
	return r, nil
}

// childEnvFixed turns off auto-update / non-essential traffic for unattended
// runs. The child also gets HOME, PATH, LANG, USER, LOGNAME, TMPDIR and
// CLAUDE_CONFIG_DIR (clicore.ChildEnv).
var childEnvFixed = []string{"DISABLE_AUTOUPDATER=1", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1", "DISABLE_TELEMETRY=1"}

// Version runs `claude --version`.
func (r *Runner) Version(ctx context.Context) (string, error) {
	out, err := r.Output(ctx, 10*time.Second, "--version")
	return strings.TrimSpace(out), err
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
