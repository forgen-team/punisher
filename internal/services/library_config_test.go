package services

import (
	"context"
	"testing"
)

// PUNISHER_LIBRARY_AHEAD=0 must disable ahead pre-generation (it used to be
// silently treated as 2), and study pre-generation has its own switch. Nil
// Store/Keys prove no work is attempted.
func TestLibraryPregenerationSwitches(t *testing.T) {
	s := &Service{Cfg: Config{OwnerUserID: "owner", LibraryAhead: 0, StudyPregenerate: false}}
	if rep, err := s.GenerateLibraryAhead(context.Background()); err != nil || rep.Ready != 0 || len(rep.Generated) != 0 {
		t.Fatalf("library ahead=0 must be a no-op: %+v %v", rep, err)
	}
	if rep, err := s.PregenerateStudyStarts(context.Background(), []string{"c1"}); err != nil || len(rep.Generated) != 0 {
		t.Fatalf("study pregenerate disabled must be a no-op: %+v %v", rep, err)
	}
	s.Cfg.StudyPregenerate = true
	if rep, err := s.PregenerateStudyStarts(context.Background(), nil); err != nil || len(rep.Generated) != 0 {
		t.Fatalf("no starts must be a no-op: %+v %v", rep, err)
	}
}
