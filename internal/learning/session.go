package learning

import "math"

// Session time of the today screen (quality design v2.2 §5 / Q3): "강의 약 n분
// + 확인 약 7분 + 복습 약 k분".
const (
	// DefaultCheckItems: items of a typical '확인하기' (objectives + the integrative item).
	DefaultCheckItems = 4
	// MinutesPerCheckItem: average time of one check item [추정 1.7분].
	MinutesPerCheckItem = 1.7
)

// SessionMinutes is today's session split (TodayResult.Session).
type SessionMinutes struct {
	// LectureMinutes: the first recommendation (resume = the minutes left); 0 when nothing to learn.
	LectureMinutes int `json:"lectureMinutes"`
	// CheckMinutes: the check after that lecture (DefaultCheckItems × MinutesPerCheckItem); 0 without a lecture.
	CheckMinutes int `json:"checkMinutes"`
	// ReviewMinutes: today's reviews (ReviewToday.EstimatedMinutes).
	ReviewMinutes int `json:"reviewMinutes"`
}

// SessionFor is the session of today's first recommendation and reviews.
func SessionFor(items []Recommendation, review ReviewToday) SessionMinutes {
	s := SessionMinutes{ReviewMinutes: review.EstimatedMinutes}
	if len(items) > 0 {
		s.LectureMinutes = items[0].EstimatedMinutes
		s.CheckMinutes = int(math.Round(DefaultCheckItems * MinutesPerCheckItem))
	}
	return s
}
