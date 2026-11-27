package services

import (
	"context"
	"errors"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

// preparingJob is the running item-set generation of a preparing check
// session (ok=false for other sessions or when none runs).
func (s *Service) preparingJob(ctx context.Context, userID string, sess models.CheckSession) (models.GenerationJob, bool) {
	if sess.Status != "preparing" {
		return models.GenerationJob{}, false
	}
	if sess.Kind != check.PurposeCheck { // diagnostic, synthesis: the session's own generation
		job, err := s.Store.ActiveSessionItemsJob(ctx, userID, sess.ID)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			logf("check %s: items job: %v", sess.ID, err)
		}
		return job, err == nil
	}
	if sess.LectureID == "" {
		return models.GenerationJob{}, false
	}
	cl, err := s.loadCheckLecture(ctx, userID, sess.LectureID)
	if err != nil {
		logf("check %s: lecture of the preparing session: %v", sess.ID, err)
		return models.GenerationJob{}, false
	}
	job, err := s.Store.ActiveCheckItemsJob(ctx, userID, s.checkItemKey(ctx, userID, cl.lec))
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			logf("check %s: items job: %v", sess.ID, err)
		}
		return models.GenerationJob{}, false
	}
	return job, true
}

// CancelCheck is [취소] while a session's items are being made (check,
// diagnostic, synthesis): the item-set
// generation is cancelled (CancelJob — the AI CLI process group is killed,
// nothing is saved) and the session ends abandoned. A session that is not
// preparing any more (items arrived) is ErrCheckState (409).
func (s *Service) CancelCheck(ctx context.Context, userID, id string) (models.CheckDetail, error) {
	st, err := s.loadCheckState(ctx, userID, id)
	if err != nil {
		return models.CheckDetail{}, err
	}
	if st.sess.Status != "preparing" {
		return models.CheckDetail{}, ErrCheckState
	}
	if job, ok := s.preparingJob(ctx, userID, st.sess); ok {
		if _, err := s.CancelJob(ctx, userID, job.ID); err != nil && !errors.Is(err, ErrJobNotActive) {
			return models.CheckDetail{}, err
		}
	}
	// The cancelled runner abandons the session it started; this one (it may
	// be waiting on another session's generation) is abandoned here at once.
	err = s.Store.SetCheckSessionStatus(ctx, userID, id, []string{"preparing"}, "abandoned", "")
	if errors.Is(err, store.ErrCheckClosed) {
		if st, lerr := s.loadCheckState(ctx, userID, id); lerr == nil && st.sess.Status == "abandoned" {
			return s.GetCheck(ctx, userID, id)
		}
		return models.CheckDetail{}, ErrCheckState
	}
	if err != nil {
		return models.CheckDetail{}, checkBusy(err)
	}
	return s.GetCheck(ctx, userID, id)
}

// startSessionJob records a session's own item generation (diagnostic,
// synthesis) as a running check_items job so [취소] can stop it. "" when it
// could not be recorded: the generation still runs (tokens via recordUsage).
func (s *Service) startSessionJob(userID, conceptID, sessionID string) string {
	job, err := s.Store.StartSessionItemsJob(context.Background(), userID, conceptID, sessionID)
	if err != nil {
		logf("session %s: record items job: %v", sessionID, err)
		return ""
	}
	return job.ID
}

// sessionItemsRun registers jobID's generation on ctx (a no-op without a job).
func (s *Service) sessionItemsRun(ctx context.Context, jobID string) (context.Context, func()) {
	if jobID == "" {
		return ctx, func() {}
	}
	return s.runs.begin(ctx, jobID)
}

// itemsCancelled is err joined with ErrJobCancelled when jobID was cancelled
// (or is cancelled before its items are saved — then they are not saved).
func (s *Service) itemsCancelled(ctx context.Context, jobID string, err error) error {
	if jobID != "" && (s.jobCancelled(ctx, jobID) || (err == nil && !s.runs.commit(jobID))) {
		return errors.Join(ErrJobCancelled, err)
	}
	return err
}

// endSessionItems records a session generation's tokens on its job (or as a
// usage row without one) and, on failure, ends the preparing session: failed
// with text, or abandoned when it was cancelled.
func (s *Service) endSessionItems(ctx context.Context, label, userID, conceptID, sessionID, jobID string, u usage, tries []store.AttemptRecord, err error, text string) {
	if jobID == "" {
		s.recordUsage(ctx, userID, "check_items", conceptID, u, err)
	} else {
		s.finishItemsJob(ctx, label+" "+sessionID, jobID, u, tries, err)
	}
	if err == nil {
		return
	}
	status := "failed"
	if errors.Is(err, ErrJobCancelled) {
		status, text = "abandoned", ""
	}
	if serr := s.Store.SetCheckSessionStatus(ctx, userID, sessionID, []string{"preparing"}, status, text); serr != nil && !errors.Is(serr, store.ErrCheckClosed) {
		logf("%s %s: end session: %v", label, sessionID, serr)
	}
}
