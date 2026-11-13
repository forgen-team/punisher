package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
)

// ErrNoReviews: nothing is due today (or review time is set to 0 minutes).
var ErrNoReviews = errors.New("오늘 복습할 항목이 없어요")

// metricsWindow: the retention metrics cover the last four weeks (design §2.7).
const metricsWindow = 28 * 24 * time.Hour

// ReviewSessionKey is the session key of the day's review session (KST date).
func ReviewSessionKey(now time.Time) string { return "review:" + learning.KSTDay(now) }

func (s *Service) learningSettings(ctx context.Context, userID string) models.LearningSettings {
	st, err := s.Store.GetLearningSettings(ctx, userID)
	if err != nil {
		logf("learning settings for %s: %v (using defaults)", userID, err)
	}
	return st
}

// releaseQueued lets waiting objectives into the queue once the backlog is
// under a day of throughput (store.ReleaseQueuedReviews). Failure only logs.
func (s *Service) releaseQueued(ctx context.Context, userID string, st models.LearningSettings, now time.Time) {
	throughput := learning.ReviewThroughput(st.ReviewMinutes, st.ReviewGrader)
	if _, err := s.Store.ReleaseQueuedReviews(ctx, userID, throughput, now, st.ReviewIntervals); err != nil {
		logf("release queued reviews for %s: %v", userID, err)
	}
}

// reviewHold: a new check's objectives should wait (backlog over three days of throughput).
func (s *Service) reviewHold(ctx context.Context, userID string) bool {
	st := s.learningSettings(ctx, userID)
	counts, err := s.Store.ReviewCounts(ctx, userID, s.now())
	if err != nil {
		logf("review backlog for %s: %v (no hold)", userID, err)
		return false
	}
	return learning.HoldNewReviews(counts.Due, learning.ReviewThroughput(st.ReviewMinutes, st.ReviewGrader))
}

// reviewGrader is the grader a new review session is fixed to: the setting,
// except that the tutor needs LLM access (non-owners grade themselves).
func (s *Service) reviewGrader(userID string, st models.LearningSettings) string {
	if st.ReviewGrader == learning.GraderLLM && s.AllowUser(userID) {
		return learning.GraderLLM
	}
	return learning.GraderSelf
}

// StartReview is POST /v1/reviews: the open review session of today, or a new
// one. Open review sessions of earlier days are abandoned first (applied
// evidence stays). A new session fixes its grader and its items: the top of
// the ordered queue, as many as fit in review_minutes (design §5 P2).
func (s *Service) StartReview(ctx context.Context, userID string) (models.CheckDetail, bool, error) {
	now := s.now()
	key := ReviewSessionKey(now)
	if _, err := s.Store.AbandonStaleReviews(ctx, userID, key); err != nil {
		return models.CheckDetail{}, false, err
	}
	st := s.learningSettings(ctx, userID)
	s.releaseQueued(ctx, userID, st, now)
	if open, err := s.Store.LatestCheckSession(ctx, userID, key); err == nil && (open.Status == "active" || open.Status == "preparing") {
		d, err := s.GetCheck(ctx, userID, open.ID)
		return d, false, err
	} else if err != nil && !errors.Is(err, store.ErrNotFound) {
		return models.CheckDetail{}, false, err
	}
	grader := s.reviewGrader(userID, st)
	limit := learning.ReviewThroughput(st.ReviewMinutes, grader)
	if limit <= 0 {
		return models.CheckDetail{}, false, fmt.Errorf("%w: 복습 시간이 0분이에요", ErrNoReviews)
	}
	picks, err := s.pickReviewItems(ctx, userID, now, st.ReviewIntervals, limit)
	if err != nil {
		return models.CheckDetail{}, false, err
	}
	if len(picks) == 0 {
		return models.CheckDetail{}, false, ErrNoReviews
	}
	sess, created, err := s.Store.CreateCheckSession(ctx, store.NewCheckSession{UserID: userID, Kind: check.PurposeReview,
		Grader: grader, SessionKey: key})
	if err != nil {
		return models.CheckDetail{}, false, err
	}
	if created {
		ids := make([]string, 0, len(picks))
		for _, p := range picks {
			ids = append(ids, p.Item.ID)
		}
		first := picks[0]
		ask := store.TurnWrite{ItemID: first.Item.ID, Role: "tutor", Move: check.MoveAsk, Text: first.Item.Prompt, ReusedItem: first.Reused}
		if _, err := s.Store.ActivateCheckSession(ctx, userID, sess.ID, ids, ask); err != nil {
			if ferr := s.Store.SetCheckSessionStatus(ctx, userID, sess.ID, []string{"preparing"}, "failed", err.Error()); ferr != nil {
				logf("review %s: fail session: %v", sess.ID, ferr)
			}
			return models.CheckDetail{}, true, err
		}
	}
	d, err := s.GetCheck(ctx, userID, sess.ID)
	return d, created, err
}

// pickReviewItems orders the due queue (learning.OrderReviews) and takes one
// item per objective, skipping objectives without any visible item.
func (s *Service) pickReviewItems(ctx context.Context, userID string, now time.Time, intervals []int, limit int) ([]store.ReviewPick, error) {
	queue, err := s.Store.ReviewQueue(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	ordered := learning.OrderReviews(queue, now, intervals)
	ids := make([]string, 0, len(ordered))
	for _, c := range ordered {
		ids = append(ids, c.ObjectiveID)
	}
	byObjective, err := s.Store.ReviewItems(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := []store.ReviewPick{}
	for _, c := range ordered {
		if p, ok := byObjective[c.ObjectiveID]; ok && len(out) < limit {
			out = append(out, p)
		}
	}
	return out, nil
}

// reusedItem: the item's ask in this session was marked as a reused item.
func (st checkState) reusedItem(itemID string) bool {
	for _, t := range st.turns {
		if t.ItemID == itemID && t.Move == check.MoveAsk && t.ReusedItem {
			return true
		}
	}
	return false
}

// ReviewSummary is GET /v1/reviews/summary: due, overdue and waiting counts,
// the estimate for today and today's session (waiting objectives are released first).
func (s *Service) ReviewSummary(ctx context.Context, userID string) (models.ReviewSummary, error) {
	now := s.now()
	st := s.learningSettings(ctx, userID)
	s.releaseQueued(ctx, userID, st, now)
	counts, err := s.Store.ReviewCounts(ctx, userID, now)
	if err != nil {
		return models.ReviewSummary{}, err
	}
	grader := s.reviewGrader(userID, st)
	sum := learning.SummarizeReviews(counts, st.ReviewMinutes, grader)
	out := models.ReviewSummary{Due: sum.Due, Overdue: sum.Overdue, Queued: sum.Queued, EstimatedMinutes: sum.EstimatedMinutes,
		Backlogged: sum.Backlogged, Throughput: learning.ReviewThroughput(st.ReviewMinutes, grader), Grader: grader,
		ReviewMinutes: st.ReviewMinutes}
	sess, err := s.Store.LatestCheckSession(ctx, userID, ReviewSessionKey(now))
	switch {
	case errors.Is(err, store.ErrNotFound):
	case err != nil:
		return out, err
	default:
		out.Today = &models.ReviewSessionRef{ID: sess.ID, Status: sess.Status, Grader: sess.Grader,
			Total: len(sess.ItemIDs), Done: resultCount(sess.Result)}
	}
	return out, nil
}

func resultCount(raw json.RawMessage) int {
	var r struct {
		Items []json.RawMessage `json:"items"`
	}
	if len(raw) == 0 {
		return 0
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		logf("decode session result: %v", err)
		return 0
	}
	return len(r.Items)
}

// MasteryMetrics is the payload of GET /v1/mastery/metrics (design §2.7).
type MasteryMetrics struct {
	Since time.Time `json:"since"`
	Weeks int       `json:"weeks"`
	// Buckets: first-attempt accuracy of reviews on unseen items, per interval and grader.
	Buckets []learning.RetentionBucket `json:"buckets"`
}

// Metrics computes the "내 기억 유지 지표" of the last four weeks.
func (s *Service) Metrics(ctx context.Context, userID string) (MasteryMetrics, error) {
	since := s.now().Add(-metricsWindow)
	samples, err := s.Store.RetentionSamples(ctx, userID, since)
	if err != nil {
		return MasteryMetrics{}, err
	}
	st := s.learningSettings(ctx, userID)
	return MasteryMetrics{Since: since, Weeks: 4, Buckets: learning.RetentionMetrics(samples, st.ReviewIntervals)}, nil
}
