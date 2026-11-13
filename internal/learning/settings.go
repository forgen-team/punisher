package learning

// DefaultExampleLanguage is used when the user has not chosen one.
const DefaultExampleLanguage = "typescript"

// ExampleLanguages maps the allowed example-code language ids to the display
// name used in prompts. Order matters only for ExampleLanguageIDs.
var ExampleLanguages = map[string]string{
	"typescript": "TypeScript",
	"javascript": "JavaScript",
	"go":         "Go",
	"java":       "Java",
	"kotlin":     "Kotlin",
	"python":     "Python",
}

// ExampleLanguageIDs is the display order of ExampleLanguages.
var ExampleLanguageIDs = []string{"typescript", "javascript", "go", "java", "kotlin", "python"}

// IsValidExampleLanguage reports whether id is an allowed example language.
func IsValidExampleLanguage(id string) bool {
	_, ok := ExampleLanguages[id]
	return ok
}

// NormalizeExampleLanguage returns id when valid, else the default.
func NormalizeExampleLanguage(id string) string {
	if IsValidExampleLanguage(id) {
		return id
	}
	return DefaultExampleLanguage
}

// Mastery/review defaults (design §9; changed in 설정 > 학습).
const (
	DefaultMasteryTarget = LevelRemember
	DefaultReviewMinutes = 10
	DefaultReviewGrader  = GraderSelf
)

// DefaultReviewIntervals is the review ladder in days (box 1·2·3).
var DefaultReviewIntervals = []int{1, 7, 30}

// Graders of an answer. A session's grader is fixed when it is created.
const (
	GraderLLM     = "llm"
	GraderSelf    = "self"
	GraderRegrade = "llm_regrade"
	// GraderNone: a session that only records answers (pretest, placement).
	GraderNone = "none"
)

// DefaultPretestEnabled: prediction questions before a lecture are on by default (design §5 P3).
const DefaultPretestEnabled = true

// Mastery setting bounds (design §3.3).
const (
	MinMasteryTarget  = 3
	MaxMasteryTarget  = 5
	MaxReviewMinutes  = 30
	MaxReviewInterval = 6 // number of ladder steps
)

// ValidMasteryTarget: the level that counts as "mastered" (3~5).
func ValidMasteryTarget(n int) bool { return n >= MinMasteryTarget && n <= MaxMasteryTarget }

// ValidReviewMinutes: daily review time, 0 (off) ~ 30.
func ValidReviewMinutes(n int) bool { return n >= 0 && n <= MaxReviewMinutes }

// ValidReviewGrader: who grades reviews by default (self | llm).
func ValidReviewGrader(g string) bool { return g == GraderSelf || g == GraderLLM }

// ValidReviewIntervals: 1~6 positive, strictly increasing day counts.
func ValidReviewIntervals(days []int) bool {
	if len(days) == 0 || len(days) > MaxReviewInterval {
		return false
	}
	for i, d := range days {
		if d <= 0 || (i > 0 && d <= days[i-1]) {
			return false
		}
	}
	return true
}
