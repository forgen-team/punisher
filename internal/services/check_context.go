package services

import (
	"fmt"
	"strings"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/prompts"
)

// checkItemsContext renders the lecture body with its anchors for the item prompt.
func checkItemsContext(cl checkLecture) (prompts.CheckItemsContext, check.SetContext) {
	ic := prompts.CheckItemsContext{ConceptName: cl.info.Name, LectureTitle: cl.lec.Title}
	sc := check.SetContext{ObjectiveIDs: cl.objectiveIDs(), Anchors: map[string]bool{}}
	for _, o := range cl.objectives {
		ic.Objectives = append(ic.Objectives, prompts.CheckObjective{ID: o.ID, Text: o.Text})
	}
	for _, sec := range cl.content.Sections {
		var b strings.Builder
		fmt.Fprintf(&b, "### [%s] %s\n", sec.Anchor, sec.Title)
		addAnchor(&ic, sc, sec.Anchor)
		for _, blk := range sec.Blocks {
			addAnchor(&ic, sc, blk.Anchor)
			writeCheckBlock(&b, blk)
			if blk.Type == lecture.BlockCode || blk.Type == lecture.BlockLog {
				sc.LectureCode = append(sc.LectureCode, blk.Text)
			}
		}
		ic.Sections = append(ic.Sections, b.String())
	}
	return ic, sc
}

// writeCheckBlock renders a block as item-prompt text: prose and code as
// written, structured blocks (trace/chart/diff) as their JSON data.
func writeCheckBlock(b *strings.Builder, blk lecture.Block) {
	switch {
	case lecture.StructuredBlock(blk.Type) && len(blk.Data) > 0:
		fmt.Fprintf(b, "[%s] (%s) %s\n", blk.Anchor, blk.Type, blk.Data)
	case blk.Type == lecture.BlockKeypoint:
		fmt.Fprintf(b, "[%s] (%s) %s\n", blk.Anchor, blk.Kind, blk.Text)
	case blk.Type == lecture.BlockMermaid || blk.Type == lecture.BlockSources || blk.Type == lecture.BlockTerm:
	case blk.Text != "":
		fmt.Fprintf(b, "[%s] %s\n", blk.Anchor, blk.Text)
	}
}

func addAnchor(ic *prompts.CheckItemsContext, sc check.SetContext, anchor string) {
	if anchor != "" && !sc.Anchors[anchor] {
		sc.Anchors[anchor] = true
		ic.Anchors = append(ic.Anchors, anchor)
	}
}
