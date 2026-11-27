package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
)

// W5a study features: study prefs, FSRS memory and repeated failure (leech) per
// objective, review forecast, daily activity and goal deadlines (migration 023).

// StudyPrefs is user_study_prefs (defaults when the user saved none).
type StudyPrefs struct {
	MemoryModel     string  `json:"memoryModel"`
	TargetRetention float64 `json:"targetRetention"`
	RestDays        []int   `json:"restDays"`
	// ReminderHour: KST hour (0~23) of the daily review reminder, -1 = off.
	ReminderHour    int    `json:"reminderHour"`
	LastRemindedDay string `json:"-"`
}

// DefaultStudyPrefs: FSRS blended with the ladder, 90% retention, no rest days, no reminder.
func DefaultStudyPrefs() StudyPrefs {
	return StudyPrefs{MemoryModel: learning.MemoryModelFSRS, TargetRetention: learning.DefaultTargetRetention, RestDays: []int{}, ReminderHour: -1}
}

// StudyPrefsPatch changes only the non-nil fields (already validated).
type StudyPrefsPatch struct {
	MemoryModel     *string
	TargetRetention *float64
	RestDays        []int // nil = keep
	ReminderHour    *int
}

func studyPrefs(ctx context.Context, c conn, userID string) (StudyPrefs, error) {
	p := DefaultStudyPrefs()
	var rest string
	err := c.row(ctx, `SELECT memory_model, target_retention, rest_days, reminder_hour, last_reminded_day
		FROM user_study_prefs WHERE user_id = $1`, userID).Scan(&p.MemoryModel, &p.TargetRetention, &rest, &p.ReminderHour, &p.LastRemindedDay)
	if errors.Is(err, sql.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return DefaultStudyPrefs(), fmt.Errorf("load study prefs: %w", err)
	}
	if !learning.ValidMemoryModel(p.MemoryModel) {
		p.MemoryModel = learning.MemoryModelFSRS
	}
	if !learning.ValidTargetRetention(p.TargetRetention) {
		p.TargetRetention = learning.DefaultTargetRetention
	}
	if jerr := json.Unmarshal([]byte(rest), &p.RestDays); jerr != nil || !learning.ValidRestDays(p.RestDays) {
		p.RestDays = []int{}
	}
	return p, nil
}

// GetStudyPrefs returns the user's study prefs.
func (s *SQLiteStore) GetStudyPrefs(ctx context.Context, userID string) (StudyPrefs, error) {
	return studyPrefs(ctx, s.c(), userID)
}

// UpdateStudyPrefs upserts the given fields.
func (s *SQLiteStore) UpdateStudyPrefs(ctx context.Context, userID string, p StudyPrefsPatch) (StudyPrefs, error) {
	rest, err := jsonParam(p.RestDays != nil, p.RestDays)
	if err != nil {
		return StudyPrefs{}, err
	}
	def := DefaultStudyPrefs()
	if _, err := s.c().exec(ctx, `
		INSERT INTO user_study_prefs (user_id, memory_model, target_retention, rest_days, reminder_hour, updated_at)
		VALUES ($1, COALESCE($2, $6), COALESCE($3, $7), COALESCE($4, '[]'), COALESCE($5, $8), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			memory_model = COALESCE($2, user_study_prefs.memory_model),
			target_retention = COALESCE($3, user_study_prefs.target_retention),
			rest_days = COALESCE($4, user_study_prefs.rest_days),
			reminder_hour = COALESCE($5, user_study_prefs.reminder_hour),
			updated_at = NOW()`,
		userID, p.MemoryModel, p.TargetRetention, rest, p.ReminderHour, def.MemoryModel, def.TargetRetention, def.ReminderHour); err != nil {
		return StudyPrefs{}, fmt.Errorf("save study prefs: %w", err)
	}
	return s.GetStudyPrefs(ctx, userID)
}

// MarkReminded records that today's reminder was sent (day = KST YYYY-MM-DD).
func (s *SQLiteStore) MarkReminded(ctx context.Context, userID, day string) error {
	_, err := s.c().exec(ctx, `UPDATE user_study_prefs SET last_reminded_day = $2 WHERE user_id = $1`, userID, day)
	return err
}

// objectiveMemory is a row of user_objective_memory.
type objectiveMemory struct {
	M         learning.Memory
	FailCount int
	Leech     bool
}

func loadObjectiveMemory(ctx context.Context, c conn, userID, objectiveID string) (objectiveMemory, error) {
	var om objectiveMemory
	var leech *time.Time
	err := c.row(ctx, `SELECT stability, difficulty, reps, last_review_at, fail_count, leech_at FROM user_objective_memory
		WHERE user_id = $1 AND objective_id = $2`, userID, objectiveID).
		Scan(&om.M.Stability, &om.M.Difficulty, &om.M.Reps, nts(&om.M.LastReviewAt), &om.FailCount, nts(&leech))
	if errors.Is(err, sql.ErrNoRows) {
		return om, nil
	}
	om.Leech = leech != nil
	return om, err
}

// applyMemory runs after an item's ladder result (next) is decided and before it
// is saved: the FSRS state and the failure count move, a new leech is marked,
// and with memory_model fsrs a ladder review date is re-timed (blended).
// Graduation steps, waiting (queued) objectives and held answers keep their date.
func applyMemory(ctx context.Context, c conn, userID, objectiveID, outcome string, next learning.Mastery, now time.Time) (learning.Mastery, error) {
	rating := learning.RatingFor(outcome)
	if rating == 0 {
		return next, nil
	}
	prefs, err := studyPrefs(ctx, c, userID)
	if err != nil {
		return next, err
	}
	om, err := loadObjectiveMemory(ctx, c, userID, objectiveID)
	if err != nil {
		return next, fmt.Errorf("load memory: %w", err)
	}
	mem := learning.NextMemory(om.M, rating, now)
	fails := om.FailCount
	if learning.CountsAsFailure(outcome) {
		fails++
	}
	var leechAt any
	if learning.BecomesLeech(fails, om.Leech) {
		leechAt = now
	}
	if _, err := c.exec(ctx, `
		INSERT INTO user_objective_memory (user_id, objective_id, stability, difficulty, reps, last_review_at, fail_count, leech_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, objective_id) DO UPDATE SET
			stability = excluded.stability, difficulty = excluded.difficulty, reps = excluded.reps,
			last_review_at = excluded.last_review_at, fail_count = excluded.fail_count,
			leech_at = COALESCE(user_objective_memory.leech_at, excluded.leech_at), updated_at = excluded.updated_at`,
		userID, objectiveID, mem.Stability, mem.Difficulty, mem.Reps, now, fails, leechAt, now); err != nil {
		return next, fmt.Errorf("save memory: %w", err)
	}
	if prefs.MemoryModel == learning.MemoryModelFSRS && next.NextReviewAt != nil && next.GradStep == learning.GradLadder && !next.Queued {
		due := learning.RetimeReview(*next.NextReviewAt, now, mem, om.M.Reps, prefs.TargetRetention)
		next.NextReviewAt = &due
	}
	return next, nil
}

// suspendedSQL: the objective is on hold ('잠시 보류') — kept out of reviews.
func suspendedSQL(objective, user string) string {
	return `EXISTS (SELECT 1 FROM user_objective_memory sm WHERE sm.user_id = ` + user + ` AND sm.objective_id = ` + objective +
		` AND sm.suspended_at IS NOT NULL)`
}

// Leech is one objective that keeps failing, or one on hold.
type Leech struct {
	ObjectiveID   string     `json:"objectiveId"`
	ObjectiveText string     `json:"objectiveText"`
	ConceptID     string     `json:"conceptId"`
	ConceptName   string     `json:"conceptName"`
	FailCount     int        `json:"failCount"`
	LeechAt       *time.Time `json:"leechAt"`
	SuspendedAt   *time.Time `json:"suspendedAt"`
}

// Leeches lists the user's repeated-failure objectives (not on hold) and the
// objectives on hold, newest first, in the user's language.
func (s *SQLiteStore) Leeches(ctx context.Context, userID, lang string) (leeches, suspended []Leech, err error) {
	list, err := collect(s.c().query(ctx, `
		SELECT mm.objective_id, `+locObjective("o", "$2")+`, o.concept_id, `+locName("c", "$2")+`,
			mm.fail_count, mm.leech_at, mm.suspended_at
		FROM user_objective_memory mm
		JOIN objectives o ON o.id = mm.objective_id AND o.status = 'active'
		JOIN concepts c ON c.id = o.concept_id AND c.status <> 'retired'
		WHERE mm.user_id = $1 AND (mm.leech_at IS NOT NULL OR mm.suspended_at IS NOT NULL)
		ORDER BY COALESCE(mm.suspended_at, mm.leech_at) DESC`, userID, lang),
		func(row rowScanner) (Leech, error) {
			var l Leech
			err := row.Scan(&l.ObjectiveID, &l.ObjectiveText, &l.ConceptID, &l.ConceptName, &l.FailCount, nts(&l.LeechAt), nts(&l.SuspendedAt))
			return l, err
		})
	if err != nil {
		return nil, nil, fmt.Errorf("leeches: %w", err)
	}
	leeches, suspended = []Leech{}, []Leech{}
	for _, l := range list {
		if l.SuspendedAt != nil {
			suspended = append(suspended, l)
		} else {
			leeches = append(leeches, l)
		}
	}
	return leeches, suspended, nil
}

// Leech actions.
const (
	LeechKeep    = "keep"    // 계속 복습: 표시와 실패 수를 지운다
	LeechSuspend = "suspend" // 잠시 보류: 복습에서 뺀다
	LeechResume  = "resume"  // 보류 풀기(되돌리기): 다시 복습에 넣는다
)

// SetLeech applies a leech action. ErrNotFound when the objective has no memory row.
func (s *SQLiteStore) SetLeech(ctx context.Context, userID, objectiveID, action string) error {
	var q string
	switch action {
	case LeechKeep:
		q = `UPDATE user_objective_memory SET leech_at = NULL, fail_count = 0, updated_at = NOW() WHERE user_id = $1 AND objective_id = $2`
	case LeechSuspend:
		q = `UPDATE user_objective_memory SET suspended_at = COALESCE(suspended_at, NOW()), updated_at = NOW() WHERE user_id = $1 AND objective_id = $2`
	case LeechResume:
		q = `UPDATE user_objective_memory SET suspended_at = NULL, updated_at = NOW() WHERE user_id = $1 AND objective_id = $2`
	default:
		return fmt.Errorf("unknown leech action %q", action)
	}
	res, err := s.c().exec(ctx, q, userID, objectiveID)
	if err != nil {
		return fmt.Errorf("leech %s: %w", action, err)
	}
	if affected(res) == 0 {
		return ErrNotFound
	}
	return nil
}

// ForecastDay is the reviews due on one KST day.
type ForecastDay struct {
	Day        string `json:"day"`
	Objectives int    `json:"objectives"`
	Cards      int    `json:"cards"`
}

// ReviewForecast counts the reviews falling due on each of the next days KST
// days (today first; anything already overdue is counted today). Waiting
// (queued) objectives have no date yet and are returned apart.
func (s *SQLiteStore) ReviewForecast(ctx context.Context, userID string, now time.Time, days int) ([]ForecastDay, int, error) {
	start := learning.KSTDayStart(now)
	end := start.AddDate(0, 0, days)
	out := make([]ForecastDay, days)
	index := map[string]int{}
	for i := range out {
		out[i].Day = learning.KSTDay(start.AddDate(0, 0, i))
		index[out[i].Day] = i
	}
	add := func(q string, card bool) error {
		rows, err := collect(s.c().query(ctx, q, userID, end), func(row rowScanner) (time.Time, error) {
			var t time.Time
			err := row.Scan(ts(&t))
			return t, err
		})
		if err != nil {
			return err
		}
		for _, t := range rows {
			i := 0
			if t.After(start) {
				i = index[learning.KSTDay(t)]
			}
			if card {
				out[i].Cards++
			} else {
				out[i].Objectives++
			}
		}
		return nil
	}
	if err := add(`SELECT m.next_review_at FROM user_objective_mastery m`+reviewScope+`
		WHERE m.user_id = $1 AND m.next_review_at IS NOT NULL AND m.next_review_at < $2 AND NOT m.queued AND m.grad_step < 2`, false); err != nil {
		return nil, 0, fmt.Errorf("forecast objectives: %w", err)
	}
	if err := add(`SELECT next_review_at FROM review_cards
		WHERE user_id = $1 AND next_review_at IS NOT NULL AND next_review_at < $2 AND suspended_at IS NULL`, true); err != nil {
		return nil, 0, fmt.Errorf("forecast cards: %w", err)
	}
	var queued int
	if err := s.c().row(ctx, `SELECT COUNT(*) FROM user_objective_mastery m`+reviewScope+`
		WHERE m.user_id = $1 AND m.queued AND m.grad_step < 2`, userID).Scan(&queued); err != nil {
		return nil, 0, fmt.Errorf("forecast queued: %w", err)
	}
	return out, queued, nil
}

// ActivityDay is what was done on one KST day.
type ActivityDay struct {
	Day      string `json:"day"`
	Reviews  int    `json:"reviews"`
	Lectures int    `json:"lectures"`
	Cards    int    `json:"cards"`
}

// Activity counts finished check/review items, completed lectures and card
// reviews per KST day over the last days days (oldest first, today last).
func (s *SQLiteStore) Activity(ctx context.Context, userID string, now time.Time, days int) ([]ActivityDay, error) {
	start := learning.KSTDayStart(now).AddDate(0, 0, -(days - 1))
	out := make([]ActivityDay, days)
	index := map[string]int{}
	for i := range out {
		out[i].Day = learning.KSTDay(start.AddDate(0, 0, i))
		index[out[i].Day] = i
	}
	count := func(q string, set func(*ActivityDay, int)) error {
		type row struct {
			day string
			n   int
		}
		rows, err := collect(s.c().query(ctx, q, userID, start), func(r rowScanner) (row, error) {
			var x row
			err := r.Scan(&x.day, &x.n)
			return x, err
		})
		if err != nil {
			return err
		}
		for _, x := range rows {
			if i, ok := index[x.day]; ok {
				set(&out[i], x.n)
			}
		}
		return nil
	}
	const day = `strftime('%Y-%m-%d', %s, '+9 hours')`
	if err := count(`SELECT `+strings.Replace(day, "%s", "created_at", 1)+` AS d, COUNT(*) FROM check_turns
		WHERE user_id = $1 AND outcome IS NOT NULL AND outcome NOT IN ('held', 'recorded', 'skipped') AND created_at >= $2 GROUP BY d`,
		func(a *ActivityDay, n int) { a.Reviews = n }); err != nil {
		return nil, fmt.Errorf("activity reviews: %w", err)
	}
	if err := count(`SELECT `+strings.Replace(day, "%s", "completed_at", 1)+` AS d, COUNT(*) FROM user_lecture_progress
		WHERE user_id = $1 AND completed_at IS NOT NULL AND completed_at >= $2 GROUP BY d`,
		func(a *ActivityDay, n int) { a.Lectures = n }); err != nil {
		return nil, fmt.Errorf("activity lectures: %w", err)
	}
	if err := count(`SELECT `+strings.Replace(day, "%s", "reviewed_at", 1)+` AS d, COUNT(*) FROM review_card_log
		WHERE user_id = $1 AND reviewed_at >= $2 GROUP BY d`,
		func(a *ActivityDay, n int) { a.Cards = n }); err != nil {
		return nil, fmt.Errorf("activity cards: %w", err)
	}
	return out, nil
}

// GoalDeadline returns a goal's due date ("" when none).
func (s *SQLiteStore) GoalDeadline(ctx context.Context, userID, goalID string) (string, error) {
	var due string
	err := s.c().row(ctx, `SELECT due_on FROM goal_deadlines WHERE user_id = $1 AND goal_id = $2`, userID, goalID).Scan(&due)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return due, err
}

// GoalDeadlines returns every due date of the user's goals (goal id → YYYY-MM-DD).
func (s *SQLiteStore) GoalDeadlines(ctx context.Context, userID string) (map[string]string, error) {
	type row struct{ id, due string }
	list, err := collect(s.c().query(ctx, `SELECT goal_id, due_on FROM goal_deadlines WHERE user_id = $1`, userID),
		func(r rowScanner) (row, error) {
			var x row
			err := r.Scan(&x.id, &x.due)
			return x, err
		})
	if err != nil {
		return nil, fmt.Errorf("goal deadlines: %w", err)
	}
	out := make(map[string]string, len(list))
	for _, x := range list {
		out[x.id] = x.due
	}
	return out, nil
}

// SetGoalDeadline sets (dueOn YYYY-MM-DD) or clears ("") a goal's due date. ErrNotFound for another user's goal.
func (s *SQLiteStore) SetGoalDeadline(ctx context.Context, userID, goalID, dueOn string) error {
	var owner string
	err := s.c().row(ctx, `SELECT user_id FROM user_goals WHERE id = $1`, goalID).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && owner != userID) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if dueOn == "" {
		_, err = s.c().exec(ctx, `DELETE FROM goal_deadlines WHERE goal_id = $1`, goalID)
		return err
	}
	_, err = s.c().exec(ctx, `INSERT INTO goal_deadlines (goal_id, user_id, due_on, updated_at) VALUES ($1, $2, $3, NOW())
		ON CONFLICT (goal_id) DO UPDATE SET due_on = excluded.due_on, updated_at = NOW()`, goalID, userID, dueOn)
	return err
}
