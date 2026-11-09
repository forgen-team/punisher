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
