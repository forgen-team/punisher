package services

import (
	"context"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/store"

	"github.com/wooo-jin/punisher/internal/prompts"
)

// UserPrompts is the user's effective instruction blocks (overrides on top of
// the code defaults). A load failure falls back to the defaults and is logged:
// generation must not fail because of a settings read.
func (s *Service) UserPrompts(ctx context.Context, userID string) prompts.Set {
	if userID == "" {
		return prompts.Defaults()
	}
	overrides, err := s.Store.PromptOverrides(ctx, userID)
	if err != nil {
		logf("prompt overrides for %s: %v (using defaults)", userID, err)
		return prompts.Defaults()
	}
	bodies := make(map[string]string, len(overrides))
	for id, o := range overrides {
		bodies[id] = o.Body
	}
	return prompts.NewSet(bodies)
}

// interactionVars are the {{vars}} of Q&A blocks.
func (s *Service) interactionVars(ctx context.Context, userID, conceptID string) map[string]string {
	vars := map[string]string{"example_language": s.exampleLanguageName(ctx, userID)}
	if info, err := s.Store.ConceptInfo(ctx, userID, conceptID); err == nil {
		vars["concept_name"] = info.Name
	}
	return vars
}

// libraryTarget plans a library lecture. In witems library lectures are shared
// across users, so they always use the code default prompts (overrides are
// personal). The standalone app has one user who is the only reader of the
// library, so with Cfg.LibraryUsesUserPrompts the owner's blocks apply too
// (the lecture stays kind=library; its prompt_version records the override).
func (s *Service) libraryTarget(ctx context.Context, owner, conceptID string) (Target, error) {
	t, err := s.BuildTarget(ctx, owner, conceptID, 0, "")
	if err != nil {
		return t, err
	}
	set := prompts.Defaults()
	if s.Cfg.LibraryUsesUserPrompts && owner != "" {
		set = s.UserPrompts(ctx, owner)
	}
	t.Prompts, t.PromptBlocks, t.Owner = set, prompts.FullBlocks, ""
	return t, nil
}

// instantTarget plans the user's own instant lecture with their prompt blocks.
func (s *Service) instantTarget(ctx context.Context, userID, conceptID string, depth int, query string) (Target, error) {
	t, err := s.BuildTarget(ctx, userID, conceptID, depth, query)
	if err != nil {
		return t, err
	}
	t.Prompts, t.PromptBlocks, t.Owner = s.UserPrompts(ctx, userID), prompts.InstantBlocks, userID
	return t, nil
}

// UnlearnedRefs keeps the prerequisites the learner has not completed/known —
// the concepts a lecture must bridge where it uses them.
func UnlearnedRefs(prereqs []store.ConceptRef) []prompts.ConceptRef {
	out := []prompts.ConceptRef{}
	for _, p := range prereqs {
		if !learning.IsDone(p.State) {
			out = append(out, prompts.ConceptRef{ID: p.ID, Name: p.Name})
		}
	}
	return out
}

// CatalogRefs lists every concept except the target and the already listed
// neighbours, for inline links to names mentioned in passing.
func CatalogRefs(catalog []store.ConceptRef, targetID string, listed []prompts.ConceptRef) []prompts.CatalogRef {
	skip := map[string]bool{targetID: true}
	for _, r := range listed {
		skip[r.ID] = true
	}
	out := []prompts.CatalogRef{}
	for _, c := range catalog {
		if !skip[c.ID] {
			out = append(out, prompts.CatalogRef{ID: c.ID, Name: c.Name, Area: c.Area, Learned: learning.IsDone(c.State)})
		}
	}
	return out
}
