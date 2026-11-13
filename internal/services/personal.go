package services

import (
	"context"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
)

// personalContext is the learner's own data a lecture prompt may use (design
// §5 P4): weak concepts and acknowledged misconceptions. Target.LectureContext
// applies it only to user-owned generations (M6).
type personalContext struct {
	weak           map[string]bool
	misconceptions []prompts.Misconception
}

// LectureContext is the prompt context of t. User-owned generations (Owner set:
// instant / regenerated) move weak concepts out of "배운 개념" and Catalog ✓ into
// "다시 짚을 개념" and add the acknowledged misconceptions; library lectures
// (Owner "") get the shared context unchanged (M6).
func (t Target) LectureContext() prompts.LectureContext {
	if t.Owner == "" {
		return t.Prompt
	}
	return personalize(t.Prompt, t.personal)
}

func personalize(pc prompts.LectureContext, p personalContext) prompts.LectureContext {
	var revisit []prompts.ConceptRef
	pc.Learned, revisit = splitLearned(pc.Learned, p.weak)
	var fromCatalog []prompts.ConceptRef
	pc.Catalog, fromCatalog = splitCatalog(pc.Catalog, p.weak)
	pc.Revisit = mergeRefs(revisit, fromCatalog)
	pc.Misconceptions = p.misconceptions
	return pc
}

// splitLearned moves the weak concepts out of learned (each concept stays in one list).
func splitLearned(learned []prompts.ConceptRef, weak map[string]bool) (kept, revisit []prompts.ConceptRef) {
	kept = []prompts.ConceptRef{}
	for _, r := range learned {
		if weak[r.ID] {
			revisit = append(revisit, r)
		} else {
			kept = append(kept, r)
		}
	}
	return kept, revisit
}

// splitCatalog moves learned (✓) weak concepts out of the catalog.
func splitCatalog(catalog []prompts.CatalogRef, weak map[string]bool) (kept []prompts.CatalogRef, revisit []prompts.ConceptRef) {
	kept = []prompts.CatalogRef{}
	for _, c := range catalog {
		if c.Learned && weak[c.ID] {
			revisit = append(revisit, prompts.ConceptRef{ID: c.ID, Name: c.Name})
		} else {
			kept = append(kept, c)
		}
	}
	return kept, revisit
}

func mergeRefs(a, b []prompts.ConceptRef) []prompts.ConceptRef {
	seen := map[string]bool{}
	out := []prompts.ConceptRef{}
	for _, r := range append(a, b...) {
		if !seen[r.ID] {
			seen[r.ID] = true
			out = append(out, r)
		}
	}
	return out
}

// weakConcepts classifies the user's evidenced concepts (learning.WeakConcept).
func (s *Service) weakConcepts(ctx context.Context, userID string) (map[string]bool, error) {
	list, err := s.Store.EvidencedMastery(ctx, userID)
	if err != nil {
		return nil, err
	}
	byConcept := map[string][]learning.ObjectiveLevel{}
	for _, o := range list {
		byConcept[o.ConceptID] = append(byConcept[o.ConceptID], learning.ObjectiveLevel{Level: o.Level, Best: o.Best, Lapses: o.Lapses})
	}
	weak := map[string]bool{}
	for id, objs := range byConcept {
		if learning.WeakConcept(objs) {
			weak[id] = true
		}
	}
	return weak, nil
}

// promptMisconceptions: open acknowledged misconceptions of conceptID and its
// prerequisites ("" = the latest of any concept), at most learning.MaxPromptMisconceptions.
func (s *Service) promptMisconceptions(ctx context.Context, userID, conceptID string) ([]prompts.Misconception, error) {
	list, err := s.Store.OpenMisconceptions(ctx, userID, conceptID, learning.MaxPromptMisconceptions)
	if err != nil {
		return nil, err
	}
	return misconceptionRefs(list), nil
}

func misconceptionRefs(list []models.Misconception) []prompts.Misconception {
	out := make([]prompts.Misconception, 0, len(list))
	for _, m := range list {
		out = append(out, prompts.Misconception{ConceptName: m.ConceptName, Text: truncateRunes(m.Text, learning.MaxMisconceptionRunes)})
	}
	return out
}
