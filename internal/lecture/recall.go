package lecture

import "strings"

// recallKeyPrefixes start the answer key line of a @@recall body.
var recallKeyPrefixes = []string{"떠올릴 것:", "Recall:"}

// RecallParts splits a @@recall body into the question (the lines before the
// key line) and the key (the "떠올릴 것:" line and what follows). Either is ""
// when missing.
func RecallParts(text string) (question, key string) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		for _, p := range recallKeyPrefixes {
			if strings.HasPrefix(t, p) {
				rest := append([]string{strings.TrimSpace(t[len(p):])}, lines[i+1:]...)
				return strings.TrimSpace(strings.Join(lines[:i], "\n")), strings.TrimSpace(strings.Join(rest, "\n"))
			}
		}
	}
	return strings.TrimSpace(text), ""
}

// RecallBlocks are the lecture's @@recall blocks with their section kinds, in order.
func RecallBlocks(c Content) []SectionBlock {
	var out []SectionBlock
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			if b.Type == BlockRecall {
				out = append(out, SectionBlock{Section: s.Kind, Block: b})
			}
		}
	}
	return out
}

// SectionBlock is a block with the kind of the section holding it.
type SectionBlock struct {
	Section string
	Block   Block
}
