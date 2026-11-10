// Package models defines API/DB transfer types for punisher-server.
package models

import (
	"encoding/json"
	"time"
)

// Concept is a concept-graph node with the caller's learning state.
type Concept struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Area              string          `json:"area"`
	Depth             int             `json:"depth"`
	Summary           string          `json:"summary"`
	PracticalCases    json.RawMessage `json:"practicalCases"`
	DocLinks          json.RawMessage `json:"docLinks"`
	Status            string          `json:"status"`
	Origin            string          `json:"origin"`
	CurriculumVersion string          `json:"curriculumVersion,omitempty"`
	ObjectiveCount    int             `json:"objectiveCount"`
	UserState         string          `json:"userState"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	// ChapterID: the user's chapter this concept belongs to ("" = shared curriculum).
	ChapterID string `json:"chapterId,omitempty"`
}

// Objective is a learning objective with the caller's state at its depth.
type Objective struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Depth     int    `json:"depth"`
	Ordinal   int    `json:"ordinal"`
	UserState string `json:"userState"`
}

// ConceptLink is a neighbouring concept seen from a concept detail page.
type ConceptLink struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Area      string `json:"area"`
	Kind      string `json:"kind"`
	UserState string `json:"userState"`
}

// ConceptDetail is the payload of GET /v1/concepts/{id}.
type ConceptDetail struct {
	Concept
	Aliases       []string      `json:"aliases"`
	Objectives    []Objective   `json:"objectives"`
	Prerequisites []ConceptLink `json:"prerequisites"`
	Dependents    []ConceptLink `json:"dependents"`
	Related       []ConceptLink `json:"related"`
	Locked        bool          `json:"locked"`
}

// Edge is a concept-graph edge (prerequisite: From before To).
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

// GraphNode is a node of the knowledge map.
type GraphNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Area     string `json:"area"`
	Depth    int    `json:"depth"`
	Status   string `json:"status"`
	State    string `json:"state"`
	Locked   bool   `json:"locked"`
	Frontier bool   `json:"frontier"`
	Unlocks  int    `json:"unlocks"`
	// Requests are the caller's pending study requests resolved to this concept
	// (only while the concept is not completed/known).
	Requests []MapRequest `json:"requests,omitempty"`
	// Inquiries: the caller's answered questions ('이 질문에 답하기') about this concept
	// (kept after the concept is learned — 나중에 개념을 볼 때 내 질문도 함께).
	Inquiries []MapInquiry `json:"inquiries,omitempty"`
	// ChapterID: the caller's chapter this concept belongs to (map grouping).
	ChapterID string `json:"chapterId,omitempty"`
}

// MapRequest is a pending study request shown on the knowledge map.
type MapRequest struct {
	ID         string   `json:"id"`
	Query      string   `json:"query"`
	Source     string   `json:"source"`
	Status     string   `json:"status"`
	ConceptIDs []string `json:"-"`
}

// MapInquiry is an answered question shown on the concepts it is about.
type MapInquiry struct {
	ID         string   `json:"id"`
	Question   string   `json:"question"`
	ConceptIDs []string `json:"-"`
}

// AreaCoverage summarises progress per area.
type AreaCoverage struct {
	Area      string `json:"area"`
	Total     int    `json:"total"`
	Completed int    `json:"completed"`
	Known     int    `json:"known"`
	// InProgress / Requested / Dismissed: per-area learning signals (strength/weakness basis).
	InProgress int `json:"inProgress"`
	Requested  int `json:"requested"`
	Dismissed  int `json:"dismissed"`
}

// Graph is the payload of GET /v1/graph.
type Graph struct {
	CurriculumVersion string         `json:"curriculumVersion"`
	Nodes             []GraphNode    `json:"nodes"`
	Edges             []Edge         `json:"edges"`
	Coverage          []AreaCoverage `json:"coverage"`
	// PendingRequests: study requests not yet mapped to any concept (분류 중).
	PendingRequests []MapRequest `json:"pendingRequests"`
	// Chapters: the caller's own chapters (새 챕터: <이름> 묶음).
	Chapters []GraphChapter `json:"chapters"`
}

// GraphChapter is one of the caller's chapters on the map.
type GraphChapter struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// FrontierItem is a recommended next concept.
type FrontierItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Area    string `json:"area"`
	Depth   int    `json:"depth"`
	Summary string `json:"summary"`
	State   string `json:"state"`
	Unlocks int    `json:"unlocks"`
}

// GraphData is the raw per-user graph loaded from the store.
type GraphData struct {
	CurriculumVersion string
	Concepts          []Concept // active concepts only
	Edges             []Edge    // all kinds, between active concepts
	States            map[string]string
	// Map extras (GET /v1/graph only; see store.LoadMapExtras).
	Requests  []MapRequest
	Inquiries []MapInquiry
	Dismissed map[string]bool
}

// StudyRequest is a request to study something (from RSS, manual input, Guardian).
type StudyRequest struct {
	ID                 string          `json:"id"`
	UserID             string          `json:"-"`
	Source             string          `json:"source"`
	SourceRef          string          `json:"sourceRef"`
	Query              string          `json:"query"`
	Context            json.RawMessage `json:"context"`
	ResolvedConceptIDs []string        `json:"resolvedConceptIds"`
	Status             string          `json:"status"`
	IdempotencyKey     string          `json:"idempotencyKey"`
	StartingConceptID  string          `json:"startingConceptId,omitempty"`
	ResolveAttempts    int             `json:"resolveAttempts"`
	ResolveError       string          `json:"resolveError,omitempty"`
	ResolvedAt         *time.Time      `json:"resolvedAt,omitempty"`
	// Learned is computed: every resolved concept is completed/known.
	Learned bool `json:"learned"`
	// Resolving is computed: concept resolution for this request is running right now.
	Resolving bool      `json:"resolving"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
