package models

import (
	"encoding/json"
	"time"
)

// Lecture is an immutable, versioned lecture record.
type Lecture struct {
	ID            string          `json:"id"`
	ConceptID     string          `json:"conceptId"`
	ObjectiveIDs  []string        `json:"objectiveIds"`
	ObjectiveKey  string          `json:"objectiveKey"`
	Depth         int             `json:"depth"`
	Kind          string          `json:"kind"`
	Status        string          `json:"status"`
	Version       int             `json:"version"`
	Title         string          `json:"title"`
	Content       json.RawMessage `json:"content,omitempty"`
	ContentHash   string          `json:"contentHash"`
	Sources       json.RawMessage `json:"sources"`
	Model         string          `json:"model"`
	PromptVersion string          `json:"promptVersion"`
	RequestedBy   string          `json:"-"`
	Flagged       bool            `json:"flagged"`
	FlagReason    string          `json:"flagReason"`
	// FlagReasonCodes: codes of FlagReason for the web to translate (internal/i18n; omitted for LLM-written or legacy text).
	FlagReasonCodes []string `json:"flagReasonCodes,omitempty"`
	// ReviewState: unreviewed (no pass review yet) | passed | flagged (a review found problems).
	ReviewState string `json:"reviewState"`
	// Reviewed: a pass review exists (check items are only written for reviewed lectures).
	Reviewed         bool   `json:"-"`
	EstimatedMinutes int    `json:"estimatedMinutes"`
	GenerationJobID  string `json:"generationJobId,omitempty"`
	// Language: ko|en — the lecture's language (i18n §1).
	Language string `json:"language"`
	// ReaderLevel: the reader level the lecture was written for (learner-level ①; older lectures = practitioner).
	ReaderLevel string `json:"readerLevel"`
	// Provider: the AI that wrote it (claude | codex | agy; older lectures = claude).
	Provider    string           `json:"provider"`
	CreatedAt   time.Time        `json:"createdAt"`
	PublishedAt *time.Time       `json:"publishedAt,omitempty"`
	Progress    *LectureProgress `json:"progress,omitempty"`
	Reviews     []LectureReview  `json:"reviews,omitempty"`
}

// Lecture review states (quality design v2.2 §2.2).
const (
	ReviewUnreviewed = "unreviewed"
	ReviewPassed     = "passed"
	ReviewFlagged    = "flagged"
)

// LectureReviewState: flagged wins (a later review found problems), then a pass record.
func LectureReviewState(reviewed, flagged bool) string {
	switch {
	case flagged:
		return ReviewFlagged
	case reviewed:
		return ReviewPassed
	}
	return ReviewUnreviewed
}

// LectureProgress is the caller's reading position.
type LectureProgress struct {
	Anchor      string     `json:"anchor"`
	Percent     int        `json:"percent"`
	StartedAt   time.Time  `json:"startedAt"`
	LastSeenAt  time.Time  `json:"lastSeenAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// LectureReview is one review-pass result.
type LectureReview struct {
	ID               string          `json:"id"`
	Verdict          string          `json:"verdict"`
	Summary          string          `json:"summary"`
	Issues           json.RawMessage `json:"issues"`
	ValidationIssues json.RawMessage `json:"validationIssues"`
	// ReaderGaps: the reviewer's beginner-reader pass (sentences assuming undefined terms).
	ReaderGaps json.RawMessage `json:"readerGaps,omitempty"`
	Attempt    int             `json:"attempt"`
	Model      string          `json:"model"`
	Reviewer   string          `json:"reviewer"`
	CreatedAt  time.Time       `json:"createdAt"`
	// Quality: the stored rubric of this review (machine rules, LLM scores,
	// independent answers, links); omitted when recorded before it existed.
	Quality json.RawMessage `json:"quality,omitempty"`
}

// GenerationJob is an LLM job (lecture generation or usage accounting).
type GenerationJob struct {
	ID             string          `json:"id"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Kind           string          `json:"kind"`
	UserID         string          `json:"-"`
	ConceptID      string          `json:"conceptId"`
	ObjectiveKey   string          `json:"objectiveKey"`
	Depth          int             `json:"depth"`
	Status         string          `json:"status"`
	Reason         string          `json:"reason"`
	Error          string          `json:"error,omitempty"`
	Attempts       int             `json:"attempts"`
	LectureID      string          `json:"lectureId,omitempty"`
	Model          string          `json:"model"`
	InputTokens    int             `json:"inputTokens"`
	OutputTokens   int             `json:"outputTokens"`
	Input          json.RawMessage `json:"input"`
	Language       string          `json:"language"`
	ReaderLevel    string          `json:"readerLevel"`
	StartedAt      *time.Time      `json:"startedAt,omitempty"`
	FinishedAt     *time.Time      `json:"finishedAt,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
}

// Question is a lecture Q&A entry (or an alternative explanation request).
type Question struct {
	ID               string          `json:"id"`
	LectureID        string          `json:"lectureId,omitempty"`
	ConceptID        string          `json:"conceptId,omitempty"`
	Anchor           string          `json:"anchor"`
	Kind             string          `json:"kind"`
	Question         string          `json:"question"`
	Answer           string          `json:"answer"`
	Status           string          `json:"status"`
	Model            string          `json:"model"`
	Error            string          `json:"error,omitempty"`
	LinkedConceptIDs []string        `json:"linkedConceptIds"`
	ProposedConcepts json.RawMessage `json:"proposedConcepts"`
	CreatedAt        time.Time       `json:"createdAt"`
	AnsweredAt       *time.Time      `json:"answeredAt,omitempty"`
	// Quote: the passage a reading thread is about (roots; "" for follow-ups and plain questions).
	Quote string `json:"quote"`
	// ParentID: the thread's root question for a follow-up ("" = a root).
	ParentID string `json:"parentId"`
	// Mode: easier | deeper | question (learner-level ④).
	Mode string `json:"mode"`
}

// Note is an anchor-based note or highlight.
type Note struct {
	ID        string    `json:"id"`
	LectureID string    `json:"lectureId,omitempty"`
	ConceptID string    `json:"conceptId,omitempty"`
	Anchor    string    `json:"anchor"`
	Kind      string    `json:"kind"`
	Quote     string    `json:"quote"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Goal is a user learning goal (target concepts → prerequisite path).
type Goal struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	TargetConceptIDs []string  `json:"targetConceptIds"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// PathStep is one concept on a goal path with the caller's state.
type PathStep struct {
	ConceptID string `json:"conceptId"`
	Name      string `json:"name"`
	Area      string `json:"area"`
	Depth     int    `json:"depth"`
	State     string `json:"state"`
	Locked    bool   `json:"locked"`
	IsTarget  bool   `json:"isTarget"`
	IsNext    bool   `json:"isNext"`
	// Dismissed: the caller marked it '필요 없음'; kept on the path (it still gates the next steps).
	Dismissed bool `json:"dismissed,omitempty"`
}

// ConceptInfo is a concept with its active objectives, for generation.
type ConceptInfo struct {
	ID             string
	Name           string
	Area           string
	Depth          int
	Summary        string
	Status         string
	PracticalCases []string
	DocLinks       []DocLink
	Objectives     []ObjectiveInfo
}

// ObjectiveInfo is an active objective.
type ObjectiveInfo struct {
	ID      string
	Text    string
	Depth   int
	Ordinal int
}

// DocLink is an official documentation link.
type DocLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ProposedConcept is a dynamically proposed graph node.
type ProposedConcept struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Area          string   `json:"area"`
	Depth         int      `json:"depth"`
	Summary       string   `json:"summary"`
	Aliases       []string `json:"aliases"`
	Prerequisites []string `json:"prerequisites"`
	Objectives    []string `json:"objectives"`
}

// Inquiry is a free-form question answered as a lecture-style explanation
// ("이 질문에 답하기"). Content uses the lecture section/block structure.
type Inquiry struct {
	ID                 string          `json:"id"`
	Question           string          `json:"question"`
	Status             string          `json:"status"` // pending | answered | failed
	Content            json.RawMessage `json:"content,omitempty"`
	ConceptIDs         []string        `json:"conceptIds"`
	ProposedConceptIDs []string        `json:"proposedConceptIds"`
	Concepts           []ConceptLink   `json:"concepts"`
	Model              string          `json:"model"`
	PromptVersion      string          `json:"promptVersion"`
	Error              string          `json:"error,omitempty"`
	ReadAt             *time.Time      `json:"readAt,omitempty"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
	AnsweredAt         *time.Time      `json:"answeredAt,omitempty"`
	// Followups: 이어서 묻기(스레드 순서대로, 상세 응답에서만). FollowupCount 는 목록에서도 채워진다.
	Followups     []InquiryFollowup `json:"followups,omitempty"`
	FollowupCount int               `json:"followupCount"`
}

// InquiryFollowup is one follow-up question and its answer in an inquiry thread.
type InquiryFollowup struct {
	ID         string          `json:"id"`
	Question   string          `json:"question"`
	Quote      string          `json:"quote,omitempty"`
	Status     string          `json:"status"` // pending | answered | failed
	Section    json.RawMessage `json:"section,omitempty"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
	AnsweredAt *time.Time      `json:"answeredAt,omitempty"`
}
