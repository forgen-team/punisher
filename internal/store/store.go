package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/wooo-jin/punisher/internal/curriculum"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

// Store is the persistence API used by handlers, services and the scheduler.
// SQLiteStore is the only implementation; tests run it on a temp file.
type Store interface {
	Ping(ctx context.Context) error
	Close() error

	// curriculum & concepts
	SyncCurriculum(ctx context.Context, seed *curriculum.Seed) (bool, error)
	LatestCurriculumVersion(ctx context.Context) (string, error)
	ListConcepts(ctx context.Context, userID string, f ConceptFilter) ([]models.Concept, error)
	GetConceptDetail(ctx context.Context, userID, conceptID string) (*models.ConceptDetail, error)
	LoadGraph(ctx context.Context, userID string) (*models.GraphData, error)
	LoadMapExtras(ctx context.Context, userID string, data *models.GraphData) error
	ResolveConcepts(ctx context.Context, normalizedQuery string) ([]string, error)
	ConceptInfo(ctx context.Context, conceptID string) (models.ConceptInfo, error)
	DoneSet(ctx context.Context, userID, conceptID string) (learning.DoneSet, error)
	ConceptCatalog(ctx context.Context, userID string) ([]ConceptRef, error)
	RelatedConcepts(ctx context.Context, userID, conceptID string) ([]ConceptRef, error)
	KnownConceptIDs(ctx context.Context) (map[string]bool, error)
	ProposeConcept(ctx context.Context, userID string, p models.ProposedConcept) (string, error)
	GetGloss(ctx context.Context, conceptID string) (string, error)
	SaveGloss(ctx context.Context, conceptID, gloss, model string) error
	LoadToday(ctx context.Context, userID string) (*TodayData, error)
	PrerequisiteEdges(ctx context.Context) ([]learning.PrereqEdge, error)
	AliasIndex(ctx context.Context) (learning.AliasIndex, error)

	// learning state
	SetConceptState(ctx context.Context, userID, conceptID, requested string) (string, error)
	UpdateProgress(ctx context.Context, userID, lectureID, anchor string, percent int) error
	CompleteLecture(ctx context.Context, userID, lectureID string) (CompleteResult, error)
	InsertFeedback(ctx context.Context, userID, lectureID, feedback, anchor, comment, altRequestID string) (string, error)
	LatestFeedback(ctx context.Context, userID string) (string, error)
	DismissLecture(ctx context.Context, userID, lectureID string) error
	RestoreLecture(ctx context.Context, userID, lectureID string) error
	DismissedConceptIDs(ctx context.Context, userID string) (map[string]bool, error)
	GetLearningSettings(ctx context.Context, userID string) (models.LearningSettings, error)
	SaveLearningSettings(ctx context.Context, userID, lang string) (models.LearningSettings, error)
	ExampleLanguageFor(ctx context.Context, userID string) string

	// lectures
	InsertLecture(ctx context.Context, l NewLecture) (models.Lecture, error)
	InsertReview(ctx context.Context, lectureID string, r models.LectureReview) error
	LatestLectureForTarget(ctx context.Context, kind, conceptID, objectiveKey string, depth int, statuses []string) (models.Lecture, error)
	PublishedLectureCovering(ctx context.Context, userID, conceptID string, depth int, objectiveIDs []string) (models.Lecture, error)
	GetLecture(ctx context.Context, userID, lectureID string) (models.Lecture, error)

	// history
	ListMyQuestions(ctx context.Context, userID, conceptID string, limit int, cursor string) ([]models.MyQuestion, string, error)
	ListMyLectures(ctx context.Context, userID, status string, limit int, cursor string) ([]models.MyLecture, string, error)
	ListConceptLectures(ctx context.Context, userID, conceptID string) ([]models.ConceptLecture, error)

	// generation jobs
	CreateJob(ctx context.Context, j NewJob) (models.GenerationJob, bool, error)
	ActiveJobForTarget(ctx context.Context, objectiveKey string, depth int) (models.GenerationJob, error)
	GetJob(ctx context.Context, userID, jobID string) (models.GenerationJob, error)
	StartJob(ctx context.Context, jobID string) error
	FinishJob(ctx context.Context, jobID string, o JobOutcome) error
	RecordUsage(ctx context.Context, userID, kind, conceptID string, o JobOutcome) error
	UsageSince(ctx context.Context, userID string, since time.Time) (lectures int, tokens int, err error)
	FailStaleJobs(ctx context.Context, olderThan time.Duration) (int64, error)

	// notes, goals, questions
	ListNotes(ctx context.Context, userID, lectureID, conceptID string) ([]models.Note, error)
	CreateNote(ctx context.Context, userID string, n models.Note) (models.Note, error)
	UpdateNote(ctx context.Context, userID, id string, anchor, quote, body *string) (models.Note, error)
	DeleteNote(ctx context.Context, userID, id string) error
	ListGoals(ctx context.Context, userID, status string) ([]models.Goal, error)
	GetGoal(ctx context.Context, userID, id string) (models.Goal, error)
	CreateGoal(ctx context.Context, userID, title string, targets []string) (models.Goal, error)
	UpdateGoal(ctx context.Context, userID, id string, title *string, targets []string, status *string) (models.Goal, error)
	DeleteGoal(ctx context.Context, userID, id string) error
	CreateQuestion(ctx context.Context, userID string, q models.Question) (models.Question, error)
	AnswerQuestion(ctx context.Context, id, answer, model string, linked []string, proposed json.RawMessage) (models.Question, error)
	FailQuestion(ctx context.Context, id, model, errMsg string) (models.Question, error)
	ListQuestions(ctx context.Context, userID, lectureID string) ([]models.Question, error)
	GetQuestion(ctx context.Context, userID, id string) (models.Question, error)

	// study requests
	CreateStudyRequest(ctx context.Context, req models.StudyRequest) (models.StudyRequest, bool, error)
	ListStudyRequests(ctx context.Context, userID, status string, limit int) ([]models.StudyRequest, error)
	GetStudyRequest(ctx context.Context, userID, id string) (models.StudyRequest, error)
	UnresolvedStudyRequests(ctx context.Context, userID string, maxAttempts, limit int) ([]models.StudyRequest, error)
	RecordResolution(ctx context.Context, id string, conceptIDs []string, startingConceptID, errMsg string) error
}
