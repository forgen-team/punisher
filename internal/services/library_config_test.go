package services

import (
	"context"
	"testing"
	"time"
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

// 새벽 창 안에서는 NightAhead(더 클 때)만큼, 밖에서는 LibraryAhead만큼 준비한다.
func TestLibraryWantNightWindow(t *testing.T) {
	kst := time.FixedZone("KST", 9*60*60)
	at := func(h int) func() time.Time {
		return func() time.Time { return time.Date(2026, 11, 16, h, 0, 0, 0, kst) }
	}
	cfg := Config{LibraryAhead: 2, NightAhead: 5, NightStartHour: 2, NightEndHour: 6}
	cases := []struct {
		name   string
		cfg    func(Config) Config
		hour   int
		want   int
		reason string
	}{
		{"night", nil, 3, 5, "library_night"},
		{"day", nil, 14, 2, "library_ahead"},
		{"window end is exclusive", nil, 6, 2, "library_ahead"},
		{"night off", func(c Config) Config { c.NightAhead = 0; return c }, 3, 2, "library_ahead"},
		{"empty window", func(c Config) Config { c.NightEndHour = 2; return c }, 2, 2, "library_ahead"},
		{"night not larger", func(c Config) Config { c.LibraryAhead = 6; return c }, 3, 6, "library_ahead"},
		{"day off, night on", func(c Config) Config { c.LibraryAhead = 0; return c }, 3, 5, "library_night"},
	}
	for _, tc := range cases {
		c := cfg
		if tc.cfg != nil {
			c = tc.cfg(c)
		}
		s := &Service{Cfg: c, Now: at(tc.hour)}
		if n, r := s.libraryWant(); n != tc.want || r != tc.reason {
			t.Errorf("%s: got (%d, %s), want (%d, %s)", tc.name, n, r, tc.want, tc.reason)
		}
	}
	// 낮·새벽 모두 0이면 아무 일도 하지 않는다(Store가 nil이어도 안전).
	s := &Service{Cfg: Config{OwnerUserID: "owner"}, Now: at(3)}
	if rep, err := s.GenerateLibraryAhead(context.Background()); err != nil || rep.Ready != 0 {
		t.Fatalf("all off must be a no-op: %+v %v", rep, err)
	}
}
