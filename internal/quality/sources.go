package quality

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/forgen-team/punisher/internal/lecture"
)

// OfficialHosts are documentation hosts accepted as sources without being in
// the curriculum (standards bodies, language/OS/database references). A host
// matches itself and its subdomains.
var OfficialHosts = []string{
	"rfc-editor.org", "ietf.org", "iana.org", "w3.org", "whatwg.org", "ecma-international.org", "tc39.es", "unicode.org",
	"man7.org", "kernel.org", "gnu.org", "freebsd.org", "openbsd.org", "developer.apple.com",
	"postgresql.org", "sqlite.org", "dev.mysql.com", "mariadb.com", "redis.io", "mongodb.com",
	"developer.mozilla.org", "nodejs.org", "typescriptlang.org", "go.dev", "golang.org", "python.org",
	"docs.oracle.com", "openjdk.org", "doc.rust-lang.org", "docs.rs", "cppreference.com", "kotlinlang.org",
	"learn.microsoft.com", "docs.aws.amazon.com", "cloud.google.com", "kubernetes.io", "docs.docker.com",
	"nginx.org", "curl.se", "openssl.org", "git-scm.com", "owasp.org", "nist.gov", "mitre.org", "letsencrypt.org",
	"opentelemetry.io", "prometheus.io", "kafka.apache.org", "spec.graphql.org", "json-schema.org", "json.org", "yaml.org",
}

// HostAllowed: host is an official host, or in extra, or a subdomain of one.
func HostAllowed(host string, extra map[string]bool) bool {
	host = strings.ToLower(strings.TrimPrefix(host, "www."))
	match := func(allowed string) bool {
		allowed = strings.ToLower(strings.TrimPrefix(allowed, "www."))
		return host == allowed || strings.HasSuffix(host, "."+allowed)
	}
	for _, h := range OfficialHosts {
		if match(h) {
			return true
		}
	}
	for h := range extra {
		if match(h) {
			return true
		}
	}
	return false
}

// G18: every source host is official or from the curriculum's doc_links.
func ruleSources(c lecture.Content, extra map[string]bool) RuleResult {
	var outside []string
	srcs := c.AllSources()
	for _, s := range srcs {
		u, err := url.Parse(s.URL)
		if err != nil || !HostAllowed(u.Hostname(), extra) {
			outside = append(outside, s.URL)
		}
	}
	if len(srcs) == 0 {
		return na("G18", "출처 없음")
	}
	return result("G18", len(outside) == 0, float64(len(outside)), "공식 문서 도메인",
		fmt.Sprintf("허용 목록 밖 출처 %d개: %s — 공식 문서로 바꾸세요", len(outside), strings.Join(outside, ", ")))
}

// LinkResult is one checked source link.
type LinkResult struct {
	URL    string `json:"url"`
	Status int    `json:"status"`
	// Broken: the server answered 404/410 (other failures, offline included, are not reported).
	Broken bool `json:"broken"`
}

// LinkChecker checks source links with HEAD (GET when HEAD is refused) —
// only https links on allowed hosts, so a generated URL never makes the app
// call an arbitrary or internal address. Results are cached for CacheFor.
type LinkChecker struct {
	Client   *http.Client
	CacheFor time.Duration
	mu       sync.Mutex
	cache    map[string]cachedLink
}

type cachedLink struct {
	res LinkResult
	at  time.Time
}

// NewLinkChecker returns a checker with a 3-second per-request limit and a day of cache.
func NewLinkChecker() *LinkChecker {
	return &LinkChecker{Client: &http.Client{Timeout: 3 * time.Second}, CacheFor: 24 * time.Hour}
}

// maxLinksChecked bounds one lecture's link checks.
const maxLinksChecked = 12

// Check checks the lecture's source links (allowed hosts only) in parallel
// and returns the answered ones. A nil checker checks nothing.
func (l *LinkChecker) Check(ctx context.Context, c lecture.Content, extra map[string]bool) []LinkResult {
	if l == nil {
		return nil
	}
	var urls []string
	for _, s := range c.AllSources() {
		u, err := url.Parse(s.URL)
		if err == nil && u.Scheme == "https" && HostAllowed(u.Hostname(), extra) && len(urls) < maxLinksChecked {
			urls = append(urls, s.URL)
		}
	}
	out := make([]LinkResult, len(urls))
	var wg sync.WaitGroup
	for i, u := range urls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = l.one(ctx, u)
		}()
	}
	wg.Wait()
	answered := out[:0]
	for _, r := range out {
		if r.Status != 0 {
			answered = append(answered, r)
		}
	}
	return answered
}

func (l *LinkChecker) one(ctx context.Context, raw string) LinkResult {
	l.mu.Lock()
	if c, ok := l.cache[raw]; ok && time.Since(c.at) < l.CacheFor {
		l.mu.Unlock()
		return c.res
	}
	l.mu.Unlock()
	status := l.status(ctx, http.MethodHead, raw)
	if status == http.StatusMethodNotAllowed || status == http.StatusForbidden || status == http.StatusNotImplemented {
		status = l.status(ctx, http.MethodGet, raw)
	}
	res := LinkResult{URL: raw, Status: status, Broken: status == http.StatusNotFound || status == http.StatusGone}
	if status != 0 {
		l.mu.Lock()
		if l.cache == nil {
			l.cache = map[string]cachedLink{}
		}
		l.cache[raw] = cachedLink{res: res, at: time.Now()}
		l.mu.Unlock()
	}
	return res
}

// status is the HTTP status of one request (0 = no answer: offline, timeout, refused).
func (l *LinkChecker) status(ctx context.Context, method, raw string) int {
	req, err := http.NewRequestWithContext(ctx, method, raw, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("User-Agent", "punisher-link-check")
	resp, err := l.Client.Do(req)
	if err != nil {
		return 0
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
	return resp.StatusCode
}

// BrokenLinkIssues are the broken links as advisory issues.
func BrokenLinkIssues(results []LinkResult) []lecture.Issue {
	var out []lecture.Issue
	for _, r := range results {
		if r.Broken {
			out = append(out, lecture.Issue{Path: "quality.L01", Message: fmt.Sprintf("출처 링크가 열리지 않음(HTTP %d): %s — 있는 공식 문서 주소로 바꾸세요", r.Status, r.URL), Advisory: true})
		}
	}
	return out
}
