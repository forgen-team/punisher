package lecture

import (
	"regexp"
	"strings"
)

// Text checks shared by markdown-bodied blocks (fences, mermaid placement,
// stray directives).

// mermaidProseRun: this many consecutive prose lines after a blank line means
// the diagram's end was missed and the following text fell into it.
const mermaidProseRun = 2

var (
	hangulRe        = regexp.MustCompile(`[\p{Hangul}]`)
	mdHeadingRe     = regexp.MustCompile(`^#{1,6}\s`)
	mermaidSyntaxRe = regexp.MustCompile(`-->|---|==>|-\.->|->>|-->>|--x|:::|^\s*(flowchart|graph|sequenceDiagram|stateDiagram|classDiagram|erDiagram|gantt|pie|journey|participant|actor|subgraph|end|classDef|class|style|linkStyle|click|note|loop|alt|else|opt|par|rect|autonumber|activate|deactivate|state|direction|section|title|%%)\b|^\s*[\w.-]+\s*[\[\(\{>]`)
)

// MermaidSwallowedProse returns why a mermaid source contains lecture prose
// ("" when it looks like a diagram): a markdown heading (## ), an @@ directive,
// or, after a blank line, mermaidProseRun+ consecutive Korean lines that are
// not diagram syntax.
func MermaidSwallowedProse(src string) string {
	afterBlank, run := false, 0
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case mdHeadingRe.MatchString(t):
			return "markdown heading " + clipLine(t)
		case strings.Contains(t, "@@"):
			return "@@ directive " + clipLine(t)
		case t == "":
			afterBlank, run = true, 0
			continue
		}
		if afterBlank && hangulRe.MatchString(t) && !mermaidSyntaxRe.MatchString(t) {
			run++
			if run >= mermaidProseRun {
				return "prose paragraph " + clipLine(t)
			}
			continue
		}
		afterBlank, run = false, 0
	}
	return ""
}

var (
	strayDirectiveRe = regexp.MustCompile(`@@(?:lecture|section|end|markdown|mermaid|code|log|misconception|term|sources|keypoint|trace|chart|diff|pause|exercise|answer|summary)\b`)
	inlineCodeRe     = regexp.MustCompile("`[^`]*`")
)

// StrayDirective returns the first block directive left inside markdown text
// ("" when none). Directives only work at the start of a line, so anywhere
// else the model's "@@term" is meaningless and the sentence loses a word
// (2026-11-10: "이 점은 @@term 에서 다루는 ALU 설계"). Code fences and inline
// code are exempt so a lecture can still show the literal text.
func StrayDirective(text string) string {
	open := ""
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimLeft(line, " ")
		if marker := fenceMarker(t); marker != "" && len(line)-len(t) <= 3 {
			if open == "" {
				open = marker
				continue
			}
			if marker[0] == open[0] && len(marker) >= len(open) && strings.TrimSpace(t[len(marker):]) == "" {
				open = ""
				continue
			}
		}
		if open != "" {
			continue
		}
		if d := strayDirectiveRe.FindString(inlineCodeRe.ReplaceAllString(line, "")); d != "" {
			return d
		}
	}
	return ""
}

// mermaidHeaderRe matches a line that is only a mermaid diagram header
// ("sequenceDiagram", "flowchart LR", "graph TD", ...). Prose that merely
// mentions the word does not match because the whole line must be the header.
var mermaidHeaderRe = regexp.MustCompile(`^\s*(sequenceDiagram|classDiagram|erDiagram|gantt|journey|mindmap|timeline|stateDiagram(-v2)?|pie(\s+(showData|title\b.*))?|(flowchart|graph)\s+(TD|TB|BT|RL|LR))\s*$`)

// MermaidOutsideBlock returns why markdown text carries a mermaid diagram
// that the renderer would show as plain code ("" when it does not): a
// ```mermaid fence, a plain fence whose first line is a diagram header, or a
// bare header line outside any fence (2026-11-09: a sequenceDiagram appeared
// as raw code in an instant lecture).
func MermaidOutsideBlock(text string) string {
	open, lang, first := "", "", true
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimLeft(line, " ")
		indented := len(line)-len(t) > 3
		if marker := fenceMarker(t); marker != "" && !indented {
			if open == "" {
				open, lang, first = marker, strings.ToLower(strings.TrimSpace(t[len(marker):])), true
				if lang == "mermaid" {
					return "```mermaid code fence"
				}
				continue
			}
			if marker[0] == open[0] && len(marker) >= len(open) && strings.TrimSpace(t[len(marker):]) == "" {
				open = ""
				continue
			}
		}
		if open != "" {
			if first && strings.TrimSpace(line) != "" {
				first = false
				if (lang == "" || lang == "text") && mermaidHeaderRe.MatchString(line) {
					return "code fence starting with " + clipLine(strings.TrimSpace(line))
				}
			}
			continue
		}
		if mermaidHeaderRe.MatchString(line) {
			return "bare diagram header " + clipLine(strings.TrimSpace(line))
		}
	}
	return ""
}

func clipLine(s string) string {
	r := []rune(s)
	if len(r) > 40 {
		return string(r[:40]) + "…"
	}
	return s
}

// UnclosedFence reports whether markdown text opens a ``` / ~~~ code fence it
// never closes (CommonMark: a fence closes with the same character, at least
// as long, on its own line).
func UnclosedFence(text string) bool {
	open := ""
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimLeft(line, " ")
		if len(line)-len(t) > 3 {
			continue
		}
		marker := fenceMarker(t)
		if marker == "" {
			continue
		}
		if open == "" {
			open = marker
		} else if marker[0] == open[0] && len(marker) >= len(open) && strings.TrimSpace(t[len(marker):]) == "" {
			open = ""
		}
	}
	return open != ""
}

func fenceMarker(line string) string {
	for _, ch := range []byte{'`', '~'} {
		n := 0
		for n < len(line) && line[n] == ch {
			n++
		}
		if n >= 3 {
			return line[:n]
		}
	}
	return ""
}
