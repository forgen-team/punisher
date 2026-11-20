package llm

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
)

// AI providers the owner can choose (user_learning_settings.ai_provider).
const (
	ProviderClaude = "claude" // Claude Code CLI (default)
	ProviderCodex  = "codex"  // OpenAI Codex CLI
	ProviderAgy    = "agy"    // Antigravity CLI (Gemini)
	// DefaultProvider is used when nothing (or an unknown value) is saved.
	DefaultProvider = ProviderClaude
)

// Providers is the display order of the providers.
var Providers = []string{ProviderClaude, ProviderCodex, ProviderAgy}

// ValidProvider reports a known provider id.
func ValidProvider(p string) bool {
	return p == ProviderClaude || p == ProviderCodex || p == ProviderAgy
}

// NormalizeProvider maps "" and unknown values to the default.
func NormalizeProvider(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	if ValidProvider(p) {
		return p
	}
	return DefaultProvider
}

// Client is one provider's generation client (claudecli/codexcli/agycli).
type Client interface {
	Create(ctx context.Context, apiKey string, req MessagesRequest) (MessagesResult, error)
	Stream(ctx context.Context, apiKey string, req MessagesRequest, onDelta func(string) error) (MessagesResult, error)
}

// Tiers are a provider's models for the lecture and fast tiers ("" = the CLI's default model).
type Tiers struct {
	Lecture string
	Fast    string
}

// fastTimeoutFactor stretches fast-tier timeouts for non-claude providers:
// callers size them for claude haiku, and the other CLIs' default models
// answer a short prompt more slowly (codex ~5s for a one-word reply).
const fastTimeoutFactor = 2

// Router sends every call to the provider selected right now (read per
// call, so a settings change applies without a restart). Callers pass the
// claude tier models (services.Config); for other providers the request's
// tier picks that provider's model: an explicit req.Tier, else FastModel →
// Tiers.Fast and anything else → Tiers.Lecture (check grading follows
// lecture; the regrade model follows fast when it is FastModel, else lecture).
type Router struct {
	Clients map[string]Client
	// Tiers per non-claude provider (claude models pass through).
	Tiers map[string]Tiers
	// FastModel is the claude fast-tier model callers use (e.g. "haiku").
	FastModel string
	// RegradeModel is the claude model of check regrades (CheckRegradeModel);
	// only used to warn when a provider would regrade with the grading model.
	RegradeModel string
	// Selected returns the provider to use; nil or "" = DefaultProvider.
	Selected func(ctx context.Context) string

	regradeWarned sync.Map // provider → struct{}
}

// Provider is the provider a call made now would use.
func (r *Router) Provider(ctx context.Context) string {
	if r.Selected == nil {
		return DefaultProvider
	}
	return NormalizeProvider(r.Selected(ctx))
}

// TierOf is req's tier: req.Tier when set, else fast for FastModel and lecture otherwise.
func (r *Router) TierOf(req MessagesRequest) Tier {
	if req.Tier != "" {
		return req.Tier
	}
	if r.FastModel != "" && strings.EqualFold(strings.TrimSpace(req.Model), strings.TrimSpace(r.FastModel)) {
		return TierFast
	}
	return TierLecture
}

func (r *Router) route(ctx context.Context, req MessagesRequest) (string, Client, MessagesRequest, error) {
	p := r.Provider(ctx)
	c, ok := r.Clients[p]
	if !ok || c == nil {
		return p, nil, req, &LLMError{StatusCode: http.StatusServiceUnavailable, Code: p + "_unavailable", Provider: p,
			Message: p + ": " + p + "_unavailable: provider is not available in this build", Err: ErrGenerationUnavailable}
	}
	tier := r.TierOf(req)
	if p != ProviderClaude {
		t := r.Tiers[p]
		r.warnSameRegradeModel(p, t, req)
		if tier == TierFast {
			req.Model = t.Fast
			if req.Timeout > 0 {
				req.Timeout *= fastTimeoutFactor
			}
		} else {
			req.Model = t.Lecture
		}
	}
	req.Tier = tier
	return p, c, req, nil
}

// warnSameRegradeModel logs once per provider when a regrade runs on a
// provider without tier models: grading and regrade then both use the CLI's
// default model, so the "second opinion from a different model" is lost.
func (r *Router) warnSameRegradeModel(p string, t Tiers, req MessagesRequest) {
	if r.RegradeModel == "" || t.Lecture != "" || t.Fast != "" || !strings.EqualFold(strings.TrimSpace(req.Model), strings.TrimSpace(r.RegradeModel)) {
		return
	}
	if _, loaded := r.regradeWarned.LoadOrStore(p, struct{}{}); loaded {
		return
	}
	log.Printf("[llm] %s: no lecture/fast models configured, so check regrades use the same CLI default model as grading (set PUNISHER_%s_FAST_MODEL or PUNISHER_%s_LECTURE_MODEL for a different one)",
		p, strings.ToUpper(p), strings.ToUpper(p))
}

// Create implements services.LLM.
func (r *Router) Create(ctx context.Context, apiKey string, req MessagesRequest) (MessagesResult, error) {
	p, c, req, err := r.route(ctx, req)
	if err != nil {
		return MessagesResult{Provider: p}, err
	}
	res, err := c.Create(ctx, apiKey, req)
	res.Provider = p
	return res, err
}

// Stream implements services.LLM.
func (r *Router) Stream(ctx context.Context, apiKey string, req MessagesRequest, onDelta func(string) error) (MessagesResult, error) {
	p, c, req, err := r.route(ctx, req)
	if err != nil {
		return MessagesResult{Provider: p}, err
	}
	res, err := c.Stream(ctx, apiKey, req, onDelta)
	res.Provider = p
	return res, err
}
