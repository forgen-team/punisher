package models

import "time"

// Misconception is a misconception the learner acknowledged on a check result
// (user_misconceptions, design §5 P4). Text is the turn's grade.misconception.
type Misconception struct {
	ID          string `json:"id"`
	ConceptID   string `json:"conceptId"`
	ConceptName string `json:"conceptName"`
	// ObjectiveID: "" for an integrative item.
	ObjectiveID   string     `json:"objectiveId"`
	ObjectiveText string     `json:"objectiveText"`
	Text          string     `json:"text"`
	TurnID        string     `json:"turnId"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"createdAt"`
	ResolvedAt    *time.Time `json:"resolvedAt,omitempty"`
}
