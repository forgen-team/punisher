package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
)

const (
	libraryAttempts   = 2 // first draft + one regeneration
	libraryJobTimeout = 15 * time.Minute
	defaultMaxTokens  = 16000
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

// GenerateLibraryAhead keeps the owner's next N (Cfg.LibraryAhead) recommended
// lectures ready. Candidates are today's recommendations followed by the
// frontier; done objectives are never planned. Idempotent per (objective set,
// depth, KST day). LibraryAhead <= 0 disables ahead pre-generation.
func (s *Service) GenerateLibraryAhead(ctx context.Context) (LibraryReport, error) {
	owner := s.Cfg.OwnerUserID
	if owner == "" || s.Cfg.LibraryAhead <= 0 {
		return LibraryReport{}, nil
	}
	candidates, err := s.libraryCandidates(ctx, owner)
	if err != nil {
		return LibraryReport{}, err
	}
	return s.generateLibrary(ctx, owner, candidates, s.Cfg.LibraryAhead, "library_ahead")
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
		if _, err := s.Store.LatestLectureForTarget(ctx, learning.JobLibrary, plan.ConceptID, plan.ObjectiveKey, plan.Depth, libraryStatuses); err == nil {
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
			Reason: reason, Input: input,
		})
		if errors.Is(err, store.ErrJobActive) || (err == nil && !created) {
			rep.Skipped = append(rep.Skipped, plan.ObjectiveKey)
			continue
		}
		if err != nil {
			return rep, err
		}
		lec, err := s.runLibraryJob(ctx, apiKey, job.ID, target)
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
	data, err := s.Store.LoadToday(ctx, userID)
	if err != nil {
		return nil, err
	}
	in := todayInput(data, 20)
	today := learning.BuildToday(in)
	seen := map[string]bool{}
	out := []string{}
	for _, r := range today.Items {
		if r.Kind != learning.RecResume && !seen[r.ConceptID] {
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
		if c := data.Concepts[f.ID]; !seen[f.ID] && c.HasPending && !c.Proposed {
			seen[f.ID] = true
			out = append(out, f.ID)
		}
	}
	return out, nil
}

// runLibraryJob: outline → lecture → review → validate (→ one regeneration).
// LLM errors mark the job failed and store nothing.
func (s *Service) runLibraryJob(ctx context.Context, apiKey, jobID string, t Target) (models.Lecture, error) {
	ctx, cancel := context.WithTimeout(ctx, libraryJobTimeout)
	defer cancel()
	if err := s.Store.StartJob(ctx, jobID); err != nil {
		return models.Lecture{}, err
	}
	var u usage
	fail := func(err error) (models.Lecture, error) {
		if ferr := s.Store.FinishJob(context.WithoutCancel(ctx), jobID, u.outcome("failed", "", err, "")); ferr != nil {
			logf("finish job %s: %v", jobID, ferr)
		}
		return models.Lecture{}, err
	}

	vars := prompts.LectureVars(t.Prompt)
	teacher := t.Prompts.Text(prompts.BlockTeacherSystem, vars)
	outline, err := s.complete(ctx, apiKey, llm.MessagesRequest{
		Model: s.Cfg.LectureModel, System: teacher, User: prompts.Outline(t.Prompt, t.Prompts), MaxTokens: 4000,
	}, &u)
	if err != nil {
		return fail(fmt.Errorf("outline: %w", err))
	}
	maxTokens := s.Cfg.LectureMaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	result, err := RunReviewLoop(ctx, libraryAttempts, LoopFuncs{
		Generate: func(ctx context.Context, notes string) (string, error) {
			return s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.LectureModel, System: teacher,
				User: prompts.Lecture(t.Prompt, outline, notes, t.Prompts), MaxTokens: maxTokens, Timeout: 6 * time.Minute}, &u)
		},
		Review: func(ctx context.Context, markup string) (ReviewOutcome, error) {
			text, err := s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.LectureModel,
				System: t.Prompts.Text(prompts.BlockReviewerSystem, vars),
				User:   prompts.Review(t.Prompt, markup, t.Prompts), MaxTokens: 4000, Timeout: 4 * time.Minute}, &u)
			if err != nil {
				return ReviewOutcome{}, err
			}
			return ParseReview(text), nil
		},
		Parse: t.parseAndValidate,
	})
	if err != nil {
		return fail(err)
	}

	final, err := s.storeAttempts(ctx, t, jobID, result, u.model)
	if err != nil {
		return fail(fmt.Errorf("store lecture: %w", err))
	}
	if err := s.Store.FinishJob(ctx, jobID, u.outcome("succeeded", result.Status, nil, final.ID)); err != nil {
		logf("finish job %s: %v", jobID, err)
	}
	return final, nil
}

// storeAttempts saves every parsed attempt as an immutable version (earlier
// ones archived) with its review; the final one is published/needs_revision.
func (s *Service) storeAttempts(ctx context.Context, t Target, jobID string, res LoopResult, model string) (models.Lecture, error) {
	var final models.Lecture
	for i, a := range res.Attempts {
		if a.ParseErr != nil {
			continue
		}
		status := "archived"
		if i == res.Final {
			status = res.Status
		}
		kind := learning.JobLibrary
		if t.Owner != "" {
			kind = learning.JobInstant
		}
		saved, err := s.saveLecture(ctx, t, a.Content, kind, status, t.Owner, jobID, model, false, "")
		if err != nil {
			return final, err
		}
		verdict := "fail"
		if a.Passed() {
			verdict = "pass"
		}
		if err := s.Store.InsertReview(ctx, saved.ID, models.LectureReview{
			Verdict: verdict, Summary: a.Review.Summary, Issues: issuesJSON(a.Review.Issues),
			ValidationIssues: issuesJSON(a.Issues), Attempt: a.Number, Model: model, Reviewer: "llm",
		}); err != nil {
			return final, err
		}
		if i == res.Final {
			final = saved
		}
	}
	return final, nil
}

func (s *Service) saveLecture(ctx context.Context, t Target, c lecture.Content, kind, status, requestedBy, jobID, model string, flagged bool, flagReason string) (models.Lecture, error) {
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
		Flagged: flagged, FlagReason: flagReason, EstimatedMinutes: c.TotalMinutes(), GenerationJobID: jobID,
	})
}

// todayInput converts store data into learning.TodayInput.
func todayInput(d *store.TodayData, limit int) learning.TodayInput {
	depthOf := make(map[string]int, len(d.Concepts))
	for id, c := range d.Concepts {
		depthOf[id] = c.Depth
	}
	in := learning.TodayInput{Concepts: d.Concepts, Edges: d.Edges, Resume: d.Resume, Requests: d.Requests, Limit: limit}
	for _, g := range d.Goals {
		for _, target := range g.Targets {
			in.Goals = append(in.Goals, learning.GoalInput{ID: g.ID, Title: g.Title, Path: learning.PathTo(target, d.Edges, depthOf)})
		}
	}
	return in
}

// Today builds the user's recommendations.
func (s *Service) Today(ctx context.Context, userID string, limit int) (learning.TodayResult, error) {
	data, err := s.Store.LoadToday(ctx, userID)
	if err != nil {
		return learning.TodayResult{}, err
	}
	return learning.BuildToday(todayInput(data, limit)), nil
}
