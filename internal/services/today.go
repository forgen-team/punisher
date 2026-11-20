package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

// todayInput converts store data into learning.TodayInput.
func todayInput(d *store.TodayData, limit int) learning.TodayInput {
	depthOf := make(map[string]int, len(d.Concepts))
	for id, c := range d.Concepts {
		depthOf[id] = c.Depth
	}
	in := learning.TodayInput{Concepts: d.Concepts, Edges: d.Edges, Resume: d.Resume, Requests: d.Requests, Limit: limit, Lapsed: d.Lapsed,
		Unchecked: d.Unchecked}
	for _, g := range d.Goals {
		for _, target := range g.Targets {
			in.Goals = append(in.Goals, learning.GoalInput{ID: g.ID, Title: g.Title, Path: learning.PathTo(target, d.Edges, depthOf)})
		}
	}
	return in
}

// Today builds the user's recommendations.
func (s *Service) Today(ctx context.Context, userID string, limit int) (learning.TodayResult, error) {
	st, serr := s.Store.GetLearningSettings(ctx, userID)
	if serr != nil {
		logf("learning settings for %s: %v (using defaults)", userID, serr)
	}
	s.releaseQueued(ctx, userID, st, s.now())
	data, err := s.Store.LoadToday(ctx, userID, s.now())
	if err != nil {
		return learning.TodayResult{}, err
	}
	in := todayInput(data, limit)
	in.Locale = st.Locale
	in.ReaderLevel = st.ReaderLevel // 0단계 기초 추천 여부·순서(learner-level ②)
	in.Review = learning.ReviewInput{Counts: data.Review, ReviewMinutes: st.ReviewMinutes, Grader: s.reviewGrader(userID, st)}
	res := learning.BuildToday(in)
	// 이미 만든 강의가 있는 추천은 그 강의의 계산 시간을 보인다(없으면 기본값) — 세션 합계와 같은 기준.
	if in.LectureMinutes = s.recommendedLectureMinutes(ctx, userID, res.Items); len(in.LectureMinutes) > 0 {
		res = learning.BuildToday(in)
	}
	res.NextChapter = learning.CheckNextChapter(learning.RemainingMinutes(data.Concepts), st.DailyMinutes)
	return res, nil
}

// recommendedLectureMinutes: for each non-resume recommendation whose next lecture
// (the one '강의 시작' would open) already exists, that lecture's time by concept.
// Lookup failures keep the default and are logged — the today screen must still load.
func (s *Service) recommendedLectureMinutes(ctx context.Context, userID string, items []learning.Recommendation) map[string]int {
	out := map[string]int{}
	for _, item := range items {
		if item.Kind == learning.RecResume {
			continue
		}
		_, plan, err := s.planLecture(ctx, userID, item.ConceptID, 0)
		if err != nil {
			if !errors.Is(err, learning.ErrNothingToLearn) && !errors.Is(err, store.ErrNotFound) {
				logf("today minutes plan %s: %v", item.ConceptID, err)
			}
			continue
		}
		lec, err := s.Store.PublishedLectureCovering(ctx, userID, item.ConceptID, plan.Depth, plan.ObjectiveIDs, plan.Language, plan.ReaderLevel, store.LangPrefer)
		if err != nil {
			if !errors.Is(err, store.ErrNotFound) {
				logf("today minutes lecture %s: %v", item.ConceptID, err)
			}
			continue
		}
		if m := lectureMinutes(lec); m > 0 {
			out[item.ConceptID] = m
		}
	}
	return out
}

// lectureMinutes is the time the lecture page shows (learning.EstimateMinutes of
// the content — the section sum), or the stored value when the content does not decode.
func lectureMinutes(l models.Lecture) int {
	var c lecture.Content
	if len(l.Content) == 0 || json.Unmarshal(l.Content, &c) != nil || len(c.Sections) == 0 {
		return l.EstimatedMinutes
	}
	return learning.EstimateMinutes(c)
}
