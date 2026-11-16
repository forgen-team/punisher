package lecture

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Lab command policy (v3 design §4.2). A @@lab block shows one command the
// reader runs in their own terminal; the app never runs it. Only a single
// simple command of a read-only tool with allow-listed flags passes: no
// operators, pipes, redirects, substitutions, assignments or globs, and no
// path that can hold secrets. Everything not listed is rejected.

// Lab target systems (block attribute os=…).
const (
	LabMacOS = "macos"
	LabLinux = "linux"
	LabAny   = "any"
)

// ValidLabOS reports a known os attribute.
func ValidLabOS(os string) bool { return os == LabMacOS || os == LabLinux || os == LabAny }

var (
	reNum     = regexp.MustCompile(`^\d{1,7}$`)
	reHost    = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,252})$`)
	reSysctl  = regexp.MustCompile(`^[a-z][a-z0-9_.]{1,80}$`)
	reGetconf = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,40}$`)
	reURL     = regexp.MustCompile(`^https?://[A-Za-z0-9.-]+(:\d{1,5})?(/[A-Za-z0-9._~%/?=&-]*)?$`)
	reProc    = regexp.MustCompile(`^/proc/(loadavg|meminfo|cpuinfo|stat|uptime|version|vmstat|net/[a-z0-9_]+|sys/[a-z0-9_/]+)$`)
	rePort    = regexp.MustCompile(`^:\d{1,5}$`)
)

// secretPath: fragments of paths that may hold credentials.
var secretPath = []string{".ssh", ".aws", ".gnupg", ".config/gh", ".netrc", ".env", "shadow", "id_rsa", "id_ed25519",
	".kube", ".docker", "keychain", "environ", "cmdline", ".npmrc", ".pypirc", ".git-credentials", "credentials", "token", "secret"}

var safePathPrefix = []string{".", "~", "/tmp", "/var/log", "/usr", "/etc/hosts", "/etc/resolv.conf", "/etc/os-release", "/Applications", "/opt"}

type toolPolicy struct {
	os    []string // systems that have the tool
	check func(args []string) error
}

var labTools = map[string]toolPolicy{
	"lsof":       {os: []string{LabMacOS, LabLinux}, check: checkLsof},
	"ss":         {os: []string{LabLinux}, check: shortFlags("tulnpasx46")},
	"netstat":    {os: []string{LabMacOS, LabLinux}, check: checkNetstat},
	"ps":         {os: []string{LabMacOS, LabLinux}, check: checkPs},
	"vm_stat":    {os: []string{LabMacOS}, check: noArgs},
	"free":       {os: []string{LabLinux}, check: oneOf("-h", "-m", "-g", "-k")},
	"uptime":     {os: []string{LabMacOS, LabLinux}, check: noArgs},
	"nproc":      {os: []string{LabLinux}, check: noArgs},
	"uname":      {os: []string{LabMacOS, LabLinux}, check: oneOf("-a", "-m", "-r", "-s")},
	"getconf":    {os: []string{LabMacOS, LabLinux}, check: single(reGetconf)},
	"ulimit":     {os: []string{LabMacOS, LabLinux}, check: oneOf("-a", "-n", "-u")},
	"sysctl":     {os: []string{LabMacOS, LabLinux}, check: checkSysctl},
	"df":         {os: []string{LabMacOS, LabLinux}, check: flagsThenPaths("hkiP", 1)},
	"du":         {os: []string{LabMacOS, LabLinux}, check: flagsThenPaths("shkc", 1)},
	"ls":         {os: []string{LabMacOS, LabLinux}, check: flagsThenPaths("lahd1", 3)},
	"stat":       {os: []string{LabMacOS, LabLinux}, check: flagsThenPaths("", 3)},
	"file":       {os: []string{LabMacOS, LabLinux}, check: flagsThenPaths("", 3)},
	"cat":        {os: []string{LabLinux}, check: checkCatProc},
	"dig":        {os: []string{LabMacOS, LabLinux}, check: checkDig},
	"nslookup":   {os: []string{LabMacOS, LabLinux}, check: hostThen(reRecordOrNone)},
	"host":       {os: []string{LabMacOS, LabLinux}, check: hostThen(reRecordOrNone)},
	"ping":       {os: []string{LabMacOS, LabLinux}, check: checkPing},
	"traceroute": {os: []string{LabMacOS, LabLinux}, check: checkTraceroute},
	"curl":       {os: []string{LabMacOS, LabLinux}, check: checkCurl},
}

var reRecordOrNone = regexp.MustCompile(`^(A|AAAA|MX|NS|TXT|CNAME|SOA|PTR|-type=[A-Z]+)$`)

// CheckLabCommand validates one lab command for a target system.
func CheckLabCommand(cmd, os string) error {
	if !ValidLabOS(os) {
		return fmt.Errorf("lab os %q: want macos, linux or any", os)
	}
	args, err := simpleCommand(cmd)
	if err != nil {
		return err
	}
	p, ok := labTools[args[0]]
	if !ok {
		return fmt.Errorf("lab tool %q is not on the read-only list", args[0])
	}
	for _, want := range targetSystems(os) {
		if !contains(p.os, want) {
			return fmt.Errorf("lab tool %q is not available on %s", args[0], want)
		}
	}
	if err := p.check(args[1:]); err != nil {
		return fmt.Errorf("lab %s: %w", args[0], err)
	}
	return nil
}

func targetSystems(os string) []string {
	if os == LabAny {
		return []string{LabMacOS, LabLinux}
	}
	return []string{os}
}

// simpleCommand parses cmd and returns its words when it is exactly one plain
// command: literal or single/double-quoted literal words only.
func simpleCommand(cmd string) ([]string, error) {
	cmd = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(cmd), "$ "))
	if cmd == "" || strings.ContainsAny(cmd, "\n\r") {
		return nil, errors.New("lab command must be one line")
	}
	f, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
	if err != nil {
		return nil, fmt.Errorf("lab command does not parse: %w", err)
	}
	if len(f.Stmts) != 1 {
		return nil, errors.New("lab command must be a single command")
	}
	st := f.Stmts[0]
	if st.Background || st.Coprocess || st.Negated || len(st.Redirs) > 0 {
		return nil, errors.New("lab command must not use &, !, or redirects")
	}
	call, ok := st.Cmd.(*syntax.CallExpr)
	if !ok || len(call.Assigns) > 0 || len(call.Args) == 0 {
		return nil, errors.New("lab command must be a plain command (no pipes, lists, subshells or assignments)")
	}
	var out []string
	for _, w := range call.Args {
		s, err := literalWord(w)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func literalWord(w *syntax.Word) (string, error) {
	var b strings.Builder
	for _, part := range w.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			if strings.ContainsAny(p.Value, "*?[]{}") {
				return "", errors.New("lab command must not use globs or braces")
			}
			b.WriteString(p.Value)
		case *syntax.SglQuoted:
			if p.Dollar {
				return "", errors.New("lab command must not use $'…'")
			}
			b.WriteString(p.Value)
		case *syntax.DblQuoted:
			for _, q := range p.Parts {
				lit, ok := q.(*syntax.Lit)
				if !ok {
					return "", errors.New("lab command must not expand variables or commands")
				}
				b.WriteString(lit.Value)
			}
		default:
			return "", errors.New("lab command must not expand variables, commands or arithmetic")
		}
	}
	return b.String(), nil
}

func noArgs(args []string) error {
	if len(args) > 0 {
		return errors.New("takes no arguments")
	}
	return nil
}

func oneOf(allowed ...string) func([]string) error {
	return func(args []string) error {
		for _, a := range args {
			if !contains(allowed, a) {
				return fmt.Errorf("argument %q not allowed", a)
			}
		}
		if len(args) > 1 {
			return errors.New("one flag at most")
		}
		return nil
	}
}

func single(re *regexp.Regexp) func([]string) error {
	return func(args []string) error {
		if len(args) != 1 || !re.MatchString(args[0]) {
			return errors.New("takes one name")
		}
		return nil
	}
}

// shortFlags accepts only -abc style flags built from letters, no values.
func shortFlags(letters string) func([]string) error {
	return func(args []string) error {
		for _, a := range args {
			if !isShortFlags(a, letters) {
				return fmt.Errorf("argument %q not allowed", a)
			}
		}
		return nil
	}
}

func isShortFlags(a, letters string) bool {
	if len(a) < 2 || a[0] != '-' || a[1] == '-' {
		return false
	}
	for _, r := range a[1:] {
		if !strings.ContainsRune(letters, r) {
			return false
		}
	}
	return true
}

func flagsThenPaths(letters string, maxPaths int) func([]string) error {
	return func(args []string) error {
		paths := 0
		for _, a := range args {
			if strings.HasPrefix(a, "-") {
				if letters == "" || !isShortFlags(a, letters) {
					return fmt.Errorf("flag %q not allowed", a)
				}
				continue
			}
			if err := safePath(a); err != nil {
				return err
			}
			paths++
		}
		if paths > maxPaths {
			return fmt.Errorf("at most %d paths", maxPaths)
		}
		return nil
	}
}

func safePath(p string) error {
	low := strings.ToLower(p)
	for _, s := range secretPath {
		if strings.Contains(low, s) {
			return fmt.Errorf("path %q may hold secrets", p)
		}
	}
	if strings.Contains(p, "..") {
		return fmt.Errorf("path %q must not climb with ..", p)
	}
	for _, pre := range safePathPrefix {
		if p == pre || strings.HasPrefix(p, pre+"/") || (pre == "." && p == ".") {
			return nil
		}
	}
	return fmt.Errorf("path %q is outside the allowed places", p)
}

func checkLsof(args []string) error {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case isShortFlags(a, "nP"), strings.HasPrefix(a, "-i") && !strings.ContainsAny(a[2:], " "), strings.HasPrefix(a, "-sTCP:"):
		case a == "-p" || a == "-c":
			if i+1 >= len(args) || (a == "-p" && !reNum.MatchString(args[i+1])) || (a == "-c" && !reHost.MatchString(args[i+1])) {
				return fmt.Errorf("%s needs a value", a)
			}
			i++
		case rePort.MatchString(a):
		default:
			return fmt.Errorf("argument %q not allowed", a)
		}
	}
	return nil
}

func checkNetstat(args []string) error {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case isShortFlags(a, "anrsvlt"):
		case a == "-p" || a == "-f":
			if i+1 >= len(args) || !contains([]string{"tcp", "udp", "inet", "inet6"}, args[i+1]) {
				return fmt.Errorf("%s needs tcp, udp, inet or inet6", a)
			}
			i++
		default:
			return fmt.Errorf("argument %q not allowed", a)
		}
	}
	return nil
}

var psColumns = []string{"pid", "ppid", "comm", "%cpu", "%mem", "rss", "vsz", "state", "stat", "etime", "nlwp", "user"}

// checkPs allows -e/-A, -p <pid> and -o with listed columns only (no args/command).
func checkPs(args []string) error {
	hasO := false
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "-e", "-A":
		case "-p":
			if i+1 >= len(args) || !reNum.MatchString(args[i+1]) {
				return errors.New("-p needs a pid")
			}
			i++
		case "-o":
			if i+1 >= len(args) {
				return errors.New("-o needs columns")
			}
			for _, c := range strings.Split(args[i+1], ",") {
				if !contains(psColumns, strings.TrimSuffix(c, "=")) {
					return fmt.Errorf("column %q not allowed", c)
				}
			}
			hasO = true
			i++
		default:
			return fmt.Errorf("argument %q not allowed", a)
		}
	}
	if !hasO {
		return errors.New("needs -o with listed columns")
	}
	return nil
}

func checkSysctl(args []string) error {
	names := 0
	for _, a := range args {
		switch {
		case a == "-n":
		case reSysctl.MatchString(a):
			names++
		default:
			return fmt.Errorf("argument %q not allowed (read one name, no =)", a)
		}
	}
	if names != 1 {
		return errors.New("reads exactly one name")
	}
	return nil
}

func checkCatProc(args []string) error {
	if len(args) != 1 || !reProc.MatchString(args[0]) {
		return errors.New("reads one listed /proc file only")
	}
	return nil
}

func checkDig(args []string) error {
	hosts := 0
	for _, a := range args {
		switch {
		case contains([]string{"+short", "+trace", "+noall", "+answer", "+stats"}, a):
		case strings.HasPrefix(a, "@") && reHost.MatchString(a[1:]):
		case reRecordOrNone.MatchString(a):
		case reHost.MatchString(a):
			hosts++
		default:
			return fmt.Errorf("argument %q not allowed", a)
		}
	}
	if hosts != 1 {
		return errors.New("needs one domain")
	}
	return nil
}

func hostThen(extra *regexp.Regexp) func([]string) error {
	return func(args []string) error {
		hosts := 0
		for _, a := range args {
			switch {
			case extra.MatchString(a):
			case reHost.MatchString(a):
				hosts++
			default:
				return fmt.Errorf("argument %q not allowed", a)
			}
		}
		if hosts < 1 || hosts > 2 {
			return errors.New("needs a domain (and optionally a server)")
		}
		return nil
	}
}

func countArg(args []string, i int, flag string, maxN int) (int, error) {
	if i+1 >= len(args) {
		return 0, fmt.Errorf("%s needs a number", flag)
	}
	n, err := strconv.Atoi(args[i+1])
	if err != nil || n < 1 || n > maxN {
		return 0, fmt.Errorf("%s must be 1~%d", flag, maxN)
	}
	return n, nil
}

func checkPing(args []string) error {
	counted, hosts := false, 0
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-c":
			if _, err := countArg(args, i, "-c", 5); err != nil {
				return err
			}
			counted = true
			i++
		case a == "-n" || a == "-q":
		case reHost.MatchString(a):
			hosts++
		default:
			return fmt.Errorf("argument %q not allowed", a)
		}
	}
	if !counted || hosts != 1 {
		return errors.New("needs -c (≤5) and one host")
	}
	return nil
}

func checkTraceroute(args []string) error {
	hosts := 0
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-m":
			if _, err := countArg(args, i, "-m", 20); err != nil {
				return err
			}
			i++
		case a == "-n":
		case reHost.MatchString(a):
			hosts++
		default:
			return fmt.Errorf("argument %q not allowed", a)
		}
	}
	if hosts != 1 {
		return errors.New("needs one host")
	}
	return nil
}

var curlFlags = []string{"-I", "--head", "-s", "-S", "-sS", "-sI", "-Is", "-v", "-L", "-sL", "--http1.1", "--http2"}

// checkCurl allows a read-only request to one http(s) URL: head/verbose/silent
// flags, -o /dev/null, -w with a format, --max-time. No data, upload, headers,
// config files, credentials or output files.
func checkCurl(args []string) error {
	urls := 0
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case contains(curlFlags, a):
		case a == "-o":
			if i+1 >= len(args) || args[i+1] != "/dev/null" {
				return errors.New("-o only to /dev/null")
			}
			i++
		case a == "-w":
			if i+1 >= len(args) || strings.ContainsAny(args[i+1], "@`$") {
				return errors.New("-w needs a plain format")
			}
			i++
		case a == "--max-time" || a == "-m":
			if _, err := countArg(args, i, a, 60); err != nil {
				return err
			}
			i++
		case reURL.MatchString(a) && !strings.Contains(a, "@"):
			urls++
		default:
			return fmt.Errorf("argument %q not allowed", a)
		}
	}
	if urls != 1 {
		return errors.New("needs exactly one http(s) URL")
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
