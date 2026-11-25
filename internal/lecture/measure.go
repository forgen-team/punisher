package lecture

import (
	"encoding/json"
	"strings"
	"unicode"
)

// Measure is what a lecture's reading time is computed from (quality design
// v2.2 §5): characters without whitespace, split into prose and code/log, and
// the blocks that take time beyond reading.
type Measure struct {
	// ProseChars: markdown-bodied text, pause questions, term glosses and the
	// captions/notes of structured blocks (code fences inside markdown count as code).
	ProseChars int
	// CodeChars: code and log blocks, diff before/after and fenced code inside markdown.
	CodeChars int
	Exercises int
	Pauses    int
	Recalls   int
	TraceRows int
	// Hangul / Latin letters of the prose: the lecture language (Language).
	HangulLetters int
	LatinLetters  int
}

// Language is "ko" when the prose is mostly Hangul, "en" when it is mostly
// Latin letters ("ko" when there is no prose: the old default).
func (m Measure) Language() string {
	if m.LatinLetters > 2*m.HangulLetters && m.LatinLetters > 0 {
		return "en"
	}
	return "ko"
}

// addProse counts prose text: visible characters and letter scripts.
func (m *Measure) addProse(text string) {
	m.ProseChars += visibleChars(text)
	for _, r := range text {
		switch {
		case unicode.Is(unicode.Hangul, r):
			m.HangulLetters++
		case r < unicode.MaxASCII && unicode.IsLetter(r):
			m.LatinLetters++
		}
	}
}

// MeasureContent counts c (mermaid sources and source lists are not read as text).
func MeasureContent(c Content) Measure {
	var m Measure
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			m.add(b)
		}
	}
	return m
}

func (m *Measure) add(b Block) {
	switch b.Type {
	case BlockCode, BlockLog:
		m.CodeChars += visibleChars(b.Text)
	case BlockTerm, BlockPause:
		m.addProse(b.Text)
		if b.Type == BlockPause {
			m.Pauses++
		}
	case BlockTrace:
		var d TraceData
		if json.Unmarshal(b.Data, &d) == nil {
			m.addProse(d.Caption)
			m.addProse(d.Note)
			m.TraceRows += len(d.Rows)
		}
	case BlockChart:
		var d ChartData
		if json.Unmarshal(b.Data, &d) == nil {
			m.addProse(d.Title)
			m.addProse(d.Note)
		}
	case BlockDiff:
		var d DiffData
		if json.Unmarshal(b.Data, &d) == nil {
			m.CodeChars += visibleChars(d.Before) + visibleChars(d.After)
			m.addProse(d.Note)
		}
	default:
		if markdownBodied[b.Type] {
			prose, code := splitFenced(b.Text)
			m.addProse(prose)
			m.CodeChars += visibleChars(code)
			switch b.Type {
			case BlockExercise:
				m.Exercises++
			case BlockRecall:
				m.Recalls++
			}
		}
	}
}

// ProseOnly is the markdown text of a block outside ``` / ~~~ fences.
func ProseOnly(text string) string {
	prose, _ := splitFenced(text)
	return prose
}

// splitFenced separates markdown text outside and inside code fences.
func splitFenced(text string) (prose, code string) {
	var p, c strings.Builder
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
			c.WriteString(line + "\n")
		} else {
			p.WriteString(line + "\n")
		}
	}
	return p.String(), c.String()
}

// visibleChars counts runes that are not whitespace (공백 제외 글자 수).
func visibleChars(s string) int {
	n := 0
	for _, r := range s {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}
