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
	if sess.Status != "preparing" || sess.Kind != check.PurposeCheck || sess.LectureID == "" {
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

// CancelCheck is [취소] while a check's items are being made: the item-set
// generation is cancelled (CancelJob — the AI CLI process group is killed,
// nothing is saved) and the session ends abandoned. A session that is not
// preparing any more (items arrived) is ErrCheckState (409).
func (s *Service) CancelCheck(ctx context.Context, userID, id string) (models.CheckDetail, error) {
	st, err := s.loadCheckState(ctx, userID, id)
	if err != nil {
		return models.CheckDetail{}, err
	}
	if st.sess.Status != "preparing" || st.sess.Kind != check.PurposeCheck {
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
