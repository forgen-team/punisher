package learning

import "time"

// Reading-thread modes (lecture_questions.mode, learner-level design ④).
const (
	ThreadEasier   = "easier"   // explain the passage (or the previous answer) more simply
	ThreadDeeper   = "deeper"   // explain it more deeply
	ThreadQuestion = "question" // the reader's own question (question text required)
)

// ValidThreadMode reports one of the three thread modes.
func ValidThreadMode(mode string) bool {
	return mode == ThreadEasier || mode == ThreadDeeper || mode == ThreadQuestion
}

// Level suggestion (learner-level ④ 수준 보정 신호): many '더 쉽게' requests in a
// week suggest one level lower; closing the suggestion hides it for a week.
const (
	LevelSuggestionWindow    = 7 * 24 * time.Hour
	LevelSuggestionMinEasier = 5
	LevelSuggestionEasierWhy = "easier_often"
)

// LowerReaderLevel is one level below level (advanced→practitioner→novice→beginner);
// "" for beginner, the unset level "" and unknown values.
func LowerReaderLevel(level string) string {
	switch level {
	case ReaderAdvanced:
		return ReaderPractitioner
	case ReaderPractitioner:
		return ReaderNovice
	case ReaderNovice:
		return ReaderBeginner
	}
	return ""
}

// LevelSuggestionSince is the start of the '더 쉽게' counting window: the last
// week, but not before the level last changed (a just-lowered level is not
// asked to go lower again on the same requests).
func LevelSuggestionSince(now time.Time, levelChangedAt *time.Time) time.Time {
	since := now.Add(-LevelSuggestionWindow)
	if levelChangedAt != nil && levelChangedAt.After(since) {
		return *levelChangedAt
	}
	return since
}

// SuggestLowerLevel is the level to suggest ("" = none): the saved level has a
// lower one, the suggestion was not closed within the window, and easier
// counts at least LevelSuggestionMinEasier requests since LevelSuggestionSince.
func SuggestLowerLevel(level string, easier int, dismissedAt *time.Time, now time.Time) string {
	to := LowerReaderLevel(level)
	if to == "" || easier < LevelSuggestionMinEasier {
		return ""
	}
	if dismissedAt != nil && now.Sub(*dismissedAt) < LevelSuggestionWindow {
		return ""
	}
	return to
}
