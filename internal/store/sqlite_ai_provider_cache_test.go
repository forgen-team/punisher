package store

import (
	"context"
	"testing"

	"github.com/forgen-team/punisher/internal/llm"
)

// A failed read keeps the last provider read instead of switching to claude.
func TestAIProviderForKeepsLastValueOnError(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	codex := llm.ProviderCodex
	if _, err := s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{AIProvider: &codex}); err != nil {
		t.Fatal(err)
	}
	broken, cancel := context.WithCancel(ctx)
	cancel()
	if got := s.AIProviderFor(broken, testUser); got != llm.ProviderClaude {
		t.Fatalf("nothing read yet: default expected, got %q", got)
	}
	if got := s.AIProviderFor(ctx, testUser); got != codex {
		t.Fatalf("AIProviderFor = %q", got)
	}
	if _, err := s.GetLearningSettings(broken, testUser); err == nil {
		t.Fatal("a cancelled context must make the read fail")
	}
	if got := s.AIProviderFor(broken, testUser); got != codex {
		t.Fatalf("read error must keep the last value %q, got %q", codex, got)
	}
}
