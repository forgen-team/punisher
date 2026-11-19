package witems

import (
	"context"
	"strings"

	"github.com/forgen-team/punisher/internal/claudecli"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/services"
)

// GatewayStatus reports the host claude-gateway in the shape of the local CLI
// status, so the settings screen and the Claude banner work unchanged.
type GatewayStatus struct {
	Client *llm.ClaudeGatewayClient
}

// Status maps GET /health of the gateway: "ok" is ready; "degraded" carries the
// CLI failure code in Reason (claude_not_logged_in, claude_unavailable: …).
func (g GatewayStatus) Status(ctx context.Context) claudecli.Status {
	h, err := g.Client.Health(ctx)
	if err != nil {
		code := llm.GatewayErrorCode(err)
		if code == "" {
			code = claudecli.CodeCLIMissing
		}
		return claudecli.Status{Code: code}
	}
	st := claudecli.Status{Installed: true, Version: h.ClaudeVersion, Active: h.Active, Queued: h.Queued}
	if h.Status == "ok" {
		st.Ready = true
		loggedIn := true
		st.LoggedIn = &loggedIn
		return st
	}
	code, _, _ := strings.Cut(h.Reason, ":")
	st.Code = strings.TrimSpace(code)
	if st.Code == claudecli.CodeNotLoggedIn {
		loggedIn := false
		st.LoggedIn = &loggedIn
	}
	return st
}

// OwnerSignals asks rss-server for the Witems owner's reading signals. Inside
// the app the only user is the local one, but rss-server knows the owner by
// their Witems user id.
type OwnerSignals struct {
	Source      services.SignalSource
	OwnerUserID string
}

func (o OwnerSignals) LearningSignals(ctx context.Context, _ string) (services.LearningSignals, error) {
	return o.Source.LearningSignals(ctx, o.OwnerUserID)
}
