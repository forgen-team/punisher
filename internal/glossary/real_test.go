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

// TestDeeperConceptLinks: terms that now have a dedicated depth-3 concept open
// that concept from '이 개념 더 깊게' — in both locales (the English overlay
// keeps the base term's concept).
func TestDeeperConceptLinks(t *testing.T) {
	g, err := Load(seedfiles.Glossary)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"g.lru": "ds.cache-eviction-policies", "g.cache-eviction": "ds.cache-eviction-policies",
		"g.tlb": "arch.tlb-huge-pages", "g.mmu-tlb": "arch.tlb-huge-pages",
		"g.simd": "arch.simd-vectorization",
		"g.mmap": "os.mmap-page-fault", "g.page-fault": "os.mmap-page-fault",
		"g.hpa": "infra.autoscaling-feedback-loop", "g.sidecar": "infra.service-mesh-sidecar",
		"g.cpu-throttling": "infra.cfs-cpu-throttling", "g.eviction": "infra.scheduling-preemption-eviction",
		"g.graceful-shutdown": "infra.graceful-shutdown-draining", "g.prestop-hook": "infra.graceful-shutdown-draining",
		"g.connection-draining": "infra.graceful-shutdown-draining",
		"g.congestion-control":  "net.tcp-congestion-control", "g.hol-blocking": "net.hol-blocking-quic",
		"g.listen-backlog": "net.tcp-accept-backlog", "g.ephemeral-port": "net.time-wait-port-exhaustion",
		"g.autovacuum": "db.vacuum-bloat", "g.dead-tuple": "db.vacuum-bloat", "g.table-bloat": "db.vacuum-bloat",
		"g.planner-statistics": "db.query-optimizer-statistics",
	}
	for id, concept := range want {
		for _, loc := range []string{"ko", "en"} {
			term, ok := g.TermIn(id, loc)
			if !ok || term.Concept != concept {
				t.Errorf("%s (%s): concept %q, want %q", id, loc, term.Concept, concept)
			}
		}
	}
}
