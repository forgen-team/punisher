package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/models"
)

const settingsColumns = `example_language, daily_minutes, mastery_target, review_intervals, review_minutes, review_grader, pretest_enabled, lab_os, adaptive_intervals, locale, reader_level, area_familiarity, updated_at,
	level_suggestion_dismissed_at, reader_level_changed_at, ai_provider`

func defaultLearningSettings() models.LearningSettings {
	return models.LearningSettings{
		ExampleLanguage: learning.DefaultExampleLanguage, DailyMinutes: learning.DefaultDailyMinutes,
		MasteryTarget: learning.DefaultMasteryTarget, ReviewIntervals: append([]int{}, learning.DefaultReviewIntervals...),
		ReviewMinutes: learning.DefaultReviewMinutes, ReviewGrader: learning.DefaultReviewGrader,
		PretestEnabled: learning.DefaultPretestEnabled, LabOS: defaultLabOS(),
		AdaptiveIntervals: learning.DefaultAdaptiveIntervals, Locale: defaultLocale(),
		AreaFamiliarity: map[string]string{}, AIProvider: llm.DefaultProvider,
	}
}

// defaultLabOS: the standalone app runs on the reader's own machine, so the lab
// environment defaults to its OS (lab_os ” = not chosen).
func defaultLabOS() string {
	if runtime.GOOS == "linux" {
		return learning.LabOSLinux
	}
	return learning.LabOSMacOS
}

func scanSettings(row rowScanner) (models.LearningSettings, error) {
	var out models.LearningSettings
	var intervals, familiarity string
	var updated time.Time
	err := row.Scan(&out.ExampleLanguage, &out.DailyMinutes, &out.MasteryTarget, &intervals, &out.ReviewMinutes, &out.ReviewGrader, &out.PretestEnabled,
		&out.LabOS, &out.AdaptiveIntervals, &out.Locale, &out.ReaderLevel, &familiarity, ts(&updated),
		nts(&out.LevelSuggestionDismissedAt), nts(&out.ReaderLevelChangedAt), &out.AIProvider)
	if err != nil {
		return out, err
	}
	out.UpdatedAt = &updated
	out.ExampleLanguage = learning.NormalizeExampleLanguage(out.ExampleLanguage)
	if !learning.ValidLocale(out.Locale) {
		out.Locale = defaultLocale()
	}
	if !learning.ValidLabOS(out.LabOS) {
		out.LabOS = defaultLabOS()
	}
	if jerr := json.Unmarshal([]byte(intervals), &out.ReviewIntervals); jerr != nil || !learning.ValidReviewIntervals(out.ReviewIntervals) {
		out.ReviewIntervals = append([]int{}, learning.DefaultReviewIntervals...)
	}
	if !learning.ValidReaderLevel(out.ReaderLevel) {
		out.ReaderLevel = "" // 모르는 값은 '아직 정하지 않음'으로 읽는다(처음 설정을 다시 띄운다).
	}
	out.AreaFamiliarity = scanFamiliarity(familiarity)
	out.AIProvider = llm.NormalizeProvider(out.AIProvider) // 모르는 값은 claude 로 읽는다.
	return out, nil
}

// scanFamiliarity decodes area_familiarity, keeping only known areas and values (never nil).
func scanFamiliarity(raw string) map[string]string {
	out := map[string]string{}
	var stored map[string]string
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		log.Printf("[store] area familiarity %q: %v (reading as {})", raw, err)
		return out
	}
	for area, f := range stored {
		if curriculum.IsValidArea(area) && learning.ValidFamiliarity(f) {
			out[area] = f
		}
	}
	return out
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
	PretestEnabled  *bool
	LabOS           *string
	// AdaptiveIntervals false resets every objective's ease to 2.0 (v3 §7).
	AdaptiveIntervals *bool
	Locale            *string
	// ReaderLevel: one of the four levels (never "" — "" is only the unsaved state).
	ReaderLevel *string
	// AreaFamiliarity replaces the whole map when non-nil (an empty map clears it).
	AreaFamiliarity map[string]string
	// AIProvider: claude | codex | agy (llm.ValidProvider).
	AIProvider *string
}

// UpdateLearningSettings upserts the given fields and keeps the rest (defaults on first save).
func (s *SQLiteStore) UpdateLearningSettings(ctx context.Context, userID string, p LearningSettingsPatch) (models.LearningSettings, error) {
	intervals, err := jsonParam(p.ReviewIntervals != nil, p.ReviewIntervals)
	if err != nil {
		return models.LearningSettings{}, err
	}
	familiarity, err := jsonParam(p.AreaFamiliarity != nil, p.AreaFamiliarity)
	if err != nil {
		return models.LearningSettings{}, err
	}
	def := defaultLearningSettings()
	defIntervals, _ := json.Marshal(def.ReviewIntervals)
	out, err := scanSettings(s.c().row(ctx, `
		INSERT INTO user_learning_settings (user_id, example_language, daily_minutes, mastery_target, review_intervals,
			review_minutes, review_grader, pretest_enabled, lab_os, adaptive_intervals, locale, reader_level, area_familiarity, updated_at,
			reader_level_changed_at, ai_provider)
		VALUES ($1, COALESCE($2, $8), COALESCE($3, $9), COALESCE($4, $10),
			COALESCE($5, $11), COALESCE($6, $12), COALESCE($7, $13),
			COALESCE($14, $15), COALESCE($16, ''), COALESCE($17, $18), COALESCE($19, $20),
			COALESCE($21, ''), COALESCE($22, '{}'), NOW(), CASE WHEN $21 IS NOT NULL THEN NOW() END, COALESCE($23, $24))
		ON CONFLICT (user_id) DO UPDATE SET
			example_language = COALESCE($2, user_learning_settings.example_language),
			daily_minutes = COALESCE($3, user_learning_settings.daily_minutes),
			mastery_target = COALESCE($4, user_learning_settings.mastery_target),
			review_intervals = COALESCE($5, user_learning_settings.review_intervals),
			review_minutes = COALESCE($6, user_learning_settings.review_minutes),
			review_grader = COALESCE($7, user_learning_settings.review_grader),
			pretest_enabled = COALESCE($14, user_learning_settings.pretest_enabled),
			lab_os = COALESCE($16, user_learning_settings.lab_os),
			adaptive_intervals = COALESCE($17, user_learning_settings.adaptive_intervals),
			locale = COALESCE($19, user_learning_settings.locale),
			reader_level = COALESCE($21, user_learning_settings.reader_level),
			reader_level_changed_at = CASE WHEN $21 IS NOT NULL AND $21 <> user_learning_settings.reader_level
				THEN NOW() ELSE user_learning_settings.reader_level_changed_at END,
			area_familiarity = COALESCE($22, user_learning_settings.area_familiarity),
			ai_provider = COALESCE($23, user_learning_settings.ai_provider),
			updated_at = NOW()
		RETURNING `+settingsColumns,
		userID, p.ExampleLanguage, p.DailyMinutes, p.MasteryTarget, intervals, p.ReviewMinutes, p.ReviewGrader,
		def.ExampleLanguage, def.DailyMinutes, def.MasteryTarget, string(defIntervals), def.ReviewMinutes, def.ReviewGrader,
		p.PretestEnabled, def.PretestEnabled, p.LabOS, p.AdaptiveIntervals, def.AdaptiveIntervals, p.Locale, def.Locale,
		p.ReaderLevel, familiarity, p.AIProvider, def.AIProvider))
	if err != nil {
		return out, fmt.Errorf("save learning settings: %w", err)
	}
	if p.AdaptiveIntervals != nil && !*p.AdaptiveIntervals {
		if _, err := s.c().exec(ctx, `UPDATE user_objective_mastery SET ease = $2 WHERE user_id = $1 AND ease <> $2`,
			userID, learning.EaseDefault); err != nil {
			return out, fmt.Errorf("reset ease: %w", err)
		}
	}
	return out, nil
}

// DismissLevelSuggestion records that the user closed the lower-level
// suggestion at at (hidden for learning.LevelSuggestionWindow). Without saved
// settings there is nothing to suggest (no chosen level), so nothing is stored.
func (s *SQLiteStore) DismissLevelSuggestion(ctx context.Context, userID string, at time.Time) error {
	if _, err := s.c().exec(ctx, `UPDATE user_learning_settings SET level_suggestion_dismissed_at = $2 WHERE user_id = $1`,
		userID, at); err != nil {
		return fmt.Errorf("dismiss level suggestion: %w", err)
	}
	return nil
}

// jsonParam is v as a JSON text parameter when set, else nil (COALESCE keeps the column).
func jsonParam(set bool, v any) (any, error) {
	if !set {
		return nil, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
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

// ReaderLevelFor is the user's effective reader level (learning.EffectiveReaderLevel:
// not chosen → practitioner); falls back to practitioner on error.
func (s *SQLiteStore) ReaderLevelFor(ctx context.Context, userID string) string {
	st, err := s.GetLearningSettings(ctx, userID)
	if err != nil {
		log.Printf("[store] reader level for %s: %v (using default)", userID, err)
		return learning.DefaultReaderLevel
	}
	return learning.EffectiveReaderLevel(st.ReaderLevel)
}

// LocaleFor is the user's locale (ko|en); falls back to the default on error.
func (s *SQLiteStore) LocaleFor(ctx context.Context, userID string) string {
	st, err := s.GetLearningSettings(ctx, userID)
	if err != nil {
		log.Printf("[store] locale for %s: %v (using default)", userID, err)
		return defaultLocale()
	}
	return st.Locale
}

// lastAIProvider remembers each store's last successfully read provider per
// user (key aiProviderKey) so a transient DB error does not silently switch
// generation to the default AI.
var lastAIProvider sync.Map

type aiProviderKey struct {
	store  *SQLiteStore
	userID string
}

// AIProviderFor is the user's selected AI (claude | codex | agy). On a read
// error it is the last value read successfully, else the default (claude).
func (s *SQLiteStore) AIProviderFor(ctx context.Context, userID string) string {
	key := aiProviderKey{s, userID}
	st, err := s.GetLearningSettings(ctx, userID)
	if err != nil {
		if last, ok := lastAIProvider.Load(key); ok {
			log.Printf("[store] ai provider for %s: %v (using last read %q)", userID, err, last)
			return last.(string)
		}
		log.Printf("[store] ai provider for %s: %v (using default)", userID, err)
		return llm.DefaultProvider
	}
	lastAIProvider.Store(key, st.AIProvider)
	return st.AIProvider
}
