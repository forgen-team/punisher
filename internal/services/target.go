package services

import (
	"context"
	"log"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
)

func logf(format string, args ...any) { log.Printf("[services] "+format, args...) }

// Target is a fully prepared generation target.
type Target struct {
	Plan       learning.LecturePlan
	Concept    models.ConceptInfo
	Prompt     prompts.LectureContext
	Validation lecture.ValidationContext
	// Prompts: instruction blocks to use. Zero value = code defaults (library lectures).
	Prompts prompts.Set
	// PromptBlocks: blocks this generation uses (for the recorded prompt version).
	PromptBlocks []string
	// Owner: requested_by of user-owned (instant/regenerated) lectures; "" = library.
	Owner string
	// personal: the learner's weak concepts and misconceptions (used only when Owner is set — LectureContext).
	personal personalContext
}

// promptVersion is the version recorded on lectures saved from t.
func (t Target) promptVersion() string {
	return t.Prompts.Version(t.PromptBlocks...)
}

// BuildTarget plans the next lecture for (user, concept, depth) and gathers
// prompt/validation context. Returns learning.ErrNothingToLearn when every
// objective at that depth is completed/known (dedup invariant).
func (s *Service) BuildTarget(ctx context.Context, userID, conceptID string, depth int, query string) (Target, error) {
	concept, err := s.Store.ConceptInfo(ctx, userID, conceptID)
	if err != nil {
		return Target{}, err
	}
	done, err := s.Store.DoneSet(ctx, userID, conceptID)
	if err != nil {
		return Target{}, err
	}
	refs := make([]learning.ObjectiveRef, 0, len(concept.Objectives))
	for _, o := range concept.Objectives {
		refs = append(refs, learning.ObjectiveRef{ID: o.ID, Depth: o.Depth, Ordinal: o.Ordinal})
	}
	plan, err := learning.PlanLecture(conceptID, refs, done, depth)
	if err != nil {
		return Target{}, err
	}
	return s.targetForPlan(ctx, userID, concept, plan, query)
}

// TargetForLecture rebuilds the generation target of an existing lecture (same
// concept, objectives and depth) regardless of what the user already learned.
func (s *Service) TargetForLecture(ctx context.Context, userID string, lec models.Lecture) (Target, error) {
	concept, err := s.Store.ConceptInfo(ctx, userID, lec.ConceptID)
	if err != nil {
		return Target{}, err
	}
	plan := learning.LecturePlan{
		ConceptID: lec.ConceptID, ObjectiveIDs: lec.ObjectiveIDs,
		ObjectiveKey: learning.ObjectiveKey(lec.ObjectiveIDs), Depth: lec.Depth,
	}
	return s.targetForPlan(ctx, userID, concept, plan, "")
}

func (s *Service) targetForPlan(ctx context.Context, userID string, concept models.ConceptInfo, plan learning.LecturePlan, query string) (Target, error) {
	conceptID := concept.ID
	related, err := s.Store.RelatedConcepts(ctx, userID, conceptID)
	if err != nil {
		return Target{}, err
	}
	known, err := s.Store.KnownConceptIDs(ctx, userID)
	if err != nil {
		return Target{}, err
	}
	feedback, err := s.Store.LatestFeedback(ctx, userID)
	if err != nil {
		return Target{}, err
	}

	pc := prompts.LectureContext{
		ConceptID: concept.ID, ConceptName: concept.Name, Area: concept.Area, Depth: plan.Depth,
		Summary: concept.Summary, PracticalCases: concept.PracticalCases, StyleHint: prompts.StyleHintFor(feedback),
		Query: query, ExampleLanguage: s.exampleLanguageName(ctx, userID),
	}
	for _, d := range concept.DocLinks {
		pc.DocLinks = append(pc.DocLinks, prompts.DocLink{Title: d.Title, URL: d.URL})
	}
	planned := map[string]bool{}
	for _, id := range plan.ObjectiveIDs {
		planned[id] = true
	}
	allowed := map[string]bool{}
	for _, o := range concept.Objectives {
		allowed[o.ID] = true
		if planned[o.ID] {
			pc.Objectives = append(pc.Objectives, prompts.ObjectiveRef{ID: o.ID, Text: o.Text})
		}
	}
	for _, r := range related {
		ref := prompts.ConceptRef{ID: r.ID, Name: r.Name}
		pc.Linkable = append(pc.Linkable, ref)
		if learning.IsDone(r.State) {
			pc.Learned = append(pc.Learned, ref)
		}
	}
	prereqs, err := s.Store.PrerequisitesOf(ctx, userID, conceptID)
	if err != nil {
		return Target{}, err
	}
	pc.Unlearned = UnlearnedRefs(prereqs)
	catalog, err := s.Store.ConceptCatalog(ctx, userID)
	if err != nil {
		return Target{}, err
	}
	pc.Catalog = CatalogRefs(catalog, conceptID, pc.Linkable)
	var personal personalContext
	if personal.weak, err = s.weakConcepts(ctx, userID); err != nil {
		return Target{}, err
	}
	if personal.misconceptions, err = s.promptMisconceptions(ctx, userID, conceptID); err != nil {
		return Target{}, err
	}

	return Target{
		Plan:     plan,
		Concept:  concept,
		Prompt:   pc,
		personal: personal,
		Validation: lecture.ValidationContext{
			KnownConceptIDs:     known,
			PlannedObjectiveIDs: plan.ObjectiveIDs,
			AllowedObjectiveIDs: allowed,
		},
	}, nil
}

// exampleLanguageName is the display name of the user's example-code language.
func (s *Service) exampleLanguageName(ctx context.Context, userID string) string {
	return learning.ExampleLanguages[s.Store.ExampleLanguageFor(ctx, userID)]
}

// parseAndValidate turns a new draft's markup into content and its issues:
// structural validation, then the machine checks of quality design v2.2 §4
// (self links, meta phrases, material blocks, volume). Stored lectures never
// go through this.
func (t Target) parseAndValidate(markup string) (lecture.Content, []lecture.Issue, error) {
	content, err := lecture.ParseMarkup(markup, t.Plan.ConceptID, t.Plan.Depth)
	if err != nil {
		return content, nil, err
	}
	lecture.NormalizeInlineTerms(&content, t.Validation.KnownConceptIDs)
	issues := lecture.Validate(content, t.Validation)
	return content, append(issues, lecture.MachineCheck(content)...), nil
}
