package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
)

const (
	libraryAttempts   = 2                                             // first draft + one regeneration
	libraryJobTimeout = libraryAttempts*attemptBudget + 5*time.Minute // + outline
	defaultMaxTokens  = 24000
)

// libraryStatuses count as "already exists" — published or waiting for human review.
var libraryStatuses = []string{"published", "needs_revision"}

// LibraryReport summarises one pre-generation run.
type LibraryReport struct {
	Ready     int      `json:"ready"`
	Generated []string `json:"generated"`
	Failed    []string `json:"failed"`
	Skipped   []string `json:"skipped"`
}

// GenerateLibraryAhead keeps the owner's next N recommended lectures ready:
// Cfg.LibraryAhead during the day, Cfg.NightAhead (when larger) in the KST
// night window, so the morning list opens without waiting. Candidates are
// today's recommendations followed by the frontier; done objectives are never
// planned. Idempotent per (objective set, depth, language, KST day). N <= 0
// disables ahead pre-generation.
func (s *Service) GenerateLibraryAhead(ctx context.Context) (LibraryReport, error) {
	owner := s.Cfg.OwnerUserID
	want, reason := s.libraryWant()
	if owner == "" || want <= 0 {
		return LibraryReport{}, nil
	}
	candidates, err := s.libraryCandidates(ctx, owner)
	if err != nil {
		return LibraryReport{}, err
	}
	return s.generateLibrary(ctx, owner, candidates, want, reason)
}

// PregenerateStudyStarts pre-generates the library lecture for each resolved
// study request's starting concept so the first lecture is ready when the
// user opens it. Independent of LibraryAhead (Cfg.StudyPregenerate) and
// bounded by the daily lecture/token budget like every library job.
func (s *Service) PregenerateStudyStarts(ctx context.Context, conceptIDs []string) (LibraryReport, error) {
	owner := s.Cfg.OwnerUserID
	if owner == "" || !s.Cfg.StudyPregenerate || len(conceptIDs) == 0 {
		return LibraryReport{}, nil
	}
	return s.generateLibrary(ctx, owner, conceptIDs, len(conceptIDs), "study_request")
}

// generateLibrary ensures up to `want` of candidates have a reviewed library
// lecture for today's plan, generating missing ones in order.
func (s *Service) generateLibrary(ctx context.Context, owner string, candidates []string, want int, reason string) (LibraryReport, error) {
	var rep LibraryReport
	apiKey, err := s.Keys.UserKey(ctx, owner)
	if err != nil {
		return rep, err
	}
	day := learning.KSTDay(s.now())
	for _, conceptID := range candidates {
		if rep.Ready >= want {
			break
		}
		target, err := s.libraryTarget(ctx, owner, conceptID)
		if errors.Is(err, learning.ErrNothingToLearn) {
			continue
		}
		if err != nil {
			return rep, err
		}
		plan := target.Plan
		if _, err := s.Store.LatestLectureForTarget(ctx, learning.JobLibrary, plan.ConceptID, plan.ObjectiveKey, plan.Depth, plan.Language, plan.ReaderLevel, libraryStatuses); err == nil {
			rep.Ready++
			continue
		} else if !errors.Is(err, store.ErrNotFound) {
			return rep, err
		}
		if err := s.CheckBudget(ctx, owner, true); err != nil {
			return rep, err
		}
		input, _ := json.Marshal(map[string]any{"objectiveIds": plan.ObjectiveIDs})
		job, created, err := s.Store.CreateJob(ctx, store.NewJob{
			IdempotencyKey: learning.LibraryJobKey(plan, day), Kind: learning.JobLibrary, UserID: owner,
			ConceptID: plan.ConceptID, ObjectiveKey: plan.ObjectiveKey, Depth: plan.Depth,
			Reason: reason, Input: input, Language: plan.Language, ReaderLevel: plan.ReaderLevel,
		})
		if errors.Is(err, store.ErrJobActive) || (err == nil && !created) {
			rep.Skipped = append(rep.Skipped, plan.ObjectiveKey)
			continue
		}
		if err != nil {
			return rep, err
		}
		lec, err := s.runLibraryJob(ctx, apiKey, job.ID, target, nil)
		if err != nil {
			logf("library job %s (%s) failed: %v", job.ID, plan.ObjectiveKey, err)
			rep.Failed = append(rep.Failed, plan.ObjectiveKey)
			// Gateway down / CLI logged out / usage limit: stop this run
			// instead of failing every remaining candidate.
			if errors.Is(err, llm.ErrGenerationUnavailable) || errors.Is(err, llm.ErrRateLimited) {
				return rep, err
			}
			continue
		}
		rep.Generated = append(rep.Generated, lec.ID)
		rep.Ready++
	}
	return rep, nil
}

func (s *Service) libraryCandidates(ctx context.Context, userID string) ([]string, error) {
	data, err := s.Store.LoadToday(ctx, userID, s.now())
	if err != nil {
		return nil, err
	}
	in := todayInput(data, 20)
	in.ReaderLevel = s.Store.ReaderLevelFor(ctx, userID)
	hideFoundations := learning.FoundationsHidden(in.ReaderLevel)
	today := learning.BuildToday(in)
	seen := map[string]bool{}
	out := []string{}
	for _, r := range today.Items {
		if r.Kind != learning.RecResume && !seen[r.ConceptID] && !data.Concepts[r.ConceptID].Private {
			seen[r.ConceptID] = true
			out = append(out, r.ConceptID)
		}
	}
	nodes := make([]learning.Node, 0, len(data.Concepts))
	states := map[string]string{}
	for _, c := range data.Concepts {
		nodes = append(nodes, learning.Node{ID: c.ID, Depth: c.Depth})
		states[c.ID] = c.State
	}
	for _, f := range learning.Analyze(nodes, data.Edges, states).Frontier {
		c := data.Concepts[f.ID]
		if hideFoundations && c.Area == curriculum.AreaFoundations {
			continue // 숨김 수준에는 0단계 강의를 미리 만들지 않는다.
		}
		if !seen[f.ID] && c.HasPending && !c.Proposed && !c.Private && !c.Dismissed {
			seen[f.ID] = true
			out = append(out, f.ID)
		}
	}
	return out, nil
}

// runLibraryJob: outline → lecture → validate → review (→ one regeneration).
// LLM errors mark the job failed and store nothing. onStage (optional)
// receives the loop's stages (regenerate streams them).
func (s *Service) runLibraryJob(ctx context.Context, apiKey, jobID string, t Target, onStage func(string, int)) (models.Lecture, error) {
	ctx, cancel := context.WithTimeout(ctx, libraryJobTimeout)
	defer cancel()
	if err := s.Store.StartJob(ctx, jobID); err != nil {
		return models.Lecture{}, err
	}
	var u usage
	var result LoopResult
	fail := func(err error) (models.Lecture, error) {
		o := u.outcome("failed", "", err, "")
		if len(result.Attempts) > 0 {
			o.Attempts = result.AttemptLog()
		}
		if ferr := s.Store.FinishJob(context.WithoutCancel(ctx), jobID, o); ferr != nil {
			logf("finish job %s: %v", jobID, ferr)
		}
		return models.Lecture{}, err
	}

	pc := t.LectureContext()
	outline, err := s.complete(ctx, apiKey, llm.MessagesRequest{
		Model: s.Cfg.LectureModel, System: prompts.WithOutputLanguage(t.Prompts.Text(prompts.BlockTeacherSystem, prompts.LectureVars(pc)), t.Plan.Language),
		User: prompts.Outline(pc, t.Prompts), MaxTokens: 4000,
	}, &u)
	if err != nil {
		return fail(fmt.Errorf("outline: %w", err))
	}
	loop := s.lectureLoop(apiKey, t, outline, &u)
	loop.OnStage = onStage
	result, err = RunReviewLoop(ctx, libraryAttempts, loop)
	if err != nil {
		return fail(err)
	}

	final, err := s.storeAttempts(ctx, t, jobID, result, u.model, u.provider)
	if err != nil {
		return fail(fmt.Errorf("store lecture: %w", err))
	}
	done := u.outcome("succeeded", result.Status, nil, final.ID)
	done.Attempts = result.AttemptLog()
	if err := s.Store.FinishJob(ctx, jobID, done); err != nil {
		logf("finish job %s: %v", jobID, err)
	}
	return final, nil
}

// storeAttempts saves every parsed attempt as an immutable version (earlier
// ones archived) with its review; the final one is published/needs_revision.
func (s *Service) storeAttempts(ctx context.Context, t Target, jobID string, res LoopResult, model, provider string) (models.Lecture, error) {
	var final models.Lecture
	// The final attempt is saved last so it is the newest version of the target.
	order := make([]int, 0, len(res.Attempts))
	for i, a := range res.Attempts {
		if a.drafted() && i != res.Final {
			order = append(order, i)
		}
	}
	if res.Final >= 0 {
		order = append(order, res.Final)
	}
	for _, i := range order {
		a := res.Attempts[i]
		status := "archived"
		if i == res.Final {
			status = res.Status
		}
		kind := learning.JobLibrary
		if t.Owner != "" {
			kind = learning.JobInstant
		}
		saved, err := s.saveLecture(ctx, t, a.Content, kind, status, t.Owner, jobID, model, provider)
		if err != nil {
			return final, err
		}
		verdict := "fail"
		if a.Passed() {
			verdict = "pass"
		}
		if err := s.Store.InsertReview(ctx, saved.ID, models.LectureReview{
			Verdict: verdict, Summary: a.Review.Summary, Issues: issuesJSON(a.Review.Issues),
			ValidationIssues: issuesJSON(a.Issues), ReaderGaps: issuesJSON(a.Review.ReaderGaps),
			Attempt: a.Number, Model: model, Reviewer: "llm",
		}); err != nil {
			return final, err
		}
		if i == res.Final {
			final = saved
		}
	}
	return final, nil
}

func (s *Service) saveLecture(ctx context.Context, t Target, c lecture.Content, kind, status, requestedBy, jobID, model, provider string) (models.Lecture, error) {
	c.EstimatedMinutes = learning.EstimateMinutes(c) // 표시 시간은 본문 분량으로 계산한다(섹션 minutes 아님, v2.2 §5).
	hash, err := lecture.Hash(c)
	if err != nil {
		return models.Lecture{}, err
	}
	content, err := json.Marshal(c)
	if err != nil {
		return models.Lecture{}, err
	}
	sources, err := json.Marshal(c.AllSources())
	if err != nil {
		return models.Lecture{}, err
	}
	return s.Store.InsertLecture(ctx, store.NewLecture{
		ConceptID: t.Plan.ConceptID, ObjectiveIDs: t.Plan.ObjectiveIDs, ObjectiveKey: t.Plan.ObjectiveKey,
		Depth: t.Plan.Depth, Kind: kind, Status: status, Title: c.Title, Content: content, ContentHash: hash,
		Sources: sources, Model: model, PromptVersion: t.promptVersion(), RequestedBy: requestedBy,
		EstimatedMinutes: c.EstimatedMinutes, GenerationJobID: jobID,
		Language: t.Plan.Language, ReaderLevel: t.Plan.ReaderLevel, Provider: provider,
	})
}

// libraryWant is how many recommended lectures to keep ready right now and the
// job reason recorded for them ("library_night" inside the night window).
func (s *Service) libraryWant() (int, string) {
	c := s.Cfg
	if c.NightAhead > c.LibraryAhead && c.NightStartHour != c.NightEndHour &&
		learning.InKSTHours(s.now(), c.NightStartHour, c.NightEndHour) {
		return c.NightAhead, "library_night"
	}
	return c.LibraryAhead, "library_ahead"
}
