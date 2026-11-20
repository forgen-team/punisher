package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

type languageOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func languageOptions() []languageOption {
	out := make([]languageOption, 0, len(learning.ExampleLanguageIDs))
	for _, id := range learning.ExampleLanguageIDs {
		out = append(out, languageOption{ID: id, Label: learning.ExampleLanguages[id]})
	}
	return out
}

// GetSettings handles GET /v1/me/settings (defaults when never saved).
func (h *LearnHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	st, err := h.Store.GetLearningSettings(r.Context(), userID)
	if err != nil {
		writeServiceError(w, r, "get settings", err)
		return
	}
	writeJSON(w, http.StatusOK, h.settingsPayload(r, userID, st))
}

// settingsPayload is the GET/PUT settings response: the settings, the example
// languages and levelSuggestion ({to, reason} or null, learner-level ④).
func (h *LearnHandler) settingsPayload(r *http.Request, userID string, st models.LearningSettings) map[string]any {
	suggestion, err := h.Svc.SuggestLevel(r.Context(), userID, st)
	if err != nil { // 제안은 부가 정보 — 설정 화면을 막지 않는다.
		log.Printf("level suggestion %s: %v", userID, err)
		suggestion = nil
	}
	return map[string]any{"settings": st, "exampleLanguages": languageOptions(), "levelSuggestion": suggestion}
}

type settingsBody struct {
	ExampleLanguage *string `json:"exampleLanguage"`
	DailyMinutes    *int    `json:"dailyMinutes"`
	MasteryTarget   *int    `json:"masteryTarget"`
	ReviewIntervals []int   `json:"reviewIntervals"`
	ReviewMinutes   *int    `json:"reviewMinutes"`
	ReviewGrader    *string `json:"reviewGrader"`
	PretestEnabled  *bool   `json:"pretestEnabled"`
	LabOS           *string `json:"labOs"`
	// AdaptiveIntervals false resets every objective's ease (interval scale) to the ladder.
	AdaptiveIntervals *bool   `json:"adaptiveIntervals"`
	Locale            *string `json:"locale"`
	// ReaderLevel: beginner|novice|practitioner|advanced ("" is rejected — only the unsaved state).
	ReaderLevel *string `json:"readerLevel"`
	// AreaFamiliarity replaces the saved map: {"<area>": "new|heard|can_explain"} ({} clears it).
	AreaFamiliarity map[string]string `json:"areaFamiliarity"`
	// AIProvider: claude|codex|agy — the AI used for every generation from the next call on.
	AIProvider *string `json:"aiProvider"`
}

var dailyMinuteOptions = map[int]bool{15: true, 30: true, 45: true, 60: true}

// Codes of the settings 400 responses (error.code; the web shows a message per code).
const (
	ErrorInvalidLanguage      ErrorCode = "INVALID_LANGUAGE"
	ErrorInvalidGrader        ErrorCode = "INVALID_GRADER"
	ErrorInvalidDailyMinutes  ErrorCode = "INVALID_DAILY_MINUTES"
	ErrorInvalidMasteryTarget ErrorCode = "INVALID_MASTERY_TARGET"
	ErrorInvalidIntervals     ErrorCode = "INVALID_INTERVALS"
	ErrorInvalidReviewMinutes ErrorCode = "INVALID_REVIEW_MINUTES"
	ErrorInvalidLabOS         ErrorCode = "INVALID_LAB_OS"
	ErrorInvalidLocale        ErrorCode = "INVALID_LOCALE"
	ErrorInvalidReaderLevel   ErrorCode = "INVALID_READER_LEVEL"
	ErrorInvalidFamiliarity   ErrorCode = "INVALID_AREA_FAMILIARITY"
	ErrorSettingsRequired     ErrorCode = "SETTINGS_REQUIRED"
	ErrorInvalidAIProvider    ErrorCode = "INVALID_AI_PROVIDER"
)

// settingsError is a rejected field: the 400 code and the (English) message.
type settingsError struct {
	code ErrorCode
	msg  string
}

// settingsPatch validates the given fields; nil error = valid.
func settingsPatch(body settingsBody) (store.LearningSettingsPatch, *settingsError) {
	p := store.LearningSettingsPatch{DailyMinutes: body.DailyMinutes, MasteryTarget: body.MasteryTarget,
		ReviewIntervals: body.ReviewIntervals, ReviewMinutes: body.ReviewMinutes, PretestEnabled: body.PretestEnabled,
		AdaptiveIntervals: body.AdaptiveIntervals}
	if invalid := readerPatch(&p, body); invalid != nil {
		return p, invalid
	}
	if body.AIProvider != nil {
		provider := strings.ToLower(strings.TrimSpace(*body.AIProvider))
		if !llm.ValidProvider(provider) {
			return p, &settingsError{ErrorInvalidAIProvider, "aiProvider must be " + strings.Join(llm.Providers, "|")}
		}
		p.AIProvider = &provider
	}
	if body.Locale != nil {
		l := strings.ToLower(strings.TrimSpace(*body.Locale))
		if !learning.ValidLocale(l) {
			return p, &settingsError{ErrorInvalidLocale, "locale must be ko|en"}
		}
		p.Locale = &l
	}
	if body.LabOS != nil {
		os := strings.ToLower(strings.TrimSpace(*body.LabOS))
		if !learning.ValidLabOS(os) {
			return p, &settingsError{ErrorInvalidLabOS, "labOs must be macos|linux|none"}
		}
		p.LabOS = &os
	}
	if body.ExampleLanguage != nil {
		lang := strings.ToLower(strings.TrimSpace(*body.ExampleLanguage))
		if !learning.IsValidExampleLanguage(lang) {
			return p, &settingsError{ErrorInvalidLanguage, "exampleLanguage must be one of " + strings.Join(learning.ExampleLanguageIDs, "|")}
		}
		p.ExampleLanguage = &lang
	}
	if body.ReviewGrader != nil {
		grader := strings.ToLower(strings.TrimSpace(*body.ReviewGrader))
		if !learning.ValidReviewGrader(grader) {
			return p, &settingsError{ErrorInvalidGrader, "reviewGrader must be self|llm"}
		}
		p.ReviewGrader = &grader
	}
	switch {
	case body.DailyMinutes != nil && !dailyMinuteOptions[*body.DailyMinutes]:
		return p, &settingsError{ErrorInvalidDailyMinutes, "dailyMinutes must be 15|30|45|60"}
	case body.MasteryTarget != nil && !learning.ValidMasteryTarget(*body.MasteryTarget):
		return p, &settingsError{ErrorInvalidMasteryTarget, "masteryTarget must be 3~5"}
	case body.ReviewIntervals != nil && !learning.ValidReviewIntervals(body.ReviewIntervals):
		return p, &settingsError{ErrorInvalidIntervals, "reviewIntervals must be 1~6 increasing positive day counts"}
	case body.ReviewMinutes != nil && !learning.ValidReviewMinutes(*body.ReviewMinutes):
		return p, &settingsError{ErrorInvalidReviewMinutes, "reviewMinutes must be 0~30"}
	case p.ExampleLanguage == nil && p.DailyMinutes == nil && p.MasteryTarget == nil && p.ReviewIntervals == nil &&
		p.ReviewMinutes == nil && p.ReviewGrader == nil && p.PretestEnabled == nil && p.LabOS == nil && p.AdaptiveIntervals == nil && p.Locale == nil &&
		p.ReaderLevel == nil && p.AreaFamiliarity == nil && p.AIProvider == nil:
		return p, &settingsError{ErrorSettingsRequired, "at least one setting is required"}
	}
	return p, nil
}

// readerPatch validates the reader level and the area familiarity (learner-level ①).
func readerPatch(p *store.LearningSettingsPatch, body settingsBody) *settingsError {
	if body.ReaderLevel != nil {
		level := strings.ToLower(strings.TrimSpace(*body.ReaderLevel))
		if !learning.ValidReaderLevel(level) {
			return &settingsError{ErrorInvalidReaderLevel, "readerLevel must be " + strings.Join(learning.ReaderLevels, "|")}
		}
		p.ReaderLevel = &level
	}
	if body.AreaFamiliarity != nil {
		if !learning.ValidAreaFamiliarity(body.AreaFamiliarity) {
			return &settingsError{ErrorInvalidFamiliarity, "areaFamiliarity keys must be known areas and values new|heard|can_explain"}
		}
		p.AreaFamiliarity = body.AreaFamiliarity
	}
	return nil
}

// PutSettings handles PUT /v1/me/settings (partial: only the given fields change).
// Affects lectures/answers generated afterwards; existing lectures are not regenerated.
func (h *LearnHandler) PutSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body settingsBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	patch, invalid := settingsPatch(body)
	if invalid != nil {
		writeError(w, r, http.StatusBadRequest, invalid.code, invalid.msg)
		return
	}
	st, err := h.Store.UpdateLearningSettings(r.Context(), userID, patch)
	if err != nil {
		writeServiceError(w, r, "save settings", err)
		return
	}
	writeJSON(w, http.StatusOK, h.settingsPayload(r, userID, st))
}
