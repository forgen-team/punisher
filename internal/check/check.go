// Package check holds the pure rules of mastery checks (design §4): the item
// set the LLM writes, the tutor's per-answer reply, the dialogue state machine
// and the leak / question / praise checks. No DB, no LLM.
package check

import (
	"encoding/json"
	"strings"
)

// Item purposes and formats (validated here, no DB CHECK — design §0.3).
const (
	PurposeCheck  = "check"
	PurposeReview = "review"

	FormatRecall      = "recall"
	FormatPredict     = "predict"
	FormatApply       = "apply"
	FormatIntegrative = "integrative"
)

// Tutor/learner moves (check_turns.move).
const (
	MoveAsk       = "ask"
	MoveNarrow    = "narrow"
	MoveExplain   = "explain"
	MoveReask     = "reask"
	MoveChallenge = "challenge"
	MoveWrap      = "wrap"
	MoveNext      = "next"
	MoveAnswer    = "answer"
	MoveReveal    = "reveal"
	MoveSelfGrade = "self_grade"
	MoveDispute   = "dispute"
)

// Rubric is how an item is graded (check_items.rubric).
type Rubric struct {
	Must           []string `json:"must"`
	Nice           []string `json:"nice"`
	Misconceptions []string `json:"misconceptions"`
	Answer         string   `json:"answer"`
	Anchor         string   `json:"anchor"`
	Covers         []string `json:"covers"`
}

// Item is one generated question (LLM output, then stored as a check_items row).
type Item struct {
	ObjectiveID string          `json:"objective_id"`
	Purpose     string          `json:"purpose"`
	Format      string          `json:"format"`
	Prompt      string          `json:"prompt"`
	Material    json.RawMessage `json:"material,omitempty"`
	Rubric      Rubric          `json:"rubric"`
}

// ItemSet is the LLM output of one item generation.
type ItemSet struct {
	Items []Item `json:"items"`
}

// Integrative reports whether the item covers the whole lecture (no single objective).
func (it Item) Integrative() bool { return it.ObjectiveID == "" }

// Stage is where the item's dialogue stands when an answer is graded.
type Stage struct {
	// Answers: learner answers on this item including the one being graded.
	Answers int
	// Challenged: a challenge (반론) was already asked — this answer defends it.
	Challenged  bool
	Integrative bool
	// Last: no item follows (the closing move is wrap instead of next).
	Last bool
	// Kind: the session kind (check | review). A tutored review allows one
	// narrowing question and no explanation (design §5 P2).
	Kind string
}

// MaxAnswers per item (design §4.2): first answer, after narrowing, after re-asking.
const MaxAnswers = 3

// MaxReviewAnswers per tutored review item: first answer and the one after narrowing.
const MaxReviewAnswers = 2

// MaxAnswersFor is the answer limit of an item in a session of this kind.
func MaxAnswersFor(kind string) int {
	if kind == PurposeReview {
		return MaxReviewAnswers
	}
	return MaxAnswers
}

func (st Stage) maxAnswers() int { return MaxAnswersFor(st.Kind) }

func (st Stage) closing() string {
	if st.Last {
		return MoveWrap
	}
	return MoveNext
}

// AllowedMoves are the tutor moves possible for this answer, sent to the LLM before the call.
func (st Stage) AllowedMoves() []string {
	end := st.closing()
	switch {
	case st.Challenged || st.Answers >= st.maxAnswers():
		return []string{end}
	case st.Answers == 1 && st.Integrative:
		return []string{MoveChallenge, MoveNarrow, end}
	case st.Answers == 1:
		return []string{MoveNarrow, end}
	default:
		return []string{MoveExplain, end}
	}
}

// ExpectedMove is the move the state machine requires for a verdict (held ends the item).
func (st Stage) ExpectedMove(verdict string) string {
	end := st.closing()
	if verdict == "held" || st.Challenged || st.Answers >= st.maxAnswers() {
		return end
	}
	correct := verdict == "correct"
	switch {
	case st.Answers == 1 && correct && st.Integrative:
		return MoveChallenge
	case correct:
		return end
	case st.Answers == 1:
		return MoveNarrow
	default:
		return MoveExplain
	}
}

// EndsItem reports whether a tutor move closes the item.
func EndsItem(move string) bool { return move == MoveNext || move == MoveWrap }

// leakChecked: moves that must not state the answer (design §4.2, N8).
var leakChecked = map[string]bool{MoveAsk: true, MoveNarrow: true, MoveReask: true, MoveChallenge: true}

// CountQuestions counts question marks (ASCII and full-width).
func CountQuestions(text string) int {
	return strings.Count(text, "?") + strings.Count(text, "？")
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
