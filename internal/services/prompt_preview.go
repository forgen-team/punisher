package services

import (
	"context"
	"errors"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
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
	previewQuote     = "(예시) 지적된 문장"
	previewFinding   = "(예시) 이전 검수의 차단 지적이 여기에 들어갑니다"
	previewBlock     = "(재작성 전·후 블록이 여기에 들어갑니다)"
	previewQuery     = "(예시 학습 요청) 배포 후 API 응답이 갑자기 느려졌다"
	// previewMisconception fills "이 수강생이 인정한 오개념" when the user has none, so the block shows.
	previewMisconception = "(예시 오개념) 확인 결과에서 인정한 오개념이 여기에 들어갑니다"
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
	pc := t.LectureContext()
	if len(pc.Misconceptions) == 0 {
		pc.Misconceptions = []prompts.Misconception{{ConceptName: pc.ConceptName, Text: previewMisconception}}
	}
	vars := prompts.LectureVars(pc)
	out := PromptPreview{BlockID: blockID, Stage: block.Stage, ConceptID: pc.ConceptID, ConceptName: pc.ConceptName,
		UnknownVars: prompts.UnknownVars(set.Raw(blockID), block.Vars)}

	qa := prompts.QAContext{ConceptID: pc.ConceptID, ConceptName: pc.ConceptName, LectureTitle: "(예시 강의 제목)",
		SectionTitle: "(예시 섹션)", SectionText: "(강의 섹션 본문이 여기에 들어갑니다)", Linkable: pc.Linkable,
		LearnedBefore: pc.Learned, Misconceptions: pc.Misconceptions, ExampleLanguage: pc.ExampleLanguage, Reader: pc.Reader}
	switch block.Stage {
	case prompts.StageOutline:
		out.System, out.User = set.Text(prompts.BlockTeacherSystem, vars), prompts.Outline(pc, set)
		out.Version = set.Version(prompts.FullBlocks...)
	case prompts.StageLecture:
		out.System, out.User = set.Text(prompts.BlockTeacherSystem, vars), prompts.Lecture(pc, "", "", set)
		out.Version = set.Version(prompts.InstantBlocks...)
	case prompts.StageReview:
		switch blockID {
		case prompts.BlockReviewRecheck:
			out.System, out.User = set.Text(prompts.BlockReviewerSystem, vars), prompts.Recheck(pc,
				[]prompts.RecheckFinding{{ID: 1, Kind: "factual_error", Section: "walkthrough", Quote: previewQuote,
					Message: previewFinding, Before: []string{previewBlock}, After: []string{previewBlock}}},
				[]prompts.RecheckBlock{{Section: "walkthrough", Text: previewBlock}}, set)
		case prompts.BlockAdjudicator:
			out.System, out.User = set.Text(prompts.BlockAdjudicator, vars), prompts.Adjudication(pc, prompts.AdjudicationCase{
				Kind: "factual_error", Section: "walkthrough", Quote: previewQuote, Message: previewFinding,
				ReaderImpact: "(예시) 독자가 그대로 따르면 이르는 틀린 결과", Blocks: []string{previewBlock}})
		default:
			out.System, out.User = set.Text(prompts.BlockReviewerSystem, vars), prompts.Review(pc, previewDraft, set)
		}
		out.Version = set.Version(prompts.FullBlocks...)
	case prompts.StageGloss:
		out.System = set.Text(prompts.BlockGlossSystem, vars)
		out.User = prompts.Gloss(pc.ConceptName, pc.Area, pc.Summary, pc.Reader)
	case prompts.StageQA:
		out.System, out.User = set.Text(prompts.BlockQASystem, vars), prompts.Question(qa, previewQuestion)
	case prompts.StageExplain:
		out.System = set.Text(prompts.BlockExplainSystem, vars)
		out.User = prompts.ExplainSelection(prompts.ExplainContext{ConceptID: pc.ConceptID, ConceptName: pc.ConceptName,
			LectureTitle: qa.LectureTitle, SectionTitle: qa.SectionTitle, SectionText: qa.SectionText,
			ExampleLanguage: pc.ExampleLanguage, Reader: pc.Reader}, previewSelection)
	case prompts.StageResolve:
		cands := make([]prompts.ResolveCandidate, 0, len(pc.Linkable)+1)
		cands = append(cands, prompts.ResolveCandidate{ID: pc.ConceptID, Name: pc.ConceptName, Area: pc.Area, Summary: pc.Summary})
		for _, r := range pc.Linkable {
			cands = append(cands, prompts.ResolveCandidate{ID: r.ID, Name: r.Name})
		}
		out.System, out.User = set.Text(prompts.BlockResolveSystem, nil), prompts.Resolve(previewQuery, cands, curriculum.ProposalAreas)
	case prompts.StageCheck:
		out.System, out.User = set.Text(blockID, vars), previewCheck(blockID, pc)
	case prompts.StagePretest:
		out.System, out.User = set.Text(blockID, vars), previewPretest(pc)
	}
	out.System = prompts.WithOutputLanguage(out.System, t.Plan.Language) // 실제 호출과 같은 출력 언어 규칙
	if out.Version == "" {
		out.Version = set.Version(blockID)
	}
	return out, nil
}

// previewTarget builds the example concept's context with all of its
// depth-1 objectives, ignoring what the user already learned. It is the
// user's own (Owner set) so the personal blocks show as in instant lectures.
func (s *Service) previewTarget(ctx context.Context, userID string) (Target, error) {
	lang := s.Store.LocaleFor(ctx, userID)
	concept, err := s.Store.ConceptInfo(ctx, userID, PreviewConceptID, lang)
	if errors.Is(err, store.ErrNotFound) {
		catalog, cerr := s.Store.ConceptCatalog(ctx, userID, lang)
		if cerr != nil || len(catalog) == 0 {
			return Target{}, err
		}
		concept, err = s.Store.ConceptInfo(ctx, userID, catalog[0].ID, lang)
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
	plan.Language, plan.ReaderLevel = lang, s.Store.ReaderLevelFor(ctx, userID) // 미리보기도 호출하는 사용자의 수준 절
	t, err := s.targetForPlan(ctx, userID, concept, plan, "")
	t.Owner = userID
	return t, err
}

// previewCheck is the user prompt of a check step with example content.
func previewCheck(blockID string, pc prompts.LectureContext) string {
	if blockID == prompts.BlockCheckTutor {
		return prompts.CheckTutor(prompts.CheckTutorContext{ConceptName: pc.ConceptName, Objective: "(예시 목표)",
			Prompt: "(예시 문항) 입력이 두 배가 되면 비용은 어떻게 늘어나나요?", Must: []string{"(예시 기준)"},
			Answer: "(모범답이 여기에 들어갑니다)", LearnerAnswer: "(학습자 답이 여기에 들어갑니다)",
			Allowed: []string{"narrow", "next"}, Reader: pc.Reader})
	}
	objectives := make([]prompts.CheckObjective, 0, len(pc.Objectives))
	for _, o := range pc.Objectives {
		objectives = append(objectives, prompts.CheckObjective{ID: o.ID, Text: o.Text})
	}
	return prompts.CheckItems(prompts.CheckItemsContext{ConceptName: pc.ConceptName, LectureTitle: "(예시 강의 제목)",
		Objectives: objectives, Anchors: []string{"s1", "s2"}, Sections: []string{"[s1] (강의 본문이 여기에 들어갑니다)"}, Reader: pc.Reader})
}

// previewPretest is the user prompt of a lecture-start prediction set for the example concept.
func previewPretest(pc prompts.LectureContext) string {
	objectives := []prompts.DiagnosticObjective{}
	for _, o := range pc.Objectives {
		if len(objectives) < check.MaxPretestItems {
			objectives = append(objectives, prompts.DiagnosticObjective{ID: o.ID, ConceptName: pc.ConceptName, Text: o.Text})
		}
	}
	return prompts.Diagnostic(prompts.DiagnosticContext{Purpose: check.PurposePretest, Objectives: objectives,
		Count: check.PretestCount(len(objectives)), Reader: pc.Reader})
}
