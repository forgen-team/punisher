package lecture

import (
	"fmt"
	"strings"
)

// RenderMarkup writes stored content back as the markup the LLM writes
// (ParseMarkup's inverse), so a stored lecture can go through the same review
// prompt as a fresh draft (backfill review).
func RenderMarkup(c Content) string {
	var b strings.Builder
	b.WriteString("@@lecture\n")
	fmt.Fprintf(&b, "title: %s\n", c.Title)
	fmt.Fprintf(&b, "objectives: %s\n", strings.Join(c.ObjectiveIDs, ", "))
	for _, scope := range c.NewScope {
		fmt.Fprintf(&b, "new_scope: %s\n", scope)
	}
	fmt.Fprintf(&b, "prerequisites: %s\n", strings.Join(c.PrerequisitesReferenced, ", "))
	for _, s := range c.Sections {
		fmt.Fprintf(&b, "@@section %s minutes=%d title=%s\n", s.Kind, s.Minutes, s.Title)
		for _, blk := range s.Blocks {
			renderBlock(&b, blk)
		}
	}
	b.WriteString("@@end\n")
	return b.String()
}

// RenderBlock is one block as markup (a review quotes single blocks).
func RenderBlock(blk Block) string {
	var b strings.Builder
	renderBlock(&b, blk)
	return b.String()
}

func renderBlock(b *strings.Builder, blk Block) {
	switch blk.Type {
	case BlockCode, BlockLog, BlockDiff:
		fmt.Fprintf(b, "@@%s lang=%s\n", blk.Type, blk.Lang)
	case BlockKeypoint:
		fmt.Fprintf(b, "@@keypoint kind=%s\n", blk.Kind)
	case BlockLab:
		fmt.Fprintf(b, "@@lab os=%s\n", blk.Kind)
	case BlockTerm:
		proposed := ""
		if blk.Proposed {
			proposed = " proposed"
		}
		fmt.Fprintf(b, "@@term concept=%s%s term=%s\n", blk.ConceptID, proposed, blk.Term)
	case BlockSources:
		b.WriteString("@@sources\n")
		for _, src := range blk.Sources {
			fmt.Fprintf(b, "- [%s](%s)\n", src.Title, src.URL)
		}
		return
	default:
		fmt.Fprintf(b, "@@%s\n", blk.Type)
	}
	body := blk.Text
	if StructuredBlock(blk.Type) && len(blk.Data) > 0 {
		body = string(blk.Data)
	}
	if body != "" {
		b.WriteString(strings.TrimRight(body, "\n") + "\n")
	}
}

// HasMaterialBlocks reports whether c uses any v2.2 material block (keypoint,
// trace, chart, diff, pause, exercise, answer, summary). Lectures written before
// them are reviewed on their content, not on the presence of these blocks.
func HasMaterialBlocks(c Content) bool {
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			switch b.Type {
			case BlockKeypoint, BlockTrace, BlockChart, BlockDiff, BlockPause, BlockExercise, BlockAnswer, BlockSummary, BlockLab:
				return true
			}
		}
	}
	return false
}
