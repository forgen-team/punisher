package learning

import (
	"reflect"
	"testing"

	"github.com/forgen-team/punisher/internal/curriculum"
)

// foundationsTodayInput: the 13-concept track (chained like the seed's first
// links) plus two depth-1 roots in other areas.
func foundationsTodayInput(level string, limit int, done ...string) TodayInput {
	concepts := map[string]TodayConcept{}
	var edges []PrereqEdge
	for i, id := range curriculum.FoundationsTrack {
		concepts[id] = TodayConcept{ID: id, Name: id, Area: curriculum.AreaFoundations, Depth: 0, HasPending: true}
		if i > 0 {
			edges = append(edges, PrereqEdge{From: curriculum.FoundationsTrack[i-1], To: id})
		}
	}
	concepts["net.http"] = TodayConcept{ID: "net.http", Name: "HTTP", Area: "network-web", Depth: 1, HasPending: true}
	concepts["os.process"] = TodayConcept{ID: "os.process", Name: "프로세스", Area: "os", Depth: 1, HasPending: true}
	for _, id := range done {
		c := concepts[id]
		c.State, c.HasPending = Completed, false
		concepts[id] = c
	}
	return TodayInput{Concepts: concepts, Edges: edges, Limit: limit, ReaderLevel: level}
}

func todayIDs(res TodayResult) []string {
	out := []string{}
	for _, r := range res.Items {
		out = append(out, r.ConceptID)
	}
	return out
}

func TestTodayFoundationsByReaderLevel(t *testing.T) {
	track := curriculum.FoundationsTrack
	// beginner: the track in order first (locks inside the track do not hold it back).
	got := todayIDs(BuildToday(foundationsTodayInput(ReaderBeginner, 5)))
	if !reflect.DeepEqual(got, track[:5]) {
		t.Fatalf("beginner = %v, want %v", got, track[:5])
	}
	// beginner who finished the first two continues from the third.
	got = todayIDs(BuildToday(foundationsTodayInput(ReaderBeginner, 2, track[0], track[1])))
	if !reflect.DeepEqual(got, track[2:4]) {
		t.Fatalf("beginner after 2 = %v", got)
	}
	// novice: 5·8·9·10·11, then the rest of the track in order.
	want := []string{track[4], track[7], track[8], track[9], track[10], track[0], track[1]}
	got = todayIDs(BuildToday(foundationsTodayInput(ReaderNovice, 7)))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("novice = %v, want %v", got, want)
	}
	// after the track the usual depth-1 gaps follow.
	got = todayIDs(BuildToday(foundationsTodayInput(ReaderBeginner, 3, track...)))
	if len(got) != 2 || got[0] == track[0] {
		t.Fatalf("beginner with the track done = %v", got)
	}
	// practitioner / advanced / unset: foundations are never recommended.
	for _, level := range []string{"", ReaderPractitioner, ReaderAdvanced} {
		res := BuildToday(foundationsTodayInput(level, 20))
		for _, r := range res.Items {
			if r.Area == curriculum.AreaFoundations {
				t.Fatalf("level %q recommended %s", level, r.ConceptID)
			}
		}
		if len(res.Items) != 2 {
			t.Fatalf("level %q = %v", level, todayIDs(res))
		}
		if !FoundationsHidden(level) || FoundationsOrder(level) != nil {
			t.Fatalf("level %q must hide foundations", level)
		}
	}
	if FoundationsHidden(ReaderBeginner) || FoundationsHidden(ReaderNovice) {
		t.Fatal("beginner/novice show foundations")
	}
	if n := len(FoundationsOrder(ReaderNovice)); n != len(track) {
		t.Fatalf("novice order covers %d of %d", n, len(track))
	}
}
