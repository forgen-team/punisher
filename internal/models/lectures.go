package models

import (
	"encoding/json"
	"time"
)

// Lecture is an immutable, versioned lecture record.
type Lecture struct {
	ID               string           `json:"id"`
	ConceptID        string           `json:"conceptId"`
	ObjectiveIDs     []string         `json:"objectiveIds"`
	ObjectiveKey     string           `json:"objectiveKey"`
	Depth            int              `json:"depth"`
	Kind             string           `json:"kind"`
	Status           string           `json:"status"`
	Version          int              `json:"version"`
	Title            string           `json:"title"`
	Content          json.RawMessage  `json:"content,omitempty"`
	ContentHash      string           `json:"contentHash"`
	Sources          json.RawMessage  `json:"sources"`
	Model            string           `json:"model"`
	PromptVersion    string           `json:"promptVersion"`
	RequestedBy      string           `json:"-"`
	Flagged          bool             `json:"flagged"`
	FlagReason       string           `json:"flagReason,omitempty"`
	EstimatedMinutes int              `json:"estimatedMinutes"`
	GenerationJobID  string           `json:"generationJobId,omitempty"`
	CreatedAt        time.Time        `json:"createdAt"`
	PublishedAt      *time.Time       `json:"publishedAt,omitempty"`
	Progress         *LectureProgress `json:"progress,omitempty"`
	Reviews          []LectureReview  `json:"reviews,omitempty"`
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
