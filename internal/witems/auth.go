// Package witems runs Punisher as an app inside the Witems shell
// (PUNISHER_MODE=witems; contract: witems docs/APP-CONTRACT.md).
//
// The shell signs users in. Every /v1 request carries the WITEMS_SESSION
// cookie (nginx forwards only that cookie); Guard checks it with Witems
// project-setup and lets only the owner through. Inside the app everyone is
// the single local user, like the standalone app.
package witems

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	sessionCookie = "WITEMS_SESSION"
	sessionTTL    = 5 * time.Minute
	authTimeout   = 3 * time.Second
	maxCached     = 1024
)

// errUnauthenticated: no session, or project-setup said the session is not valid.
var errUnauthenticated = errors.New("not authenticated")

// Guard admits only the Witems owner. A missing or rejected session is 401,
// another account is 403, a foreign Origin is 403, and project-setup being
// unreachable is 503 (never treated as "allowed").
//
// Witems services (rss-server's "study this concept", mcp-server tools) call
// the API inside the Docker network with X-Internal-Token + X-Internal-User-Id,
// like the go-shared auth. nginx clears those headers on browser requests, so
// this path is reachable only from inside; it too admits only the owner.
type Guard struct {
	AuthServiceURL string
	OwnerUserID    string
	// InternalToken is WITEMS_INTERNAL_TOKEN; empty disables service calls.
	InternalToken string
	// Origin is the Witems origin; a request with a different Origin header is refused.
	Origin string
	Client *http.Client
	// Now is for tests.
	Now func() time.Time

	mu    sync.Mutex
	cache map[string]cachedSession
}

type cachedSession struct {
	userID  string
	expires time.Time
}

// Middleware wraps the /v1 API.
func (g *Guard) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serviceUser := r.Header.Get("X-Internal-User-Id"); serviceUser != "" {
			g.serveService(w, r, serviceUser, next)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != g.Origin {
			writeError(w, http.StatusForbidden, "FORBIDDEN_ORIGIN", "forbidden origin")
			return
		}
		cookie, err := r.Cookie(sessionCookie)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "login required")
			return
		}
		userID, err := g.userFor(r.Context(), cookie.Value)
		switch {
		case errors.Is(err, errUnauthenticated):
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "login required")
			return
		case err != nil:
			writeError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "could not check the login")
			return
		case userID != g.OwnerUserID:
			writeError(w, http.StatusForbidden, "OWNER_ONLY", "only the owner account can use this app")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// serveService handles a call from another Witems service.
func (g *Guard) serveService(w http.ResponseWriter, r *http.Request, userID string, next http.Handler) {
	token := r.Header.Get("X-Internal-Token")
	if g.InternalToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(g.InternalToken)) != 1 {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid internal token")
		return
	}
	if userID != g.OwnerUserID {
		writeError(w, http.StatusForbidden, "OWNER_ONLY", "only the owner account can use this app")
		return
	}
	next.ServeHTTP(w, r)
}

func (g *Guard) now() time.Time {
	if g.Now != nil {
		return g.Now()
	}
	return time.Now()
}

// userFor returns the Witems user id of a session (cached for sessionTTL).
// Only successful checks are cached, so a revoked or newly valid session is
// seen within sessionTTL at most.
func (g *Guard) userFor(ctx context.Context, token string) (string, error) {
	now := g.now()
	g.mu.Lock()
	if c, ok := g.cache[token]; ok && now.Before(c.expires) {
		g.mu.Unlock()
		return c.userID, nil
	}
	g.mu.Unlock()

	userID, err := g.lookup(ctx, token)
	if err != nil {
		return "", err
	}
	g.mu.Lock()
	if g.cache == nil || len(g.cache) >= maxCached {
		g.cache = make(map[string]cachedSession)
	}
	g.cache[token] = cachedSession{userID: userID, expires: now.Add(sessionTTL)}
	g.mu.Unlock()
	return userID, nil
}

func (g *Guard) lookup(ctx context.Context, token string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, authTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.AuthServiceURL+"/v1/auth/me", nil)
	if err != nil {
		return "", err
	}
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", errUnauthenticated
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth service: status %d", resp.StatusCode)
	}
	var body struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&body); err != nil {
		return "", fmt.Errorf("auth service: %w", err)
	}
	if body.User.ID == "" {
		return "", errUnauthenticated
	}
	return body.User.ID, nil
}

// writeError uses the API's error envelope ({"error": {"code", "message"}}).
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}
