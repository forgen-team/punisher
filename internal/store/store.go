package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/wooo-jin/punisher/internal/chapter"
	"github.com/wooo-jin/punisher/internal/check"
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
	ResolveConcepts(ctx context.Context, userID, normalizedQuery string) ([]string, error)
	ConceptInfo(ctx context.Context, userID, conceptID string) (models.ConceptInfo, error)
	DoneSet(ctx context.Context, userID, conceptID string) (learning.DoneSet, error)
	ConceptCatalog(ctx context.Context, userID string) ([]ConceptRef, error)
	RelatedConcepts(ctx context.Context, userID, conceptID string) ([]ConceptRef, error)
	PrerequisitesOf(ctx context.Context, userID, conceptID string) ([]ConceptRef, error)
	KnownConceptIDs(ctx context.Context, userID string) (map[string]bool, error)
	ProposeConcept(ctx context.Context, userID string, p models.ProposedConcept) (string, error)
	GetGloss(ctx context.Context, conceptID string) (string, error)
	SaveGloss(ctx context.Context, conceptID, gloss, model string) error
	LoadToday(ctx context.Context, userID string) (*TodayData, error)
	PrerequisiteEdges(ctx context.Context, userID string) ([]learning.PrereqEdge, error)
	AliasIndex(ctx context.Context, userID string) (learning.AliasIndex, error)

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
	SaveLearningSettings(ctx context.Context, userID, lang string, daily int) (models.LearningSettings, error)
	UpdateLearningSettings(ctx context.Context, userID string, p LearningSettingsPatch) (models.LearningSettings, error)
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
	UsageSinceChecks(ctx context.Context, userID string, since time.Time) (tokens int, err error)
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
	// inquiries (이 질문에 답하기, 이어서 묻기)
	CreateInquiry(ctx context.Context, userID, question string, dedupeWindow time.Duration, maxPending int) (models.Inquiry, bool, error)
	CountPendingInquiries(ctx context.Context, userID string) (int, error)
	GetInquiry(ctx context.Context, userID, id string) (models.Inquiry, error)
	ListInquiries(ctx context.Context, userID string, f InquiryFilter) ([]models.Inquiry, error)
	SetInquiryPending(ctx context.Context, userID, id string) (models.Inquiry, bool, error)
	AnswerInquiry(ctx context.Context, userID, id string, a InquiryAnswer) error
	FailInquiry(ctx context.Context, userID, id, model, errMsg string, attempts []AttemptRecord) error
	SetInquiryRead(ctx context.Context, userID, id string, read bool) error
	SweepStaleInquiries(ctx context.Context, olderThan time.Duration) (int64, error)
	InquiryConcepts(ctx context.Context, userID string, ids []string) ([]models.ConceptLink, error)
	ListFollowups(ctx context.Context, userID, inquiryID string) ([]models.InquiryFollowup, error)
	CreateFollowup(ctx context.Context, userID, inquiryID, question, quote string) (models.InquiryFollowup, error)
	SetFollowupPending(ctx context.Context, userID, inquiryID, id string) (models.InquiryFollowup, error)
	AnswerFollowup(ctx context.Context, userID, inquiryID, id string, a FollowupAnswer) error
	FailFollowup(ctx context.Context, userID, id, model, errMsg string) error
	ListCompleted(ctx context.Context, userID string, f CompletedFilter, cursor string) ([]models.CompletedItem, string, error)
	CompletedAreas(ctx context.Context, userID, query string) ([]models.CompletedArea, error)

	CreateStudyRequest(ctx context.Context, req models.StudyRequest) (models.StudyRequest, bool, error)
	ListStudyRequests(ctx context.Context, userID, status string, limit int) ([]models.StudyRequest, error)
	GetStudyRequest(ctx context.Context, userID, id string) (models.StudyRequest, error)
	UnresolvedStudyRequests(ctx context.Context, userID string, maxAttempts, limit int) ([]models.StudyRequest, error)
	RecordResolution(ctx context.Context, id string, conceptIDs []string, startingConceptID, errMsg string) error
	SetStudyRequestDismissed(ctx context.Context, userID, id string, dismissed bool) error

	// lecture links ("더 깊게" / "다시 만들기")
	LinkDeeper(ctx context.Context, l DeeperLink) error
	ResolveLinkJob(ctx context.Context, jobID, lectureID string) error
	LectureLinks(ctx context.Context, userID, lectureID string) (parents, children []models.LinkedLecture, err error)

	// prompt overrides
	PromptOverrides(ctx context.Context, userID string) (map[string]PromptOverride, error)
	SavePromptOverride(ctx context.Context, userID, promptID, body, action string) (PromptOverride, error)
	ResetPromptOverride(ctx context.Context, userID, promptID string) error
	PromptHistory(ctx context.Context, userID, promptID string) ([]PromptHistoryEntry, error)
	PromptHistoryEntryByID(ctx context.Context, userID string, id int64) (PromptHistoryEntry, error)
	UserGloss(ctx context.Context, userID, conceptID, promptHash string) (string, error)
	SaveUserGloss(ctx context.Context, userID, conceptID, promptHash, gloss, model string) error

	// chapters ("다음 챕터")
	ChapterSignals(ctx context.Context, userID string) (proposed []learning.ProposedSignal, deepen []learning.DeepenSignal, covered map[string]bool, err error)
	ChapterContext(ctx context.Context, userID string) (chapter.Context, error)
	SaveChapterDraft(ctx context.Context, userID string, input any, d chapter.Draft, model string) (string, error)
	ChapterDraft(ctx context.Context, userID, draftID string) (d chapter.Draft, input json.RawMessage, applied bool, err error)
	ApplyChapter(ctx context.Context, userID, draftID string, d chapter.Draft, input json.RawMessage) (string, error)
	ListChapters(ctx context.Context, userID string) ([]ChapterInfo, error)

	// mastery checks ("확인하기")
	CreateCheckSession(ctx context.Context, n NewCheckSession) (models.CheckSession, bool, error)
	GetCheckSession(ctx context.Context, userID, id string) (models.CheckSession, error)
	ActivateCheckSession(ctx context.Context, userID, id string, itemIDs []string, ask TurnWrite) (models.CheckSession, error)
	SetCheckSessionStatus(ctx context.Context, userID, id string, from []string, to, errMsg string) error
	SweepStaleChecks(ctx context.Context, turnAge, prepareAge, abandonAge time.Duration) (int64, error)
	CountPendingChecks(ctx context.Context, userID string) (int, error)
	ListCheckTurns(ctx context.Context, userID, sessionID string) ([]models.CheckTurn, error)
	StartCheckTurns(ctx context.Context, userID, sessionID string, learner *TurnWrite, tutor *TurnWrite, maxPending int, requireActive bool) (models.CheckTurn, error)
	CompleteCheckTurn(ctx context.Context, t TurnWrite) error
	FailCheckTurn(ctx context.Context, userID, turnID, model, errMsg string) error
	ActiveCheckItems(ctx context.Context, userID string, key CheckItemKey) ([]CheckItem, error)
	CheckItemsByID(ctx context.Context, userID string, ids []string) ([]CheckItem, error)
	SaveCheckItems(ctx context.Context, key CheckItemKey, conceptID, jobID, model, promptVersion string, items []check.Item) ([]CheckItem, error)
	StartCheckItemsJob(ctx context.Context, userID, conceptID string, key CheckItemKey, maxPending int) (models.GenerationJob, bool, error)
	FinishCheckItem(ctx context.Context, f ItemFinish) (string, error)
	ResolveDispute(ctx context.Context, d DisputeResolve) (bool, error)
	ConceptMastery(ctx context.Context, userID, conceptID string) ([]models.ObjectiveMastery, error)
	ConceptsMastery(ctx context.Context, userID string, conceptIDs []string) ([]models.ObjectiveMastery, error)
	ObjectivesMastery(ctx context.Context, userID string, objectiveIDs []string) ([]models.ObjectiveMastery, error)
	ReopenObjective(ctx context.Context, userID, objectiveID string) (string, error)

	// spaced review ("복습")
	ReviewQueue(ctx context.Context, userID string, now time.Time) ([]learning.ReviewCandidate, error)
	ReviewCounts(ctx context.Context, userID string, now time.Time) (learning.ReviewCounts, error)
	ReleaseQueuedReviews(ctx context.Context, userID string, throughput int, now time.Time, intervals []int) (int, error)
	ReviewItems(ctx context.Context, userID string, objectiveIDs []string) (map[string]ReviewPick, error)
	ItemSeenBefore(ctx context.Context, userID, itemID, sessionID string) (bool, error)
	LatestCheckSession(ctx context.Context, userID, sessionKey string) (models.CheckSession, error)
	AbandonStaleReviews(ctx context.Context, userID, todayKey string) (int64, error)
	ListCheckHistory(ctx context.Context, userID, kind string, limit int, cursor string) ([]models.CheckHistoryItem, string, error)
	RetentionSamples(ctx context.Context, userID string, since time.Time) ([]learning.RetentionSample, error)
	ReviewVariantTargets(ctx context.Context, userID string, until time.Time, limit int) ([]VariantTarget, error)
	SaveVariantItems(ctx context.Context, key CheckItemKey, conceptID, jobID, model, promptVersion string, items []check.Item) ([]CheckItem, error)
}
