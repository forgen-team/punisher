package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/forgen-team/punisher/internal/claudecli"
)

type fixedStatus claudecli.Status

func (f fixedStatus) Status(context.Context) claudecli.Status { return claudecli.Status(f) }

// GET /v1/status lists every AI, names the selected one, keeps the claude
// field for older web builds and gives the fix-it text of the selected AI.
func TestStatusListsProvidersAndSelected(t *testing.T) {
	yes := true
	claude := fixedStatus{Ready: true, Installed: true, Version: "2.1.0 (Claude Code)", LoggedIn: &yes}
	h := &StatusHandler{
		Claude: claude,
		Providers: map[string]ClaudeStatusSource{
			"claude": claude,
			"codex":  fixedStatus{Installed: true, Version: "codex-cli 0.163.0", Code: "codex_not_logged_in"},
			"agy":    fixedStatus{Code: "agy_unavailable"},
		},
		Selected: func(context.Context) string { return "codex" },
	}
	rec := httptest.NewRecorder()
	h.Status(rec, httptest.NewRequest(http.MethodGet, "/v1/status", nil))
	var body struct {
		Claude    map[string]any   `json:"claude"`
		Provider  string           `json:"provider"`
		Providers []ProviderStatus `json:"providers"`
		Message   string           `json:"message"`
		Steps     []string         `json:"steps"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("%d %s (%v)", rec.Code, rec.Body, err)
	}
	if body.Provider != "codex" || body.Claude["version"] != "2.1.0 (Claude Code)" || body.Claude["ready"] != true {
		t.Fatalf("selected/claude = %q %v", body.Provider, body.Claude)
	}
	if len(body.Providers) != 3 || body.Providers[0].ID != "claude" || body.Providers[1].ID != "codex" || body.Providers[2].ID != "agy" {
		t.Fatalf("providers = %+v", body.Providers)
	}
	if p := body.Providers[1]; !p.Installed || p.Ready || p.Code != "codex_not_logged_in" || p.Version != "codex-cli 0.163.0" {
		t.Fatalf("codex line = %+v", p)
	}
	if p := body.Providers[2]; p.Installed || p.Code != "agy_unavailable" || p.LoggedIn != nil {
		t.Fatalf("agy line = %+v", p)
	}
	if !strings.Contains(body.Message, "Codex") || len(body.Steps) == 0 || !strings.Contains(body.Steps[0], "codex login") {
		t.Fatalf("fix-it for the selected AI: %q %v", body.Message, body.Steps)
	}
	if !strings.Contains(rec.Body.String(), `"loggedIn":null`) {
		t.Fatalf("unknown login must be null: %s", rec.Body)
	}
}

// Without providers (Witems app: claude-gateway only) the list is just claude.
func TestStatusClaudeOnly(t *testing.T) {
	h := &StatusHandler{Claude: fixedStatus{Code: claudecli.CodeCLIMissing}}
	rec := httptest.NewRecorder()
	h.Status(rec, httptest.NewRequest(http.MethodGet, "/v1/status", nil))
	var body statusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Provider != "claude" || len(body.Providers) != 1 || body.Providers[0].Code != claudecli.CodeCLIMissing || !strings.Contains(body.Message, "claude 명령") {
		t.Fatalf("claude only = %+v", body)
	}
}
