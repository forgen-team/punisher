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
	// Version: the lecture's version number (같은 목표 묶음의 몇 번째 판) — 다시 만든 판을 구분해 보인다.
	Version int `json:"version"`
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
	ExampleLanguage string `json:"exampleLanguage"`
	// DailyMinutes is the daily learning pace (15/30/45/60) used for the next-chapter prompt.
	DailyMinutes int `json:"dailyMinutes"`
	// MasteryTarget: the level (3~5) that counts as mastered.
	MasteryTarget int `json:"masteryTarget"`
	// ReviewIntervals: the review ladder in days (box 1·2·3 …).
	ReviewIntervals []int `json:"reviewIntervals"`
	// ReviewMinutes: daily review time (0~30).
	ReviewMinutes int `json:"reviewMinutes"`
	// ReviewGrader: default grader of reviews (self | llm).
	ReviewGrader string `json:"reviewGrader"`
	// PretestEnabled: prediction questions before a lecture (사전 진단).
	PretestEnabled bool `json:"pretestEnabled"`
	// LabOS: the reader's lab environment (macos | linux | none).
	LabOS string `json:"labOs"`
	// AdaptiveIntervals: per-objective interval scale (ease) on.
	AdaptiveIntervals bool `json:"adaptiveIntervals"`
	// Locale: screen and generation language (ko|en).
	Locale string `json:"locale"`
	// ReaderLevel: explanation density of generated text (beginner|novice|practitioner|advanced);
	// "" = not chosen yet (the web shows the first setup; generation uses practitioner).
	ReaderLevel string `json:"readerLevel"`
	// AreaFamiliarity: first-setup self-assessment per area (new|heard|can_explain); never nil ({}).
	AreaFamiliarity map[string]string `json:"areaFamiliarity"`
	// AIProvider: the AI used for every generation (claude | codex | agy; default claude).
	AIProvider string     `json:"aiProvider"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
	// LevelSuggestionDismissedAt: when the lower-level suggestion was last closed (nil = never; not sent).
	LevelSuggestionDismissedAt *time.Time `json:"-"`
	// ReaderLevelChangedAt: when ReaderLevel last changed (nil = never saved; not sent).
	ReaderLevelChangedAt *time.Time `json:"-"`
}

// CompletedItem is one entry of the "completed by area" history: a completed
// lecture or an answered inquiry thread.
type CompletedItem struct {
	Kind            string    `json:"kind"` // lecture | inquiry
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	ConceptID       string    `json:"conceptId,omitempty"`
	ConceptName     string    `json:"conceptName,omitempty"`
	Area            string    `json:"area"`
	Depth           int       `json:"depth,omitempty"`
	LectureKind     string    `json:"lectureKind,omitempty"`
	ParentLectureID string    `json:"parentLectureId,omitempty"`
	FollowupCount   int       `json:"followupCount,omitempty"`
	LastActivityAt  time.Time `json:"lastActivityAt"`
}

// CompletedArea is the number of completed items in one area.
type CompletedArea struct {
	Area  string `json:"area"`
	Count int    `json:"count"`
}
