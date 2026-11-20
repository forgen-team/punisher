package services

import (
	"context"
	"reflect"
	"testing"

	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
)

// Learner-level ②: today's picks on the shipped seed follow the reader level —
// beginner starts the foundations track in order, novice at 5·8·9·10·11,
// practitioner/advanced/unset never see it. Changing the level changes the picks.
func TestTodayFoundationsOrderByLevel(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	track := curriculum.FoundationsTrack
	picks := func(level string) []string {
		t.Helper()
		if level != "" {
			setReaderLevel(t, s, user, level)
		}
		res, err := s.Today(ctx, user, 5)
		if err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, r := range res.Items {
			out = append(out, r.ConceptID)
		}
		return out
	}
	hasFoundations := func(ids []string) bool {
		for _, id := range ids {
			for _, f := range track {
				if id == f {
					return true
				}
			}
		}
		return false
	}
	if got := picks(""); len(got) == 0 || hasFoundations(got) {
		t.Fatalf("unset level = %v", got)
	}
	if got := picks(learning.ReaderBeginner); !reflect.DeepEqual(got, track[:5]) {
		t.Fatalf("beginner = %v, want %v", got, track[:5])
	}
	want := []string{track[4], track[7], track[8], track[9], track[10]}
	if got := picks(learning.ReaderNovice); !reflect.DeepEqual(got, want) {
		t.Fatalf("novice = %v, want %v", got, want)
	}
	for _, level := range []string{learning.ReaderPractitioner, learning.ReaderAdvanced} {
		if got := picks(level); len(got) == 0 || hasFoundations(got) {
			t.Fatalf("%s = %v", level, got)
		}
	}
}

// A foundations concept plans a depth-0 lecture with its three objectives.
func TestFoundationsLecturePlanDepthZero(t *testing.T) {
	s, user := dbService(t)
	inst, err := s.instantTarget(context.Background(), user, curriculum.FoundationsTrack[0], 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Plan.Depth != 0 || len(inst.Plan.ObjectiveIDs) != 3 {
		t.Fatalf("plan = depth %d objectives %v", inst.Plan.Depth, inst.Plan.ObjectiveIDs)
	}
}
