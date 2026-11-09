package lecture

import "regexp"

// inlineTermRe matches an inline concept link in markdown: [표시 용어](term:concept-id).
var inlineTermRe = regexp.MustCompile(`\[([^\]\n]+)\]\(term:([^)\s]*)\)`)

// InlineTermLinks returns the concept ids linked inline in markdown/misconception blocks.
func InlineTermLinks(c Content) []string {
	var ids []string
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			if b.Type != BlockMarkdown && b.Type != BlockMisconception {
				continue
			}
			for _, m := range inlineTermRe.FindAllStringSubmatch(b.Text, -1) {
				ids = append(ids, m[2])
			}
		}
	}
	return ids
}

// NormalizeInlineTerms keeps inline term links whose concept exists and turns
// the rest into plain text (the label), so the viewer never offers a gloss for
// a concept that is not in the graph. It returns how many links were dropped.
func NormalizeInlineTerms(c *Content, known map[string]bool) int {
	dropped := 0
	for si := range c.Sections {
		for bi := range c.Sections[si].Blocks {
			b := &c.Sections[si].Blocks[bi]
			if b.Type != BlockMarkdown && b.Type != BlockMisconception {
				continue
			}
			b.Text = inlineTermRe.ReplaceAllStringFunc(b.Text, func(link string) string {
				m := inlineTermRe.FindStringSubmatch(link)
				if known[m[2]] {
					return link
				}
				dropped++
				return m[1]
			})
		}
	}
	return dropped
}
