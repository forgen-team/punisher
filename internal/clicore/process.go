package clicore

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
	"strings"
	"sync"
	"syscall"
	"time"
)

// Proc runs one CLI executable in an isolated, empty working directory with
// a filtered environment.
type Proc struct {
	Bin      string
	TempBase string   // per-call temp dirs are created under it
	Env      []string // full child environment (already filtered)
}

// NewProc creates TempBase and builds the child environment: fixed entries,
// PATH with the executable's directory first (npm-installed CLIs use
// #!/usr/bin/env node) and the keep variables that are set.
func NewProc(bin, tempBase string, fixed []string, keep ...string) (Proc, error) {
	if err := os.MkdirAll(tempBase, 0o700); err != nil {
		return Proc{}, fmt.Errorf("mkdir temp base: %w", err)
	}
	return Proc{Bin: bin, TempBase: tempBase, Env: ChildEnv(filepath.Dir(bin), fixed, keep...)}, nil
}

// ChildEnv is fixed + PATH (binDir first) + the keep variables that are set.
func ChildEnv(binDir string, fixed []string, keep ...string) []string {
	env := append([]string{}, fixed...)
	path := os.Getenv("PATH")
	if binDir != "" && binDir != "." {
		path = binDir + string(os.PathListSeparator) + path
	}
	env = append(env, "PATH="+path)
	for _, key := range keep {
		if v, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+v)
		}
	}
	return env
}

// BaseEnvKeys are the variables every CLI needs. USER/LOGNAME are needed on
// macOS where OAuth tokens live in the login keychain under the account name.
// The proxy / CA variables let the CLIs reach their service behind a
// corporate proxy or TLS-inspecting firewall; XDG_CONFIG_HOME is where some
// CLIs keep their login. API key variables are never passed: every provider
// runs on its own subscription login.
var BaseEnvKeys = []string{"HOME", "LANG", "USER", "LOGNAME", "TMPDIR",
	"HTTPS_PROXY", "HTTP_PROXY", "NO_PROXY", "https_proxy", "http_proxy", "no_proxy",
	"SSL_CERT_FILE", "SSL_CERT_DIR", "NODE_EXTRA_CA_CERTS", "XDG_CONFIG_HOME"}

// Output runs the CLI with args (no stdin) and returns stdout.
func (p Proc) Output(ctx context.Context, timeout time.Duration, args ...string) (string, error) {
	out, _, err := p.Outputs(ctx, timeout, args...)
	return out, err
}

// Outputs runs the CLI with args (no stdin) and returns stdout and stderr.
func (p Proc) Outputs(ctx context.Context, timeout time.Duration, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, p.Bin, args...)
	cmd.Env = p.Env
	cmd.Dir = p.TempBase
	ConfigureProcessGroup(cmd)
	var errBuf bytes.Buffer
	cmd.Stderr = &limitedWriter{w: &errBuf, n: 64 << 10}
	out, err := cmd.Output()
	return string(out), errBuf.String(), err
}

// HelpSupports probes help (args...) for the given flags. When help cannot
// be read every flag is assumed supported (the CLI will report bad flags).
func (p Proc) HelpSupports(ctx context.Context, flags []string, args ...string) map[string]bool {
	out, _ := p.ProbeHelp(ctx, flags, args...)
	return out
}

// ProbeHelp is HelpSupports that also reports whether help could be read:
// on error every flag is marked supported and err is non-nil, so callers
// can refuse (required flags) or re-probe later instead of trusting it.
func (p Proc) ProbeHelp(ctx context.Context, flags []string, args ...string) (map[string]bool, error) {
	help, err := p.Output(ctx, 15*time.Second, args...)
	out := map[string]bool{}
	for _, f := range flags {
		out[f] = err != nil || strings.Contains(help, f)
	}
	if err != nil {
		return out, fmt.Errorf("%s %s: %w", filepath.Base(p.Bin), strings.Join(args, " "), err)
	}
	return out, nil
}

// ErrStop is returned by a Run onLine callback once the CLI's final event
// (result / turn.completed) has been read: Run stops reading, kills the
// process group and reports success. A CLI (or a helper it spawned, e.g. an
// MCP server) that keeps stdout open after its answer then cannot hold the
// call until the timeout.
var ErrStop = errors.New("clicore: final event read")

// ErrStdoutRead marks a stdout read failure (e.g. a line over the 16 MiB
// scanner limit): the output is incomplete and must not be parsed as a result.
var ErrStdoutRead = errors.New("reading CLI output failed")

// Run starts one CLI process with stdin as its input. onLine receives each
// stdout line when non-nil; otherwise stdout is buffered and returned (with
// onLine the last line is returned). The whole process group is killed when
// ctx ends (timeout or client disconnect) and when onLine returns ErrStop
// (then err is nil). A stdout read error is returned wrapping ErrStdoutRead.
func (p Proc) Run(ctx context.Context, args []string, stdin string, onLine func([]byte) error) (stdout []byte, stderr string, err error) {
	workDir, err := os.MkdirTemp(p.TempBase, "call-*")
	if err != nil {
		return nil, "", fmt.Errorf("mkdir call dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	cmd := exec.CommandContext(ctx, p.Bin, args...)
	cmd.Dir = workDir
	cmd.Env = p.Env
	cmd.Stdin = strings.NewReader(stdin)
	var errBuf bytes.Buffer
	cmd.Stderr = &limitedWriter{w: &errBuf, n: 64 << 10}
	ConfigureProcessGroup(cmd)

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
		if cbErr = onLine(line); cbErr != nil {
			break
		}
	}
	scanErr := scanner.Err()
	if cbErr != nil || scanErr != nil {
		killGroupTwice(cmd)
	}
	if !errors.Is(cbErr, ErrStop) {
		_, _ = io.Copy(io.Discard, pipe)
	}
	waitErr := cmd.Wait()
	switch {
	case errors.Is(cbErr, ErrStop):
		// The answer is complete; the exit status of the killed group is irrelevant.
		return tail.Bytes(), errBuf.String(), nil
	case cbErr != nil:
		return tail.Bytes(), errBuf.String(), cbErr
	case scanErr != nil:
		return tail.Bytes(), errBuf.String(), fmt.Errorf("%w: %w", ErrStdoutRead, scanErr)
	}
	return tail.Bytes(), errBuf.String(), waitErr
}

// ConfigureProcessGroup puts the child in its own process group so a
// timeout kills grandchildren too (they would otherwise keep stdout open).
func ConfigureProcessGroup(cmd *exec.Cmd) {
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

// killGroupTwice kills the group, then again shortly after: a child forked
// while the first signal was delivered escapes it and would keep stdout or
// stderr open until WaitDelay. The leader is not reaped yet (Wait comes
// later), so its group id cannot have been reused in between.
func killGroupTwice(cmd *exec.Cmd) {
	_ = killGroup(cmd)
	time.Sleep(20 * time.Millisecond)
	_ = killGroup(cmd)
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

// ResolveBin finds an executable: an explicit path/name first, then name on
// PATH, then the candidate locations (a GUI-launched process may have a
// minimal PATH) and nvmGlob matches. Returns "" when nothing is found.
func ResolveBin(explicit, name string, candidates []string) string {
	if explicit != "" {
		if p, err := exec.LookPath(explicit); err == nil {
			return p
		}
		return ""
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	if matches, err := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin", name)); err == nil {
		candidates = append(candidates, matches...)
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return c
		}
	}
	return ""
}

// CommonBinCandidates are the usual install locations of name under home.
func CommonBinCandidates(name string) []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".local", "bin", name),
		"/opt/homebrew/bin/" + name,
		"/usr/local/bin/" + name,
		filepath.Join(home, ".npm-global", "bin", name),
		filepath.Join(home, ".volta", "bin", name),
		filepath.Join(home, "bin", name),
	}
}

// incompleteTTL is how long a value whose probe failed (help unreadable) is
// reused before it is built again (a var for tests).
var incompleteTTL = 30 * time.Second

// BinCache keeps a resolved executable's per-binary state (probed flags…),
// re-detecting when it was not found before or has been removed since.
type BinCache[T any] struct {
	mu      sync.Mutex
	bin     string
	val     T
	have    bool
	expires time.Time // zero = until the executable disappears
}

// Get returns the cached value while its executable still exists, else
// resolves again (resolve "" = not found → IsBinNotFound error). build
// reports complete=false when its probe failed: such a value is reused only
// for a short while, then built (probed) again.
func (c *BinCache[T]) Get(resolve func() string, build func(bin string) (val T, complete bool, err error)) (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero T
	if c.have {
		_, statErr := os.Stat(c.bin)
		if statErr == nil && (c.expires.IsZero() || time.Now().Before(c.expires)) {
			return c.val, nil
		}
		c.val, c.have, c.expires = zero, false, time.Time{}
	}
	bin := resolve()
	if bin == "" {
		return zero, errBinNotFound
	}
	v, complete, err := build(bin)
	if err != nil {
		return zero, err
	}
	c.bin, c.val, c.have = bin, v, true
	if !complete {
		c.expires = time.Now().Add(incompleteTTL)
	}
	return v, nil
}

var errBinNotFound = errors.New("executable not found")

// IsBinNotFound reports a BinCache miss.
func IsBinNotFound(err error) bool { return errors.Is(err, errBinNotFound) }
