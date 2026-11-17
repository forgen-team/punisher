package prompts

import (
	"strings"
	"testing"
)

func TestOutputLanguageRule(t *testing.T) {
	if OutputLanguageRule("ko") != "" || WithOutputLanguage("S", "") != "S" {
		t.Fatal("Korean needs no rule")
	}
	got := WithOutputLanguage("S", "en")
	if !strings.HasPrefix(got, "S") || !strings.Contains(got, "English") || !strings.Contains(got, "Expect:") {
		t.Fatalf("English rule: %q", got)
	}
}

// Default blocks must not pin the output language (the rule decides it).
func TestDefaultBlocksDoNotPinLanguage(t *testing.T) {
	for _, b := range Blocks {
		if strings.Contains(b.Default, "한국어로") || strings.Contains(b.Default, "한국어 존댓말") {
			t.Errorf("block %s pins Korean output", b.ID)
		}
	}
}
