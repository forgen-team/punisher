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
	out := models.LearningSettings{ExampleLanguage: learning.DefaultExampleLanguage}
	var lang string
	var updated time.Time
	err := s.c().row(ctx, `
		SELECT example_language, updated_at FROM user_learning_settings WHERE user_id = $1`, userID).Scan(&lang, ts(&updated))
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, fmt.Errorf("load learning settings: %w", err)
	}
	out.ExampleLanguage = learning.NormalizeExampleLanguage(lang)
	out.UpdatedAt = &updated
	return out, nil
}

// SaveLearningSettings upserts the user's settings. lang must already be validated.
func (s *SQLiteStore) SaveLearningSettings(ctx context.Context, userID, lang string) (models.LearningSettings, error) {
	var updated time.Time
	if err := s.c().row(ctx, `
		INSERT INTO user_learning_settings (user_id, example_language, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE SET example_language = excluded.example_language, updated_at = NOW()
		RETURNING updated_at`, userID, lang).Scan(ts(&updated)); err != nil {
		return models.LearningSettings{}, fmt.Errorf("save learning settings: %w", err)
	}
	return models.LearningSettings{ExampleLanguage: lang, UpdatedAt: &updated}, nil
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
