package glossary

import (
	"testing"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
)

// TestRealGlossaryIsValid: the shipped dictionary matches the knowledge map —
// every concept has a gloss, terms do not shadow concept names, glosses fit.
func TestRealGlossaryIsValid(t *testing.T) {
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	g, err := Load(seedfiles.Glossary)
	if err != nil {
		t.Fatal(err)
	}
	ids, names := map[string]bool{}, map[string]string{}
	for _, c := range seed.Concepts {
		ids[c.ID] = true
		for _, n := range append([]string{c.Name}, c.Aliases...) {
			names[Normalize(n)] = c.ID
		}
	}
	for _, p := range g.Validate(ids, names) {
		t.Error(p)
	}
	for id := range ids {
		if g.ConceptGloss(id) == "" {
			t.Errorf("concept %s has no gloss", id)
		}
	}
	if n := len(g.IDs()); n < 100 {
		t.Errorf("only %d basic terms", n)
	}
}

// TestRealGlossaryEnglish: every concept gloss and term is translated.
func TestRealGlossaryEnglish(t *testing.T) {
	g, err := Load(seedfiles.Glossary)
	if err != nil {
		t.Fatal(err)
	}
	c, terms := g.EnglishCoverage()
	if c != len(g.concepts) || terms != len(g.terms) {
		t.Fatalf("English coverage %d/%d concepts, %d/%d terms", c, len(g.concepts), terms, len(g.terms))
	}
	tm, _ := g.TermIn("g.strace", "en")
	if tm.Name != "strace" || tm.Gloss == "" || tm.Gloss == g.terms["g.strace"].Gloss {
		t.Fatalf("English term not applied: %+v", tm)
	}
	if g.ConceptGlossIn("os.process", "en") == g.ConceptGloss("os.process") {
		t.Fatal("English concept gloss not applied")
	}
}
