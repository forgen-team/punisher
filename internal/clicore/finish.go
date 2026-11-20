package clicore

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/forgen-team/punisher/internal/llm"
)

// Finish turns a process outcome into a result or APIError. runCtx is the
// call's run context, callerCtx the caller's. A parsed CLI result (even on
// a non-zero exit) is preferred because it carries the reason; a parse error
// that is an APIError with a specific code wins over the stderr text.
// Failure text is classified from stderr only: stdout may be (partial)
// generated text that mentions "401" or "429".
func Finish[T any](runCtx, callerCtx context.Context, codes Codes, runErr error, stderr string, parse func() (T, error)) (T, *APIError) {
	var zero T
	if callerCtx.Err() != nil {
		return zero, NewError(499, CodeClientGone, "caller gave up")
	}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return zero, NewError(http.StatusGatewayTimeout, codes.Timeout, codes.Provider+" call exceeded timeout; process group killed")
	}
	if ArgTooLong(runErr) {
		return zero, NewError(http.StatusBadRequest, CodeInvalid, codes.Provider+" prompt is too long to pass as a command-line argument")
	}
	if errors.Is(runErr, ErrStdoutRead) {
		return zero, NewError(http.StatusBadGateway, codes.CLIError, codes.Provider+" output could not be read (line too long or broken pipe)")
	}
	var execErr *exec.Error
	if errors.As(runErr, &execErr) {
		return zero, NewError(http.StatusServiceUnavailable, codes.Missing, codes.Provider+" executable not runnable: "+execErr.Error())
	}
	var pathErr *os.PathError
	if errors.As(runErr, &pathErr) {
		return zero, NewError(http.StatusServiceUnavailable, codes.Missing, codes.Provider+" executable not runnable: "+pathErr.Error())
	}
	// ErrWaitDelay: the CLI exited but a helper it spawned kept its output
	// open; a complete parsed answer is still a success.
	if errors.Is(runErr, exec.ErrWaitDelay) {
		runErr = nil
	}
	res, err := parse()
	if err == nil && runErr == nil {
		return res, nil
	}
	var ae *APIError
	if err != nil && errors.As(err, &ae) && (ae.Code != codes.CLIError || runErr == nil) {
		return zero, ae
	}
	detail := strings.TrimSpace(stderr)
	if detail == "" && ae != nil {
		return zero, ae // a parse error is already classified; its text is not re-read for markers
	}
	if detail == "" && err != nil {
		detail = err.Error()
	}
	if detail == "" && runErr != nil {
		detail = runErr.Error()
	}
	return zero, codes.Classify(detail)
}

// ArgTooLong reports an exec failure because the arguments (the prompt
// passed as argv) exceed the OS limit: E2BIG, which os/exec wraps in a
// *os.PathError that would otherwise look like a missing executable.
func ArgTooLong(err error) bool {
	return errors.Is(err, syscall.E2BIG) || (err != nil && strings.Contains(err.Error(), "argument list too long"))
}

// MaxArgPromptBytes caps a prompt passed as a single argv entry. Linux
// limits one argument to 128 KiB (MAX_ARG_STRLEN) and macOS all arguments
// plus the environment to 1 MiB (ARG_MAX); 100 KB stays under both.
const MaxArgPromptBytes = 100_000

// PromptFrom flattens a multi-turn request into one prompt.
func PromptFrom(req llm.MessagesRequest) string {
	if len(req.Messages) == 0 {
		return req.User
	}
	var b strings.Builder
	for _, m := range req.Messages {
		fmt.Fprintf(&b, "[%s]\n%s\n\n", m.Role, m.Content)
	}
	if req.User != "" {
		fmt.Fprintf(&b, "[user]\n%s\n", req.User)
	}
	return strings.TrimSpace(b.String())
}

// MissingError is the not-installed error of a provider.
func (c Codes) MissingError(detail string) *APIError {
	return NewError(http.StatusServiceUnavailable, c.Missing, detail)
}

// WithSystem prepends system instructions to a prompt for CLIs without a
// system-prompt flag, with clear delimiters.
func WithSystem(system, prompt string) string {
	system = strings.TrimSpace(system)
	if system == "" {
		return prompt
	}
	return "<system_instructions>\n" + system + "\n</system_instructions>\n\n" + prompt
}
