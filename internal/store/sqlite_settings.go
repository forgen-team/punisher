package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

// GetLearningSettings returns the user's learning settings, or the defaults
// when the user has not saved any.
func (s *SQLiteStore) GetLearningSettings(ctx context.Context, userID string) (models.LearningSettings, error) {
	out := models.LearningSettings{ExampleLanguage: learning.DefaultExampleLanguage, DailyMinutes: learning.DefaultDailyMinutes}
	var daily int
	var lang string
	var updated time.Time
	err := s.c().row(ctx, `
		SELECT example_language, daily_minutes, updated_at FROM user_learning_settings WHERE user_id = $1`, userID).
		Scan(&lang, &daily, ts(&updated))
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, fmt.Errorf("load learning settings: %w", err)
	}
	out.ExampleLanguage = learning.NormalizeExampleLanguage(lang)
	out.DailyMinutes = daily
	out.UpdatedAt = &updated
	return out, nil
}

// SaveLearningSettings upserts the user's settings; "" lang / 0 daily keep the
// current value. Values must already be validated.
func (s *SQLiteStore) SaveLearningSettings(ctx context.Context, userID, lang string, daily int) (models.LearningSettings, error) {
	out := models.LearningSettings{}
	var updated time.Time
	if err := s.c().row(ctx, `
		INSERT INTO user_learning_settings (user_id, example_language, daily_minutes, updated_at)
		VALUES ($1, COALESCE(NULLIF($2, ''), $4), COALESCE(NULLIF($3, 0), $5), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			example_language = COALESCE(NULLIF($2, ''), user_learning_settings.example_language),
			daily_minutes = COALESCE(NULLIF($3, 0), user_learning_settings.daily_minutes),
			updated_at = NOW()
		RETURNING example_language, daily_minutes, updated_at`,
		userID, lang, daily, learning.DefaultExampleLanguage, learning.DefaultDailyMinutes).
		Scan(&out.ExampleLanguage, &out.DailyMinutes, ts(&updated)); err != nil {
		return out, fmt.Errorf("save learning settings: %w", err)
	}
	out.UpdatedAt = &updated
	return out, nil
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
