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
}

// BuildTarget plans the next lecture for (user, concept, depth) and gathers
// prompt/validation context. Returns learning.ErrNothingToLearn when every
// objective at that depth is completed/known (dedup invariant).
func (s *Service) BuildTarget(ctx context.Context, userID, conceptID string, depth int, query string) (Target, error) {
	concept, err := s.Store.ConceptInfo(ctx, conceptID)
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

	related, err := s.Store.RelatedConcepts(ctx, userID, conceptID)
	if err != nil {
		return Target{}, err
	}
	known, err := s.Store.KnownConceptIDs(ctx)
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

	return Target{
		Plan:    plan,
		Concept: concept,
		Prompt:  pc,
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

// parseAndValidate turns markup into content and structural issues.
func (t Target) parseAndValidate(markup string) (lecture.Content, []lecture.Issue, error) {
	content, err := lecture.ParseMarkup(markup, t.Plan.ConceptID, t.Plan.Depth)
	if err != nil {
		return content, nil, err
	}
	return content, lecture.Validate(content, t.Validation), nil
}
