package httpapi

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	testHost  = "my-mac.tail1234.ts.net"
	testLogin = "brother@example.com"
)

func serveThrough(allowedHost string, r *http.Request) int {
	h := localOnly(allowedHost, testLogin)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec.Code
}

func TestLocalOnly(t *testing.T) {
	const login = testLogin
	cases := []struct {
		name    string
		allowed string
		method  string
		host    string
		headers map[string]string
		want    int
	}{
		{name: "loopback ip", host: "127.0.0.1:4710", want: 200},
		{name: "localhost", host: "localhost:4710", want: 200},
		{name: "loopback ip when phone is on", allowed: testHost, host: "127.0.0.1:4710", want: 200},
		{name: "loopback same origin", host: "127.0.0.1:4710", method: "POST", headers: map[string]string{"Origin": "http://127.0.0.1:4710"}, want: 200},
		{name: "loopback from other origin", host: "127.0.0.1:4710", method: "POST", headers: map[string]string{"Origin": "https://evil.example"}, want: 403},
		{name: "phone off refuses tailscale name", host: testHost, headers: map[string]string{"Tailscale-User-Login": login}, want: 403},

		{name: "serve request", allowed: testHost, host: testHost, headers: map[string]string{"Tailscale-User-Login": login}, want: 200},
		{name: "serve request with port", allowed: testHost, host: testHost + ":443", headers: map[string]string{"Tailscale-User-Login": login}, want: 200},
		{name: "serve same origin post", allowed: testHost, host: testHost, method: "POST", headers: map[string]string{"Tailscale-User-Login": login, "Origin": "https://" + testHost}, want: 200},

		{name: "another tailnet user", allowed: testHost, host: testHost, headers: map[string]string{"Tailscale-User-Login": "friend@example.com"}, want: 403},
		{name: "owner login differs by case", allowed: testHost, host: testHost, headers: map[string]string{"Tailscale-User-Login": "BROTHER@example.com"}, want: 403},
		{name: "upper-case host", allowed: testHost, host: "MY-MAC.tail1234.ts.net", headers: map[string]string{"Tailscale-User-Login": login}, want: 403},
		{name: "encoded-looking login", allowed: testHost, host: testHost, headers: map[string]string{"Tailscale-User-Login": "=?utf-8?q?brother@example.com?="}, want: 403},
		{name: "no identity header", allowed: testHost, host: testHost, want: 403},
		{name: "blank identity header", allowed: testHost, host: testHost, headers: map[string]string{"Tailscale-User-Login": "  "}, want: 403},
		{name: "forged host", allowed: testHost, host: "evil.example", headers: map[string]string{"Tailscale-User-Login": login}, want: 403},
		{name: "suffix of allowed host", allowed: testHost, host: "x." + testHost, headers: map[string]string{"Tailscale-User-Login": login}, want: 403},
		{name: "allowed host as prefix", allowed: testHost, host: testHost + ".evil.example", headers: map[string]string{"Tailscale-User-Login": login}, want: 403},
		{name: "other port on allowed host", allowed: testHost, host: testHost + ":8443", headers: map[string]string{"Tailscale-User-Login": login}, want: 403},
		{name: "other origin", allowed: testHost, host: testHost, method: "POST", headers: map[string]string{"Tailscale-User-Login": login, "Origin": "https://evil.example"}, want: 403},
		{name: "http origin of allowed host", allowed: testHost, host: testHost, method: "POST", headers: map[string]string{"Tailscale-User-Login": login, "Origin": "http://" + testHost}, want: 403},
		{name: "loopback origin on allowed host", allowed: testHost, host: testHost, method: "POST", headers: map[string]string{"Tailscale-User-Login": login, "Origin": "http://127.0.0.1:4710"}, want: 403},

		{name: "funnel with allowed host", allowed: testHost, host: testHost, headers: map[string]string{"Tailscale-User-Login": login, "Tailscale-Funnel-Request": "?1"}, want: 403},
		{name: "funnel with loopback host", allowed: testHost, host: "127.0.0.1", headers: map[string]string{"Tailscale-Funnel-Request": "?1"}, want: 403},
		{name: "proxied request claiming loopback host", allowed: testHost, host: "127.0.0.1", headers: map[string]string{"Tailscale-User-Login": login, "X-Forwarded-Host": "127.0.0.1"}, want: 403},
		{name: "tailscale header with loopback host", host: "127.0.0.1", headers: map[string]string{"Tailscale-User-Name": "x"}, want: 403},
		{name: "via with loopback host", host: "127.0.0.1", headers: map[string]string{"Via": "1.1 proxy"}, want: 403},
		{name: "forwarded for with loopback host", host: "localhost", headers: map[string]string{"X-Forwarded-For": "100.64.0.2"}, want: 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			method := tc.method
			if method == "" {
				method = http.MethodGet
			}
			r := httptest.NewRequest(method, "/v1/health", nil)
			r.Host = tc.host
			for k, v := range tc.headers {
				r.Header.Set(k, v)
			}
			if got := serveThrough(tc.allowed, r); got != tc.want {
				t.Fatalf("status = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestLocalOnlyNeedsAnOwner(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	r.Host = testHost
	r.Header.Set("Tailscale-User-Login", "")
	h := localOnly(testHost, "")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLocalOnlyMatchesEncodedLogin(t *testing.T) {
	h := localOnly(testHost, "동생@example.com")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	r := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	r.Host = testHost
	r.Header.Set("Tailscale-User-Login", mime.QEncoding.Encode("utf-8", "동생@example.com"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

// Another site's <img>/<iframe> sends no Origin but does send Sec-Fetch-Site:
// the API refuses it, a link may still open the screen, and nothing may frame the app.
func TestLocalOnlyCrossSiteWithoutOrigin(t *testing.T) {
	h := localOnly("", testLogin)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	serve := func(path, site string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.Host = "127.0.0.1:4710"
		if site != "" {
			r.Header.Set("Sec-Fetch-Site", site)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}
	for _, tc := range []struct {
		path, site string
		want       int
	}{
		{"/v1/today", "cross-site", http.StatusForbidden},
		{"/v1/today", "same-origin", http.StatusOK},
		{"/v1/today", "none", http.StatusOK},
		{"/v1/today", "", http.StatusOK},
		{"/lectures/x", "cross-site", http.StatusOK},
	} {
		rec := serve(tc.path, tc.site)
		if rec.Code != tc.want {
			t.Errorf("%s Sec-Fetch-Site=%q: %d, want %d", tc.path, tc.site, rec.Code, tc.want)
		}
		if rec.Header().Get("X-Frame-Options") != "DENY" || rec.Header().Get("Content-Security-Policy") != "frame-ancestors 'none'" {
			t.Errorf("%s: framing headers missing", tc.path)
		}
	}
}
