package claudecli

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/llm"
)

// TestRealClaudeSmoke makes one real `claude -p` call (PUNISHER_REAL_CLI_SMOKE=1).
func TestRealClaudeSmoke(t *testing.T) {
	if os.Getenv("PUNISHER_REAL_CLI_SMOKE") != "1" {
		t.Skip("set PUNISHER_REAL_CLI_SMOKE=1 to call the real claude CLI")
	}
	c := New(Config{TempBase: t.TempDir(), DefaultModel: "haiku"})
	t.Logf("status: %+v", c.Status(context.Background()))
	var deltas []string
	res, err := c.Stream(context.Background(), "", llm.MessagesRequest{System: "You are terse.",
		User: "Reply with exactly the word: pong", Timeout: 120 * time.Second}, func(d string) error {
		deltas = append(deltas, d)
		return nil
	})
	if err != nil {
		t.Fatalf("claude: %v", err)
	}
	t.Logf("result: text=%q model=%q usage=%+v latency=%dms deltas=%q", res.Text, res.Model, res.Usage, res.LatencyMs, deltas)
	if !strings.Contains(strings.ToLower(res.Text), "pong") || res.Usage.InputTokens == 0 {
		t.Fatalf("unexpected result %+v", res)
	}
}
