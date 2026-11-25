package learning

import (
	"math"
	"time"
)

// Repeated failure (leech, W5a): an objective failed this many times in total
// is shown with "풀어 쓴 강의 / 앞 개념부터 / 잠시 보류". Anki's default is 8
// failures [확인: https://docs.ankiweb.net/manual/leeches]; reviews here are
// sparser (one item per objective per review), so the bar is lower [추정].
const LeechThreshold = 4

// CountsAsFailure: outcomes that count toward the leech threshold (the same
// ones that send the ladder back to box 0).
func CountsAsFailure(outcome string) bool {
	return outcome == OutcomeFailed || outcome == OutcomeAided
}

// BecomesLeech: this failure made the objective a leech (it was not one).
func BecomesLeech(failCount int, wasLeech bool) bool {
	return !wasLeech && failCount >= LeechThreshold
}

// Rest days (user_study_prefs.rest_days): weekdays 0=Sunday … 6=Saturday on
// which the daily goal is not shown as pending and no reminder is sent.
// Nothing is lost on a rest day — reviews simply wait (no streak to break).

// ValidRestDays: distinct weekdays 0~6, at most 6 (one study day stays).
func ValidRestDays(days []int) bool {
	if len(days) > 6 {
		return false
	}
	seen := map[int]bool{}
	for _, d := range days {
		if d < 0 || d > 6 || seen[d] {
			return false
		}
		seen[d] = true
	}
	return true
}

// IsRestDay reports whether the KST day of t is a rest day.
func IsRestDay(t time.Time, restDays []int) bool {
	wd := int(KSTDayStart(t).Weekday())
	for _, d := range restDays {
		if d == wd {
			return true
		}
	}
	return false
}

// Deadline plan (goal deadlines, W5a): how much daily time the remaining
// concepts need before the due date, counting only study days.

// PlanConceptMinutes is the planned time per remaining concept: a lecture
// (about 15 minutes) plus its check (about 5) [추정 — 강의 분량 평균으로 보정 가능].
const PlanConceptMinutes = 20

// Plan statuses.
const (
	PlanNoDeadline = "no_deadline"
	PlanDone       = "done"
	PlanOnTrack    = "on_track"
	PlanTight      = "tight"
	PlanOverdue    = "overdue"
)

// DeadlinePlan is the plan of one goal.
type DeadlinePlan struct {
	DueOn             string `json:"dueOn"`
	RemainingConcepts int    `json:"remainingConcepts"`
	RemainingMinutes  int    `json:"remainingMinutes"`
	// StudyDays: study days from today through the due date (rest days excluded).
	StudyDays     int `json:"studyDays"`
	MinutesPerDay int `json:"minutesPerDay"`
	DailyMinutes  int `json:"dailyMinutes"`
	// FinishOn: when the goal ends at the daily pace (YYYY-MM-DD), "" when done.
	FinishOn string `json:"finishOn"`
	Status   string `json:"status"`
}

// PlanDeadline computes a goal's plan. dueOn is YYYY-MM-DD (KST) or "".
func PlanDeadline(remaining int, dueOn string, now time.Time, dailyMinutes int, restDays []int) DeadlinePlan {
	p := DeadlinePlan{DueOn: dueOn, RemainingConcepts: remaining, RemainingMinutes: remaining * PlanConceptMinutes, DailyMinutes: dailyMinutes}
	if dailyMinutes <= 0 {
		dailyMinutes = DefaultDailyMinutes
		p.DailyMinutes = dailyMinutes
	}
	if remaining <= 0 {
		p.Status = PlanDone
		return p
	}
	p.FinishOn = finishOn(p.RemainingMinutes, dailyMinutes, now, restDays)
	if dueOn == "" {
		p.Status = PlanNoDeadline
		return p
	}
	due, err := time.ParseInLocation("2006-01-02", dueOn, kst)
	if err != nil {
		p.Status = PlanNoDeadline
		return p
	}
	today := KSTDayStart(now)
	for d := today; !d.After(due); d = d.AddDate(0, 0, 1) {
		if !IsRestDay(d, restDays) {
			p.StudyDays++
		}
	}
	if p.StudyDays == 0 {
		p.Status = PlanOverdue
		return p
	}
	p.MinutesPerDay = int(math.Ceil(float64(p.RemainingMinutes) / float64(p.StudyDays)))
	if p.MinutesPerDay <= dailyMinutes {
		p.Status = PlanOnTrack
	} else {
		p.Status = PlanTight
	}
	return p
}

func finishOn(minutes, daily int, now time.Time, restDays []int) string {
	left := minutes
	d := KSTDayStart(now)
	if len(restDays) >= 7 {
		return ""
	}
	for left > 0 {
		if !IsRestDay(d, restDays) {
			left -= daily
		}
		if left > 0 {
			d = d.AddDate(0, 0, 1)
		}
	}
	return KSTDay(d)
}

// ValidDueOn: a YYYY-MM-DD date (KST) not before today and within five years.
func ValidDueOn(dueOn string, now time.Time) bool {
	due, err := time.ParseInLocation("2006-01-02", dueOn, kst)
	if err != nil {
		return false
	}
	today := KSTDayStart(now)
	return !due.Before(today) && due.Before(today.AddDate(5, 0, 0))
}
