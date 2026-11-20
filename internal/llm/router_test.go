package llm

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

type recordingClient struct {
	name   string
	models []string
}

func (c *recordingClient) Create(_ context.Context, _ string, req MessagesRequest) (MessagesResult, error) {
	c.models = append(c.models, req.Model)
	return MessagesResult{Text: c.name, Model: req.Model}, nil
}

func (c *recordingClient) Stream(_ context.Context, _ string, req MessagesRequest, onDelta func(string) error) (MessagesResult, error) {
	c.models = append(c.models, req.Model)
	return MessagesResult{Text: c.name}, onDelta(c.name)
}

func TestRouterDispatchesPerCallAndMapsTiers(t *testing.T) {
	claude, codex := &recordingClient{name: "claude"}, &recordingClient{name: "codex"}
	selected := ""
	r := &Router{
		Clients:   map[string]Client{ProviderClaude: claude, ProviderCodex: codex},
		Tiers:     map[string]Tiers{ProviderCodex: {Lecture: "", Fast: "gpt-mini"}},
		FastModel: "haiku",
		Selected:  func(context.Context) string { return selected },
	}
	ctx := context.Background()
	res, err := r.Create(ctx, "", MessagesRequest{Model: "sonnet", User: "x"})
	if err != nil || res.Text != "claude" || res.Provider != ProviderClaude || claude.models[0] != "sonnet" {
		t.Fatalf("default = claude with its own model: %+v %v %v", res, err, claude.models)
	}

	selected = "codex" // a settings change applies to the next call
	res, err = r.Create(ctx, "", MessagesRequest{Model: "sonnet", User: "x"})
	if err != nil || res.Text != "codex" || res.Provider != ProviderCodex {
		t.Fatalf("codex: %+v %v", res, err)
	}
	if _, err := r.Stream(ctx, "", MessagesRequest{Model: "haiku", User: "x"}, func(string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if len(codex.models) != 2 || codex.models[0] != "" || codex.models[1] != "gpt-mini" {
		t.Fatalf("lecture tier → CLI default, fast tier → fast model: %q", codex.models)
	}

	selected = "agy" // valid but not wired → unavailable with the provider's code
	_, err = r.Create(ctx, "", MessagesRequest{User: "x"})
	var le *LLMError
	if !errors.Is(err, ErrGenerationUnavailable) || !errors.As(err, &le) || le.Code != "agy_unavailable" || le.Provider != ProviderAgy {
		t.Fatalf("unwired provider = %v", err)
	}

	selected = "gemini" // unknown → default
	if res, _ := r.Create(ctx, "", MessagesRequest{User: "x"}); res.Provider != ProviderClaude {
		t.Fatalf("unknown provider must fall back to claude: %+v", res)
	}
}

type tierClient struct{ reqs []MessagesRequest }

func (c *tierClient) Create(_ context.Context, _ string, req MessagesRequest) (MessagesResult, error) {
	c.reqs = append(c.reqs, req)
	return MessagesResult{}, nil
}

func (c *tierClient) Stream(ctx context.Context, k string, req MessagesRequest, _ func(string) error) (MessagesResult, error) {
	return c.Create(ctx, k, req)
}

func TestRouterTiersTimeoutsAndRegrade(t *testing.T) {
	claude, codex := &tierClient{}, &tierClient{}
	selected := ProviderCodex
	r := &Router{
		Clients:      map[string]Client{ProviderClaude: claude, ProviderCodex: codex},
		Tiers:        map[string]Tiers{ProviderCodex: {Lecture: "gpt-big", Fast: "gpt-mini"}},
		FastModel:    "haiku",
		RegradeModel: "opus",
		Selected:     func(context.Context) string { return selected },
	}
	ctx := context.Background()
	calls := []MessagesRequest{
		{Model: "haiku", Timeout: 30 * time.Second}, // fast by model, timeout ×2
		{Model: "HAIKU "}, // case/space-insensitive, no timeout stays 0
		{Model: "sonnet", Tier: TierFast, Timeout: time.Second},   // explicit tier wins
		{Model: "haiku", Tier: TierLecture, Timeout: time.Second}, // explicit lecture
		{Model: "opus", Timeout: time.Second},                     // regrade model ≠ FastModel → lecture
	}
	for _, c := range calls {
		if _, err := r.Create(ctx, "", c); err != nil {
			t.Fatal(err)
		}
	}
	want := []struct {
		model   string
		tier    Tier
		timeout time.Duration
	}{
		{"gpt-mini", TierFast, 60 * time.Second}, {"gpt-mini", TierFast, 0}, {"gpt-mini", TierFast, 2 * time.Second},
		{"gpt-big", TierLecture, time.Second}, {"gpt-big", TierLecture, time.Second},
	}
	for i, w := range want {
		got := codex.reqs[i]
		if got.Model != w.model || got.Tier != w.tier || got.Timeout != w.timeout {
			t.Errorf("call %d: got model=%q tier=%q timeout=%s, want %+v", i, got.Model, got.Tier, got.Timeout, w)
		}
	}

	r.RegradeModel = "haiku" // regrade on the fast model → fast tier
	if _, err := r.Create(ctx, "", MessagesRequest{Model: "haiku"}); err != nil || codex.reqs[5].Tier != TierFast {
		t.Fatalf("regrade = FastModel must be fast: %+v", codex.reqs[5])
	}

	selected = ProviderClaude // claude keeps its model and timeout; the tier is still set
	if _, err := r.Create(ctx, "", MessagesRequest{Model: "haiku", Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	if got := claude.reqs[0]; got.Model != "haiku" || got.Timeout != time.Second || got.Tier != TierFast {
		t.Fatalf("claude request = %+v", got)
	}
}

func TestRouterWarnsOnceWhenRegradeUsesSameDefaultModel(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	r := &Router{
		Clients:      map[string]Client{ProviderCodex: &tierClient{}},
		Tiers:        map[string]Tiers{ProviderCodex: {}},
		FastModel:    "haiku",
		RegradeModel: "haiku",
		Selected:     func(context.Context) string { return ProviderCodex },
	}
	for range 3 {
		_, _ = r.Create(context.Background(), "", MessagesRequest{Model: "haiku"})
	}
	_, _ = r.Create(context.Background(), "", MessagesRequest{Model: "sonnet"})
	if n := strings.Count(buf.String(), "regrades use the same CLI default model"); n != 1 {
		t.Fatalf("warning logged %d times:\n%s", n, buf.String())
	}
}
