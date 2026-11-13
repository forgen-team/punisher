package learning

// Misconception statuses (user_misconceptions.status, design §5 P4 — validated here, no DB CHECK).
const (
	MisconceptionOpen     = "open"
	MisconceptionResolved = "resolved"
)

// MaxPromptMisconceptions: acknowledged misconceptions given to one prompt (related concept and its prerequisites).
const MaxPromptMisconceptions = 5

// MaxMisconceptionRunes bounds one misconception's text, stored and in prompts.
const MaxMisconceptionRunes = 300

// RecheckLapses: an objective that lapsed this often without recovering gets a "다시 확인" card (design §5 P4).
const RecheckLapses = 2

// ValidMisconceptionStatus reports whether s is a misconception status.
func ValidMisconceptionStatus(s string) bool {
	return s == MisconceptionOpen || s == MisconceptionResolved
}

// ObjectiveLevel is one active objective's evidence, the input of WeakConcept.
type ObjectiveLevel struct {
	Level  int
	Best   string
	Lapses int
}

// WeakConcept (design §5 P4): the concept has check evidence (some best ≠ none)
// and its level (minimum over active objectives) is below "혼자 답함", or an
// objective lapsed without recovering (lapses reset when the box is 2 again).
// A completed concept never checked (all none) is not weak.
func WeakConcept(objectives []ObjectiveLevel) bool {
	evidence, lapsed := false, false
	levels := make([]int, 0, len(objectives))
	for _, o := range objectives {
		levels = append(levels, o.Level)
		evidence = evidence || (o.Best != "" && o.Best != BestNone)
		lapsed = lapsed || o.Lapses >= 1
	}
	return evidence && (ConceptLevel(levels) < LevelAlone || lapsed)
}
