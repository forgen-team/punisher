package services

import (
	"context"
	"strings"
	"testing"

	"github.com/forgen-team/punisher/internal/store"
)

// TestReadingSignals: the screen's dwell times and term opens add up per
// block (unknown anchors dropped, values clipped), flag the block for a
// regeneration, and feed the reader-gap records of the quality harness.
func TestReadingSignals(t *testing.T) {
	s, _, user, f := checkService(t)
	ctx := context.Background()
	content := `{"schemaVersion":1,"title":"t","sections":[
		{"anchor":"principles","kind":"principles","title":"원리","blocks":[
			{"anchor":"principles-1","type":"markdown","text":"시퀀스는 값을 하나씩 늘린다."}]}]}`
	if _, err := testDB(s).ExecContext(ctx, `UPDATE lectures SET content = ? WHERE id = ?`, content, f.lectureID); err != nil {
		t.Fatal(err)
	}
	lec, err := s.Store.GetLecture(ctx, user, f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.RecordReadingSignals(ctx, user, lec, []store.ReadingSignal{
			{Anchor: "principles-1", DwellMs: 40 * 60_000, Terms: map[string]int{"db.sequence": 2}},
			{Anchor: "ghost-9", DwellMs: 1000},
		}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.Store.LectureReadingSignals(ctx, user, f.lectureID)
	if err != nil || len(got) != 1 || got[0].DwellMs != 2*maxReadingDwellMs || got[0].TermOpens != 4 || got[0].Terms["db.sequence"] != 4 {
		t.Fatalf("signals = %+v %v", got, err)
	}
	lines := s.signalLines(ctx, user, lec)
	if len(lines) != 1 || !strings.Contains(lines[0], "용어 설명을 4번 열어 봄") || !strings.Contains(lines[0], "오래 머묾") {
		t.Fatalf("lines = %q", lines)
	}
	if _, err := testDB(s).ExecContext(ctx, `INSERT INTO lecture_reviews (id, lecture_id, verdict, reader_gaps) VALUES ('rv', ?, 'pass', ?)`,
		f.lectureID, `[{"section":"principles","quote":"시퀀스는 값을"}]`); err != nil {
		t.Fatal(err)
	}
	recs, err := s.Store.GapRecords(ctx)
	if err != nil || len(recs) != 1 || recs[0].TermOpens["principles-1"] != 4 {
		t.Fatalf("gap records = %+v %v", recs, err)
	}
}
