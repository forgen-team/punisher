package store

import (
	"context"
	"time"
)

// StudyStore is the W5a study-feature part of the store (prefs, leeches,
// forecast, activity, deadlines, review cards, search). Kept apart from Store
// so the two interfaces grow without touching each other; SQLiteStore has both.
type StudyStore interface {
	GetStudyPrefs(ctx context.Context, userID string) (StudyPrefs, error)
	UpdateStudyPrefs(ctx context.Context, userID string, p StudyPrefsPatch) (StudyPrefs, error)
	MarkReminded(ctx context.Context, userID, day string) error
	Leeches(ctx context.Context, userID, lang string) (leeches, suspended []Leech, err error)
	SetLeech(ctx context.Context, userID, objectiveID, action string) error
	ReviewForecast(ctx context.Context, userID string, now time.Time, days int) ([]ForecastDay, int, error)
	Activity(ctx context.Context, userID string, now time.Time, days int) ([]ActivityDay, error)
	GoalDeadline(ctx context.Context, userID, goalID string) (string, error)
	GoalDeadlines(ctx context.Context, userID string) (map[string]string, error)
	SetGoalDeadline(ctx context.Context, userID, goalID, dueOn string) error
	CardDraftFor(ctx context.Context, userID, kind, sourceID string) (CardDraft, error)
	SaveCard(ctx context.Context, userID, kind, sourceID, question, answer string, now time.Time) (ReviewCard, error)
	SaveExternalCard(ctx context.Context, userID string, in ExternalCard, now time.Time) (ReviewCard, error)
	PatchCard(ctx context.Context, userID, cardID string, p CardPatch, now time.Time) (ReviewCard, error)
	PatchExternalCard(ctx context.Context, userID, externalID string, p CardPatch, now time.Time) (ReviewCard, error)
	DeleteCard(ctx context.Context, userID, cardID string, now time.Time) error
	RestoreCard(ctx context.Context, userID, cardID string) (ReviewCard, error)
	PurgeDeletedCards(ctx context.Context, now time.Time) (int64, error)
	DueCards(ctx context.Context, userID string, now time.Time, limit int) ([]ReviewCard, error)
	ListCards(ctx context.Context, userID string) ([]ReviewCard, error)
	GradeCard(ctx context.Context, userID, cardID string, rating int, now time.Time) (ReviewCard, error)
	CardCounts(ctx context.Context, userID string, now time.Time) (due, doneToday int, err error)
	Search(ctx context.Context, userID, q string, limit int) ([]SearchHit, error)
}

var _ StudyStore = (*SQLiteStore)(nil)
