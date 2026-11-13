package models

import (
	"encoding/json"
	"time"
)

// ObjectiveMastery is one objective's mastery as shown to the user (GET /v1/mastery).
type ObjectiveMastery struct {
	ObjectiveID string `json:"objectiveId"`
	ConceptID   string `json:"conceptId"`
	Text        string `json:"text"`
	Depth       int    `json:"depth"`
	Ordinal     int    `json:"ordinal"`
	// Level: 0 미확인 · 1 들어봄 · 2 따라감 · 3 혼자 답함 · 4 기억함 · 5 설명 가능.
	Level         int        `json:"level"`
	Box           int        `json:"box"`
	GradStep      int        `json:"gradStep"`
	Best          string     `json:"best"`
	IntegrativeOK bool       `json:"integrativeOk"`
	Lapses        int        `json:"lapses"`
	NextReviewAt  *time.Time `json:"nextReviewAt"`
	Stale         bool       `json:"stale"`
	Queued        bool       `json:"queued"`
	// LastGrader: who graded the last evidence (llm | self | llm_regrade; "" = none).
	LastGrader string     `json:"lastGrader"`
	UpdatedAt  *time.Time `json:"updatedAt"`
}

// ConceptMastery is the payload of GET /v1/mastery?conceptId=.
type ConceptMastery struct {
	ConceptMasterySummary
	Objectives []ObjectiveMastery `json:"objectives"`
}

// ConceptMasterySummary is a concept's level without the objectives (also the batch item).
type ConceptMasterySummary struct {
	ConceptID string `json:"conceptId"`
	// Level: the minimum level of the active objectives.
	Level         int  `json:"level"`
	Mastered      bool `json:"mastered"`
	MasteryTarget int  `json:"masteryTarget"`
	// UncheckedCount: active objectives without any evidence ("새 목표 n개 미확인").
	UncheckedCount int `json:"uncheckedCount"`
}

// CheckSession is a '확인하기' (or P2 review) session.
type CheckSession struct {
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	Grader     string          `json:"grader"`
	ConceptID  string          `json:"conceptId"`
	LectureID  string          `json:"lectureId"`
	SessionKey string          `json:"-"`
	ItemIDs    []string        `json:"itemIds"`
	Cursor     int             `json:"cursor"`
	Status     string          `json:"status"`
	Result     json.RawMessage `json:"result"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
	FinishedAt *time.Time      `json:"finishedAt,omitempty"`
}

// CheckGrade is the evidence of one judgement (check_turns.grade, without the restore snapshot).
type CheckGrade struct {
	Met           []string `json:"met"`
	Missing       []string `json:"missing"`
	Quote         string   `json:"quote"`
	Misconception string   `json:"misconception"`
	Confidence    float64  `json:"confidence"`
	// Choice: the option picked on a choice item (0-based; nil = not a choice answer).
	Choice *int `json:"choice,omitempty"`
	// Accepted / Conflict: dispute turns only.
	Accepted *bool `json:"accepted,omitempty"`
	Conflict bool  `json:"conflict,omitempty"`
}

// CheckTurn is one tutor or learner turn.
type CheckTurn struct {
	ID         string      `json:"id"`
	ItemID     string      `json:"itemId"`
	Seq        int         `json:"seq"`
	Role       string      `json:"role"`
	Move       string      `json:"move"`
	Text       string      `json:"text"`
	Status     string      `json:"status"`
	Verdict    string      `json:"verdict,omitempty"`
	Outcome    string      `json:"outcome,omitempty"`
	Aided      bool        `json:"aided"`
	Grader     string      `json:"grader,omitempty"`
	ReusedItem bool        `json:"reusedItem"`
	Grade      *CheckGrade `json:"grade,omitempty"`
	Model      string      `json:"model,omitempty"`
	Error      string      `json:"error,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
	// GradeRaw is the stored grade (with prev/appliedAt) for the service; never sent.
	GradeRaw json.RawMessage `json:"-"`
}

// CheckItemView is an item of a session as the learner sees it. Answer and
// Anchor (the model answer and where to re-read) appear once the item is done,
// or after the learner answered in a self-graded session.
type CheckItemView struct {
	ID          string          `json:"id"`
	ObjectiveID string          `json:"objectiveId"`
	Format      string          `json:"format"`
	Prompt      string          `json:"prompt"`
	Material    json.RawMessage `json:"material"`
	Done        bool            `json:"done"`
	Outcome     string          `json:"outcome,omitempty"`
	Answer      string          `json:"answer,omitempty"`
	Must        []string        `json:"must,omitempty"`
	Anchor      string          `json:"anchor,omitempty"`
	// LectureSummary: the item's lecture '한 장 요약' (markdown) once the item
	// is done, check/review sessions only; omitted when the lecture has none.
	LectureSummary string `json:"lectureSummary,omitempty"`
}

// CheckCurrent is what the learner can do now on the current item.
type CheckCurrent struct {
	ItemID string `json:"itemId"`
	// Answers: learner answers on this item so far (max 3).
	Answers      int  `json:"answers"`
	CanAnswer    bool `json:"canAnswer"`
	CanReveal    bool `json:"canReveal"`
	CanSelfGrade bool `json:"canSelfGrade"`
}

// CheckDetail is the payload of GET /v1/checks/{id} (and the POST responses).
type CheckDetail struct {
	Session CheckSession    `json:"session"`
	Items   []CheckItemView `json:"items"`
	Turns   []CheckTurn     `json:"turns"`
	// Current: nil when the session is not active.
	Current *CheckCurrent `json:"current"`
	// Pending: a tutor turn or the item generation is running (poll while true).
	Pending bool `json:"pending"`
	// Mastery: the lecture objectives' current mastery.
	Mastery []ObjectiveMastery `json:"mastery"`
	// Predictions: the learner's pretest answers on these objectives ('처음 예측', check sessions only).
	Predictions []Prediction `json:"predictions,omitempty"`
	// Placement: the chapter and the concepts answered right (placement sessions only).
	Placement *PlacementResult `json:"placement,omitempty"`
	// MisconceptionTurnIDs: turns of this session whose misconception the learner acknowledged.
	MisconceptionTurnIDs []string `json:"misconceptionTurnIds"`
}

// Prediction is the learner's pretest answer on an objective (latest first-seen per objective).
type Prediction struct {
	ObjectiveID string `json:"objectiveId"`
	Prompt      string `json:"prompt"`
	Answer      string `json:"answer"`
	// DontKnow: answered '모르겠어요'.
	DontKnow bool      `json:"dontKnow"`
	At       time.Time `json:"at"`
}

// PlacementResult is the outcome of a placement diagnostic: the concepts
// answered right are "이미 앎 후보" (known only once the learner confirms).
type PlacementResult struct {
	ChapterID  string               `json:"chapterId"`
	Candidates []PlacementCandidate `json:"candidates"`
}

// PlacementCandidate is a concept whose placement item was answered right.
type PlacementCandidate struct {
	ConceptID   string `json:"conceptId"`
	ConceptName string `json:"conceptName"`
}

// CheckHistoryItem is one '확인하기', review or '이미 앎' verification session in the history (GET /v1/checks/history).
type CheckHistoryItem struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Grader       string `json:"grader"`
	Status       string `json:"status"`
	ConceptID    string `json:"conceptId,omitempty"`
	ConceptName  string `json:"conceptName,omitempty"`
	LectureID    string `json:"lectureId,omitempty"`
	LectureTitle string `json:"lectureTitle,omitempty"`
	// ReviewDay: the KST date of a review session ("2026-11-13"; "" for checks).
	ReviewDay string `json:"reviewDay,omitempty"`
	ItemCount int    `json:"itemCount"`
	DoneCount int    `json:"doneCount"`
	// UnaidedCorrect: items that ended unaided_correct.
	UnaidedCorrect int        `json:"unaidedCorrect"`
	CreatedAt      time.Time  `json:"createdAt"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
}

// ReviewSessionRef is today's review session in the summary.
type ReviewSessionRef struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Grader string `json:"grader"`
	Total  int    `json:"total"`
	Done   int    `json:"done"`
}

// ReviewSummary is the payload of GET /v1/reviews/summary.
type ReviewSummary struct {
	Due              int    `json:"due"`
	Overdue          int    `json:"overdue"`
	Queued           int    `json:"queued"`
	EstimatedMinutes int    `json:"estimatedMinutes"`
	Backlogged       bool   `json:"backlogged"`
	Throughput       int    `json:"throughput"`
	Grader           string `json:"grader"`
	ReviewMinutes    int    `json:"reviewMinutes"`
	// Verify: known verifications mixed into today's session (bulk '이미 앎').
	Verify int `json:"verify"`
	// Today: today's latest review session (nil = none yet).
	Today *ReviewSessionRef `json:"today"`
}
