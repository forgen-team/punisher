package services

import (
	"context"
	"log"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/quality"
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
	// Quality: what the rubric checks against (allowed source hosts).
	Quality quality.Context
}

// promptVersion is the version recorded on lectures saved from t.
func (t Target) promptVersion() string {
	return t.Prompts.Version(t.PromptBlocks...)
}

// BuildTarget plans the next lecture for (user, concept, depth) and gathers
// prompt/validation context. Returns learning.ErrNothingToLearn when every
// objective at that depth is completed/known (dedup invariant).
func (s *Service) BuildTarget(ctx context.Context, userID, conceptID string, depth int, query string) (Target, error) {
	concept, plan, err := s.planLecture(ctx, userID, conceptID, depth)
	if err != nil {
		return Target{}, err
	}
	return s.targetForPlan(ctx, userID, concept, plan, query)
}

// planLecture is the next lecture plan of the concept for the user (learning.PlanLecture).
func (s *Service) planLecture(ctx context.Context, userID, conceptID string, depth int) (models.ConceptInfo, learning.LecturePlan, error) {
	lang := s.Store.LocaleFor(ctx, userID) // 새로 만드는 강의는 사용자 언어(i18n §1.2).
	concept, err := s.Store.ConceptInfo(ctx, userID, conceptID, lang)
	if err != nil {
		return concept, learning.LecturePlan{}, err
	}
	done, err := s.Store.DoneSet(ctx, userID, conceptID)
	if err != nil {
		return concept, learning.LecturePlan{}, err
	}
	refs := make([]learning.ObjectiveRef, 0, len(concept.Objectives))
	for _, o := range concept.Objectives {
		refs = append(refs, learning.ObjectiveRef{ID: o.ID, Depth: o.Depth, Ordinal: o.Ordinal})
	}
	plan, err := learning.PlanLecture(conceptID, refs, done, depth)
	plan.Language = lang
	plan.ReaderLevel = s.Store.ReaderLevelFor(ctx, userID) // 새로 만드는 강의는 사용자 수준(learner-level ①).
	return concept, plan, err
}

// TargetForLecture rebuilds the generation target of an existing lecture (same
// concept, objectives, depth, language and reader level) regardless of what the
// user already learned. lang / level "" keep the lecture's; '다시 만들기' passes
// the user's (RegenerateLecture), so names, objectives, the output rule and the
// reader section follow them.
func (s *Service) TargetForLecture(ctx context.Context, userID string, lec models.Lecture, lang, level string) (Target, error) {
	if lang == "" {
		lang = lec.Language
	}
	if level == "" {
		level = lec.ReaderLevel
	}
	lang, level = learning.NormalizeLocale(lang), learning.EffectiveReaderLevel(level)
	concept, err := s.Store.ConceptInfo(ctx, userID, lec.ConceptID, lang)
	if err != nil {
		return Target{}, err
	}
	plan := learning.LecturePlan{
		ConceptID: lec.ConceptID, ObjectiveIDs: lec.ObjectiveIDs,
		ObjectiveKey: learning.ObjectiveKey(lec.ObjectiveIDs), Depth: lec.Depth, Language: lang, ReaderLevel: level,
	}
	return s.targetForPlan(ctx, userID, concept, plan, "")
}

func (s *Service) targetForPlan(ctx context.Context, userID string, concept models.ConceptInfo, plan learning.LecturePlan, query string) (Target, error) {
	conceptID, lang := concept.ID, plan.Language
	related, err := s.Store.RelatedConcepts(ctx, userID, conceptID, lang)
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
		Query: query, ExampleLanguage: s.exampleLanguageName(ctx, userID), Glossary: s.glossaryRefs(lang),
		Reader: prompts.Reader{Level: learning.EffectiveReaderLevel(plan.ReaderLevel), Locale: lang},
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
	prereqs, err := s.Store.PrerequisitesOf(ctx, userID, conceptID, lang)
	if err != nil {
		return Target{}, err
	}
	pc.Unlearned = UnlearnedRefs(prereqs)
	catalog, err := s.Store.ConceptCatalog(ctx, userID, lang)
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
	personal.labOS = s.labOS(ctx, userID)
	if personal.repeated, err = s.Store.RepeatedMisconceptions(ctx, userID, conceptID, minRepeatedMisconception, maxRepeatedMisconceptions); err != nil {
		logf("repeated misconceptions of %s: %v", conceptID, err)
		personal.repeated = nil
	}

	return Target{
		Plan:     plan,
		Concept:  concept,
		Prompt:   pc,
		personal: personal,
		Quality:  quality.Context{AllowedHosts: s.SourceHosts},
		Validation: lecture.ValidationContext{
			KnownConceptIDs:     s.withGlossary(known),
			PlannedObjectiveIDs: plan.ObjectiveIDs,
			AllowedObjectiveIDs: allowed,
		},
	}, nil
}

// readerFor is the reader section input for the user's effective level in output language lang.
func (s *Service) readerFor(ctx context.Context, userID, lang string) prompts.Reader {
	return prompts.Reader{Level: s.Store.ReaderLevelFor(ctx, userID), Locale: learning.NormalizeLocale(lang)}
}

// exampleLanguageName is the display name of the user's example-code language.
func (s *Service) exampleLanguageName(ctx context.Context, userID string) string {
	return learning.ExampleLanguages[s.Store.ExampleLanguageFor(ctx, userID)]
}

// parseAndValidate turns a new draft's markup into content and its issues:
// structural validation, then the machine checks of quality design v2.2 §4
// (self links, meta phrases, material blocks, volume) and the rewrite rules
// of the quality rubric (advisory). Stored lectures never go through this.
func (t Target) parseAndValidate(markup string) (lecture.Content, []lecture.Issue, error) {
	content, err := lecture.ParseMarkup(markup, t.Plan.ConceptID, t.Plan.Depth)
	if err != nil {
		return content, nil, err
	}
	lecture.NormalizeInlineTerms(&content, t.Validation.KnownConceptIDs)
	issues := append(lecture.Validate(content, t.Validation), lecture.MachineCheck(content)...)
	return content, append(issues, rubricIssues(content, t.Quality)...), nil
}

// withGlossary adds the glossary term ids (term:g.…) to the link targets a lecture may use.
func (s *Service) withGlossary(known map[string]bool) map[string]bool {
	ids := s.Glossary.IDs()
	if len(ids) == 0 {
		return known
	}
	out := make(map[string]bool, len(known)+len(ids))
	for id, ok := range known {
		out[id] = ok
	}
	for _, id := range ids {
		out[id] = true
	}
	return out
}

// glossaryRefs are the basic terms for the lecture prompt, named in the lecture language.
func (s *Service) glossaryRefs(lang string) []prompts.GlossaryRef {
	terms := s.Glossary.TermsIn(lang)
	out := make([]prompts.GlossaryRef, 0, len(terms))
	for _, t := range terms {
		out = append(out, prompts.GlossaryRef{ID: t.ID, Name: t.Name})
	}
	return out
}
