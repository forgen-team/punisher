package codexcli

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/llm"
)

// TestRealCodexSmoke makes one real fast-tier `codex exec` call
// (PUNISHER_REAL_CLI_SMOKE=1): read-only sandbox, shell tool disabled, low
// reasoning effort on the default model.
func TestRealCodexSmoke(t *testing.T) {
	if os.Getenv("PUNISHER_REAL_CLI_SMOKE") != "1" {
		t.Skip("set PUNISHER_REAL_CLI_SMOKE=1 to call the real codex CLI")
	}
	c := New(Config{TempBase: t.TempDir()})
	t.Logf("status: %+v", c.Status(context.Background()))
	var deltas []string
	res, err := c.Stream(context.Background(), "", llm.MessagesRequest{System: "You are terse.",
		User: "Reply with exactly the word: pong", Tier: llm.TierFast, Timeout: 120 * time.Second}, func(d string) error {
		deltas = append(deltas, d)
		return nil
	})
	if err != nil {
		t.Fatalf("codex: %v", err)
	}
	t.Logf("result: text=%q model=%q usage=%+v latency=%dms deltas=%q", res.Text, res.Model, res.Usage, res.LatencyMs, deltas)
	if !strings.Contains(strings.ToLower(res.Text), "pong") || res.Usage.InputTokens == 0 {
		t.Fatalf("unexpected result %+v", res)
	}
}
