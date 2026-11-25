package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/services"
)

// The Witems shell sends any 401 from /apps-api/… to /login. 401 must mean
// "no session" only: an AI provider that is signed out (the CLI/gateway says
// 401) is 503, any other LLM failure 502, a non-owner 403.
func TestUpstreamAuthFailureIsNot401(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"cli signed out", &llm.LLMError{StatusCode: http.StatusUnauthorized, Code: "claude_not_logged_in", Err: llm.ErrGenerationUnavailable}, http.StatusServiceUnavailable},
		{"llm 401 without sentinel", &llm.LLMError{StatusCode: http.StatusUnauthorized, Code: "x"}, http.StatusBadGateway},
		{"not owner", services.ErrNotOwner, http.StatusForbidden},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		writeServiceError(rec, httptest.NewRequest(http.MethodPost, "/v1/lectures/instant", nil), "test", c.err)
		if rec.Code != c.want {
			t.Errorf("%s: status %d, want %d", c.name, rec.Code, c.want)
		}
	}
}
