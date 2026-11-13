package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

const settingsColumns = `example_language, daily_minutes, mastery_target, review_intervals, review_minutes, review_grader, updated_at`

func defaultLearningSettings() models.LearningSettings {
	return models.LearningSettings{
		ExampleLanguage: learning.DefaultExampleLanguage, DailyMinutes: learning.DefaultDailyMinutes,
		MasteryTarget: learning.DefaultMasteryTarget, ReviewIntervals: append([]int{}, learning.DefaultReviewIntervals...),
		ReviewMinutes: learning.DefaultReviewMinutes, ReviewGrader: learning.DefaultReviewGrader,
	}
}

func scanSettings(row rowScanner) (models.LearningSettings, error) {
	var out models.LearningSettings
	var intervals string
	var updated time.Time
	err := row.Scan(&out.ExampleLanguage, &out.DailyMinutes, &out.MasteryTarget, &intervals, &out.ReviewMinutes, &out.ReviewGrader, ts(&updated))
	if err != nil {
		return out, err
	}
	out.UpdatedAt = &updated
	out.ExampleLanguage = learning.NormalizeExampleLanguage(out.ExampleLanguage)
	if jerr := json.Unmarshal([]byte(intervals), &out.ReviewIntervals); jerr != nil || !learning.ValidReviewIntervals(out.ReviewIntervals) {
		out.ReviewIntervals = append([]int{}, learning.DefaultReviewIntervals...)
	}
	return out, nil
}

// GetLearningSettings returns the user's learning settings, or the defaults
// when the user has not saved any.
func (s *SQLiteStore) GetLearningSettings(ctx context.Context, userID string) (models.LearningSettings, error) {
	out, err := scanSettings(s.c().row(ctx, `SELECT `+settingsColumns+` FROM user_learning_settings WHERE user_id = $1`, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return defaultLearningSettings(), nil
	}
	if err != nil {
		return defaultLearningSettings(), fmt.Errorf("load learning settings: %w", err)
	}
	return out, nil
}

// LearningSettingsPatch changes only the non-nil fields. Values must already be validated.
type LearningSettingsPatch struct {
	ExampleLanguage *string
	DailyMinutes    *int
	MasteryTarget   *int
	ReviewIntervals []int // nil = keep
	ReviewMinutes   *int
	ReviewGrader    *string
}

// UpdateLearningSettings upserts the given fields and keeps the rest (defaults on first save).
func (s *SQLiteStore) UpdateLearningSettings(ctx context.Context, userID string, p LearningSettingsPatch) (models.LearningSettings, error) {
	var intervals any
	if p.ReviewIntervals != nil {
		raw, err := json.Marshal(p.ReviewIntervals)
		if err != nil {
			return models.LearningSettings{}, err
		}
		intervals = string(raw)
	}
	def := defaultLearningSettings()
	defIntervals, _ := json.Marshal(def.ReviewIntervals)
	out, err := scanSettings(s.c().row(ctx, `
		INSERT INTO user_learning_settings (user_id, example_language, daily_minutes, mastery_target, review_intervals,
			review_minutes, review_grader, updated_at)
		VALUES ($1, COALESCE($2, $8), COALESCE($3, $9), COALESCE($4, $10),
			COALESCE($5, $11), COALESCE($6, $12), COALESCE($7, $13), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			example_language = COALESCE($2, user_learning_settings.example_language),
			daily_minutes = COALESCE($3, user_learning_settings.daily_minutes),
			mastery_target = COALESCE($4, user_learning_settings.mastery_target),
			review_intervals = COALESCE($5, user_learning_settings.review_intervals),
			review_minutes = COALESCE($6, user_learning_settings.review_minutes),
			review_grader = COALESCE($7, user_learning_settings.review_grader),
			updated_at = NOW()
		RETURNING `+settingsColumns,
		userID, p.ExampleLanguage, p.DailyMinutes, p.MasteryTarget, intervals, p.ReviewMinutes, p.ReviewGrader,
		def.ExampleLanguage, def.DailyMinutes, def.MasteryTarget, string(defIntervals), def.ReviewMinutes, def.ReviewGrader))
	if err != nil {
		return out, fmt.Errorf("save learning settings: %w", err)
	}
	return out, nil
}

// SaveLearningSettings upserts the user's settings; "" lang / 0 daily keep the
// current value. Values must already be validated. (Kept for callers; see UpdateLearningSettings.)
func (s *SQLiteStore) SaveLearningSettings(ctx context.Context, userID, lang string, daily int) (models.LearningSettings, error) {
	var p LearningSettingsPatch
	if lang != "" {
		p.ExampleLanguage = &lang
	}
	if daily != 0 {
		p.DailyMinutes = &daily
	}
	return s.UpdateLearningSettings(ctx, userID, p)
}

// ExampleLanguageFor is the user's example language; falls back to the default on error.
func (s *SQLiteStore) ExampleLanguageFor(ctx context.Context, userID string) string {
	st, err := s.GetLearningSettings(ctx, userID)
	if err != nil {
		log.Printf("[store] example language for %s: %v (using default)", userID, err)
		return learning.DefaultExampleLanguage
	}
	return st.ExampleLanguage
}
