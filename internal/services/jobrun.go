package services

import (
	"context"
	"errors"
	"sync"

	"github.com/forgen-team/punisher/internal/i18n"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

// ErrJobCancelled is the context cause of a job the learner cancelled ([취소]).
var ErrJobCancelled = errors.New("generation job cancelled")

// ErrJobNotActive: the job already finished (or is saving its result) and can
// no longer be cancelled.
var ErrJobNotActive = errors.New("generation job is not running")

// ReasonCancelled is the learner-facing reason of a cancelled generation.
var ReasonCancelled = i18n.ReasonKo(i18n.ReasonCancelled)

// jobRun is one generation running in this process.
type jobRun struct {
	cancel     context.CancelCauseFunc
	cancelled  bool
	committing bool // saving its result: too late to cancel
}

// jobRuns tracks the cancel func of every generation running in this process
// (the zero value is ready to use).
type jobRuns struct {
	mu   sync.Mutex
	runs map[string]*jobRun
}

// begin registers jobID and returns its cancellable context and the func
// that unregisters it (deferred by the runner).
func (r *jobRuns) begin(parent context.Context, jobID string) (context.Context, func()) {
	ctx, cancel := context.WithCancelCause(parent)
	run := &jobRun{cancel: cancel}
	r.mu.Lock()
	if r.runs == nil {
		r.runs = map[string]*jobRun{}
	}
	r.runs[jobID] = run
	r.mu.Unlock()
	return ctx, func() {
		r.mu.Lock()
		if r.runs[jobID] == run {
			delete(r.runs, jobID)
		}
		r.mu.Unlock()
		cancel(nil)
	}
}

// commit is called right before a job saves its result: false when the job
// was cancelled (nothing may be saved); after true the job can no longer be
// cancelled. A job not registered here (tests, other runners) may commit.
func (r *jobRuns) commit(jobID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[jobID]
	if !ok {
		return true
	}
	if run.cancelled {
		return false
	}
	run.committing = true
	return true
}

// mark flags a registered job cancelled (its context is cancelled later by
// cancelRun, once the row says so). found=false: not running in this process.
func (r *jobRuns) mark(jobID string) (found bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[jobID]
	if !ok {
		return false, nil
	}
	if run.committing {
		return true, ErrJobNotActive
	}
	run.cancelled = true
	return true, nil
}

// cancelRun cancels the context of a marked job: its CLI process group is
// killed (clicore) and a gateway request is aborted (llm).
func (r *jobRuns) cancelRun(jobID string) {
	r.mu.Lock()
	run, ok := r.runs[jobID]
	r.mu.Unlock()
	if ok {
		run.cancel(ErrJobCancelled)
	}
}

// unmark undoes mark when the row could not be cancelled after all.
func (r *jobRuns) unmark(jobID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if run, ok := r.runs[jobID]; ok {
		run.cancelled = false
	}
}

// cancelled reports whether jobID was marked cancelled (its context may not be cancelled yet).
func (r *jobRuns) cancelled(jobID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[jobID]
	return ok && run.cancelled
}

// jobCancelled reports whether the job of ctx was cancelled ([취소]).
func (s *Service) jobCancelled(ctx context.Context, jobID string) bool {
	return errors.Is(context.Cause(ctx), ErrJobCancelled) || s.runs.cancelled(jobID)
}

// cancellableKinds are the job kinds a learner waits on and may cancel.
var cancellableKinds = map[string]bool{"instant": true, "library": true, "check_items": true}

// CancelJob is [취소] of a generation (POST /v1/jobs/{id}/cancel): the row
// becomes failed/cancelled, the running generation's context is cancelled
// (CLI process group killed, gateway request aborted) and nothing it made is
// saved. A finished job is ErrJobNotActive. The target is released at once,
// so the same concept can be started again right away.
func (s *Service) CancelJob(ctx context.Context, userID, jobID string) (models.GenerationJob, error) {
	job, err := s.Store.GetJob(ctx, userID, jobID)
	if err != nil {
		return models.GenerationJob{}, err
	}
	if !cancellableKinds[job.Kind] || (job.Status != "pending" && job.Status != "running") {
		return models.GenerationJob{}, ErrJobNotActive
	}
	running, err := s.runs.mark(jobID)
	if err != nil {
		return models.GenerationJob{}, err
	}
	ok, err := s.Store.CancelJob(ctx, userID, jobID)
	if err != nil || !ok {
		if running {
			s.runs.unmark(jobID)
		}
		if err != nil {
			return models.GenerationJob{}, err
		}
		return models.GenerationJob{}, ErrJobNotActive
	}
	s.runs.cancelRun(jobID)
	if s.Hub != nil {
		s.Hub.Release(jobID)
	}
	return s.Store.GetJob(ctx, userID, jobID)
}

// emitCancelled ends a cancelled job's stream: stage failed with the cancel
// reason, then the legacy error (code cancelled).
func emitCancelled(emit func(Event), attempt, maxAttempts int) {
	emit(stageEvent(StageFailed, attempt, maxAttempts, "", []string{ReasonCancelled}))
	emit(Event{Type: EventError, Data: map[string]string{"code": store.JobReasonCancelled, "message": ReasonCancelled}})
}
