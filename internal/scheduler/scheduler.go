// Package scheduler runs Punisher's background jobs for the owner account:
// stale-job recovery, study-request resolution and library pre-generation.
// All work is idempotent (generation_jobs keys), so restarts never duplicate.
package scheduler

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/services"
)

const staleJobAge = 20 * time.Minute

// inquirySweepInterval: how often interrupted question explanations are ended.
const inquirySweepInterval = 2 * time.Minute

// Scheduler ticks every Interval after an initial delay.
type Scheduler struct {
	Svc          *services.Service
	Interval     time.Duration
	InitialDelay time.Duration
}

// Run blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	go s.sweepInquiries(ctx)
	go s.sweepChecks(ctx)
	if s.Svc.Cfg.OwnerUserID == "" {
		log.Println("[scheduler] PUNISHER_OWNER_USER_ID not set: background generation disabled")
		return
	}
	timer := time.NewTimer(s.InitialDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.tick(ctx)
			timer.Reset(s.Interval)
		}
	}
}

// sweepInquiries ends question explanations whose generation is gone: at boot
// every pending row belongs to the previous process, and afterwards anything
// older than services.StaleInquiryAge. Runs without an owner too.
func (s *Scheduler) sweepInquiries(ctx context.Context) {
	sweep := func(age time.Duration) {
		if n, err := s.Svc.Store.SweepStaleInquiries(ctx, age); err != nil {
			log.Printf("[scheduler] sweep inquiries: %v", err)
		} else if n > 0 {
			log.Printf("[scheduler] ended %d interrupted inquiries", n)
		}
	}
	sweep(0)
	ticker := time.NewTicker(inquirySweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep(services.StaleInquiryAge)
		}
	}
}

// sweepChecks ends mastery-check work whose run is gone (design §4.3): at boot
// every pending turn and preparing session belongs to the previous process;
// afterwards pending turns over services.StaleCheckTurnAge and preparing
// sessions over services.StaleCheckPrepareAge fail, and sessions idle for
// services.AbandonCheckAge are abandoned (applied evidence stays). Runs without an owner too.
func (s *Scheduler) sweepChecks(ctx context.Context) {
	sweep := func(turnAge, prepareAge time.Duration) {
		if n, err := s.Svc.Store.SweepStaleChecks(ctx, turnAge, prepareAge, services.AbandonCheckAge); err != nil {
			log.Printf("[scheduler] sweep checks: %v", err)
		} else if n > 0 {
			log.Printf("[scheduler] ended %d interrupted check rows", n)
		}
	}
	sweep(0, 0)
	ticker := time.NewTicker(inquirySweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep(services.StaleCheckTurnAge, services.StaleCheckPrepareAge)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	// Every generation below (study-request resolution, library lectures and
	// their review pass) is background work: it must never block a user's
	// interactive call on the local claude CLI.
	ctx = llm.WithBackground(ctx)
	if n, err := s.Svc.Store.FailStaleJobs(ctx, staleJobAge); err != nil {
		log.Printf("[scheduler] stale jobs: %v", err)
	} else if n > 0 {
		log.Printf("[scheduler] marked %d stale jobs failed", n)
	}

	n, starts, err := s.Svc.ResolveStudyRequestsWithStarts(ctx)
	if err != nil {
		log.Printf("[scheduler] study requests: resolved %d, stopped: %v", n, err)
	} else if n > 0 {
		log.Printf("[scheduler] study requests: resolved %d", n)
	}
	// Study-request pre-generation is separate from PUNISHER_LIBRARY_AHEAD
	// (PUNISHER_STUDY_PREGENERATE); both are bounded by the daily budget.
	if len(starts) > 0 {
		rep, err := s.Svc.PregenerateStudyStarts(ctx, starts)
		logLibrary("study pregenerate", rep, err)
	}
	if s.Svc.Cfg.LibraryAhead <= 0 {
		return
	}
	rep, err := s.Svc.GenerateLibraryAhead(ctx)
	logLibrary("library", rep, err)
}

func logLibrary(label string, rep services.LibraryReport, err error) {
	switch {
	case errors.Is(err, services.ErrNoAPIKey):
		log.Printf("[scheduler] %s: no API key; skipped", label)
	case errors.Is(err, services.ErrBudgetExceeded):
		log.Printf("[scheduler] %s: %v (ready %d)", label, err, rep.Ready)
	case errors.Is(err, llm.ErrGenerationUnavailable), errors.Is(err, llm.ErrRateLimited):
		log.Printf("[scheduler] %s: generation unavailable, skipped (no fallback): %v", label, err)
	case err != nil:
		log.Printf("[scheduler] %s: %v", label, err)
	default:
		log.Printf("[scheduler] %s: ready=%d generated=%d failed=%d skipped=%d",
			label, rep.Ready, len(rep.Generated), len(rep.Failed), len(rep.Skipped))
	}
}
