package services

import (
	"context"
	"errors"

	"github.com/wooo-jin/punisher/internal/curriculum"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
)

// PreviewConceptID is the example concept used by prompt previews (falls back
// to the first concept when it is missing from the curriculum).
const PreviewConceptID = "ds.complexity"

// PromptPreview is the fully assembled prompt of one step, for reading only.
type PromptPreview struct {
	BlockID     string   `json:"blockId"`
	Stage       string   `json:"stage"`
	ConceptID   string   `json:"conceptId"`
	ConceptName string   `json:"conceptName"`
	System      string   `json:"system"`
	User        string   `json:"user"`
	Version     string   `json:"version"`
	UnknownVars []string `json:"unknownVars"`
}

// ErrUnknownPrompt: the block id is not in the registry.
var ErrUnknownPrompt = errors.New("unknown prompt block")

const (
	previewQuestion  = "(예시 질문) 이 개념을 실무에서 언제 의식해야 하나요?"
	previewSelection = "(예시 구절) 입력이 커질 때 비용이 어떤 모양으로 늘어나는지"
	previewDraft     = "(감수할 강의 초안이 여기에 들어갑니다)"
	previewQuery     = "(예시 학습 요청) 배포 후 API 응답이 갑자기 느려졌다"
)

// PreviewPrompt assembles the final system+user prompt of the step that uses
// blockID, with the user's blocks (and draft, when non-nil, in place of
// blockID) and an example concept. It makes no LLM call.
func (s *Service) PreviewPrompt(ctx context.Context, userID, blockID string, draft *string) (PromptPreview, error) {
	block, ok := prompts.LookupBlock(blockID)
	if !ok {
		return PromptPreview{}, ErrUnknownPrompt
	}
	set := s.UserPrompts(ctx, userID)
	if draft != nil {
		set = set.With(blockID, *draft)
	}
	t, err := s.previewTarget(ctx, userID)
	if err != nil {
		return PromptPreview{}, err
	}
	pc := t.Prompt
	vars := prompts.LectureVars(pc)
	out := PromptPreview{BlockID: blockID, Stage: block.Stage, ConceptID: pc.ConceptID, ConceptName: pc.ConceptName,
		UnknownVars: prompts.UnknownVars(set.Raw(blockID), block.Vars)}

	qa := prompts.QAContext{ConceptID: pc.ConceptID, ConceptName: pc.ConceptName, LectureTitle: "(예시 강의 제목)",
		SectionTitle: "(예시 섹션)", SectionText: "(강의 섹션 본문이 여기에 들어갑니다)", Linkable: pc.Linkable,
		LearnedBefore: pc.Learned, ExampleLanguage: pc.ExampleLanguage}
	switch block.Stage {
	case prompts.StageOutline:
		out.System, out.User = set.Text(prompts.BlockTeacherSystem, vars), prompts.Outline(pc, set)
		out.Version = set.Version(prompts.FullBlocks...)
	case prompts.StageLecture:
		out.System, out.User = set.Text(prompts.BlockTeacherSystem, vars), prompts.Lecture(pc, "", "", set)
		out.Version = set.Version(prompts.InstantBlocks...)
	case prompts.StageReview:
		out.System, out.User = set.Text(prompts.BlockReviewerSystem, vars), prompts.Review(pc, previewDraft, set)
		out.Version = set.Version(prompts.FullBlocks...)
	case prompts.StageGloss:
		out.System = set.Text(prompts.BlockGlossSystem, vars)
		out.User = prompts.Gloss(pc.ConceptName, pc.Area, pc.Summary)
	case prompts.StageQA:
		out.System, out.User = set.Text(prompts.BlockQASystem, vars), prompts.Question(qa, previewQuestion)
	case prompts.StageExplain:
		out.System = set.Text(prompts.BlockExplainSystem, vars)
		out.User = prompts.ExplainSelection(prompts.ExplainContext{ConceptID: pc.ConceptID, ConceptName: pc.ConceptName,
			LectureTitle: qa.LectureTitle, SectionTitle: qa.SectionTitle, SectionText: qa.SectionText,
			ExampleLanguage: pc.ExampleLanguage}, previewSelection)
	case prompts.StageResolve:
		cands := make([]prompts.ResolveCandidate, 0, len(pc.Linkable)+1)
		cands = append(cands, prompts.ResolveCandidate{ID: pc.ConceptID, Name: pc.ConceptName, Area: pc.Area, Summary: pc.Summary})
		for _, r := range pc.Linkable {
			cands = append(cands, prompts.ResolveCandidate{ID: r.ID, Name: r.Name})
		}
		out.System, out.User = set.Text(prompts.BlockResolveSystem, nil), prompts.Resolve(previewQuery, cands, curriculum.Areas)
	}
	if out.Version == "" {
		out.Version = set.Version(blockID)
	}
	return out, nil
}

// previewTarget builds the example concept's context with all of its
// depth-1 objectives, ignoring what the user already learned.
func (s *Service) previewTarget(ctx context.Context, userID string) (Target, error) {
	concept, err := s.Store.ConceptInfo(ctx, userID, PreviewConceptID)
	if errors.Is(err, store.ErrNotFound) {
		catalog, cerr := s.Store.ConceptCatalog(ctx, userID)
		if cerr != nil || len(catalog) == 0 {
			return Target{}, err
		}
		concept, err = s.Store.ConceptInfo(ctx, userID, catalog[0].ID)
	}
	if err != nil {
		return Target{}, err
	}
	refs := make([]learning.ObjectiveRef, 0, len(concept.Objectives))
	for _, o := range concept.Objectives {
		refs = append(refs, learning.ObjectiveRef{ID: o.ID, Depth: o.Depth, Ordinal: o.Ordinal})
	}
	plan, err := learning.PlanLecture(concept.ID, refs, nil, 0)
	if err != nil {
		return Target{}, err
	}
	return s.targetForPlan(ctx, userID, concept, plan, "")
}
