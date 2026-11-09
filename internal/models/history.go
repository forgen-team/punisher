package models

import "time"

// MyQuestion is one of the caller's questions with its lecture/concept context.
type MyQuestion struct {
	Question
	LectureTitle string `json:"lectureTitle"`
	ConceptName  string `json:"conceptName"`
}

// MyLecture is a lecture the caller has started or completed.
type MyLecture struct {
	ID               string     `json:"id"`
	ConceptID        string     `json:"conceptId"`
	ConceptName      string     `json:"conceptName"`
	Title            string     `json:"title"`
	Depth            int        `json:"depth"`
	Kind             string     `json:"kind"`
	EstimatedMinutes int        `json:"estimatedMinutes"`
	Percent          int        `json:"percent"`
	StartedAt        time.Time  `json:"startedAt"`
	LastSeenAt       time.Time  `json:"lastSeenAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
	// ParentLectureID: the lecture this one was opened from via "더 깊게" ("" = none).
	ParentLectureID string `json:"parentLectureId,omitempty"`
}

// LinkedLecture is a lecture connected by "이 개념 더 깊게" (parent or child).
type LinkedLecture struct {
	ID          string     `json:"id"`
	ConceptID   string     `json:"conceptId"`
	ConceptName string     `json:"conceptName"`
	Title       string     `json:"title"`
	Kind        string     `json:"kind"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	LinkedAt    time.Time  `json:"linkedAt"`
	Anchor      string     `json:"anchor"`
	Percent     int        `json:"percent"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	// LinkKind: deeper ("이 개념 더 깊게") | regenerated ("새 프롬프트로 다시 만들기").
	LinkKind      string `json:"linkKind"`
	PromptVersion string `json:"promptVersion"`
}

// ConceptLecture is a readable lecture of a concept with the caller's progress.
type ConceptLecture struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Depth            int        `json:"depth"`
	Kind             string     `json:"kind"`
	Version          int        `json:"version"`
	EstimatedMinutes int        `json:"estimatedMinutes"`
	PublishedAt      *time.Time `json:"publishedAt,omitempty"`
	// ProgressState is not_started | in_progress | completed.
	ProgressState string `json:"progressState"`
	Percent       int    `json:"percent"`
}

// LearningSettings are the caller's learning preferences.
type LearningSettings struct {
	// ExampleLanguage is the language for code examples in generated lectures/answers.
	ExampleLanguage string     `json:"exampleLanguage"`
	UpdatedAt       *time.Time `json:"updatedAt,omitempty"`
}
