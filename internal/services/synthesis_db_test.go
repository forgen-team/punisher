package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/store"
)

// synthFixture adds three of the user's own concepts in two areas, one
// objective each, all with box-1 mastery due in 20 days.
func synthFixture(t *testing.T, s *Service, user string) []string {
	t.Helper()
	ctx := context.Background()
	db := testDB(s)
	suffix := strings.TrimPrefix(user, "test-prompts-")
	var objs []string
	for i, area := range []string{"network", "network", "os"} {
		concept := fmt.Sprintf("test.synth%d-%s", i, suffix)
		obj := concept + "#1"
		objs = append(objs, obj)
		for _, st := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO concepts (id, name, area, depth, owner_user_id) VALUES (?, ?, ?, 1, ?)`, []any{concept, fmt.Sprintf("개념%d", i), area, user}},
			{`INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES (?, ?, '설명한다', 1, 1)`, []any{obj, concept}},
			{`INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best, next_review_at, last_credit_at, objective_text, objective_depth)
				VALUES (?, ?, ?, 1, 'unaided', ?, ?, '설명한다', 1)`, []any{user, obj, concept,
				store.FormatTime(time.Now().Add(20 * 24 * time.Hour)), store.FormatTime(time.Now().Add(-3 * 24 * time.Hour))}},
		} {
			if _, err := db.ExecContext(ctx, st.sql, st.args...); err != nil {
				t.Fatal(err)
			}
		}
	}
	return objs
}

func synthSetJSON(a, b, c string) string {
	item := func(covers ...string) string {
		return fmt.Sprintf(`{"objective_id":"","purpose":"synthesis","format":"integrative","prompt":"어디부터 보겠어요?",
			"rubric":{"must":["메모리"],"answer":"먼저 원인을 좁힌다","covers":["%s"]}}`, strings.Join(covers, `","`))
	}
	return `{"items":[` + item(a, c) + "," + item(b, c) + `]}`
}

// TestSynthesisFlow: not eligible → eligible with 3 checked concepts in 2
// areas → items made on start → a defended answer credits integrative_ok on its
// covers → a failed item pulls its covers' reviews forward without moving boxes
// → no second synthesis the same week.
func TestSynthesisFlow(t *testing.T) {
	s, fake, user, _ := checkService(t)
	ctx := context.Background()
	if _, _, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: check.PurposeSynthesis}); !errors.Is(err, ErrNoSynthesis) {
		t.Fatalf("no checked concepts → ErrNoSynthesis, got %v", err)
	}
	objs := synthFixture(t, s, user)
	if due, err := s.SynthesisDue(ctx, user, time.Now()); err != nil || !due {
		t.Fatalf("due = %v %v", due, err)
	}
	fake.push("items", synthSetJSON(objs[0], objs[1], objs[2]))
	d, created, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: check.PurposeSynthesis})
	if err != nil || !created || d.Session.Kind != check.PurposeSynthesis {
		t.Fatalf("start = %+v %v %v", d, created, err)
	}
	id := d.Session.ID
	got := waitIdle(t, s, user, id)
	if got.Session.Status != "active" || len(got.Items) != 2 {
		t.Fatalf("session = %+v items %d", got.Session, len(got.Items))
	}

	// Item 1: correct → challenge → defended.
	fake.push("tutor", tutorJSON("correct", "challenge", "그런데 캐시가 원인이면요?", 0.9), tutorJSON("correct", "next", "맞아요.", 0.9))
	answer(t, s, user, id, "메모리에서 읽어요")
	answer(t, s, user, id, "메모리에서 읽어요")
	// Item 2: wrong three times → failed.
	fake.push("tutor", tutorJSON("incorrect", "narrow", "어디를 먼저 볼까요?", 0.9),
		tutorJSON("incorrect", "explain", "먼저 지표를 봐요. 그럼 어디일까요?", 0.9), tutorJSON("incorrect", "wrap", "마무리할게요.", 0.9))
	answer(t, s, user, id, "모르겠어요")
	answer(t, s, user, id, "모르겠어요")
	end := answer(t, s, user, id, "모르겠어요")
	if end.Session.Status != "finished" {
		t.Fatalf("session should finish: %+v", end.Session)
	}

	type row struct {
		box   int
		integ bool
		days  float64
	}
	read := func(obj string) row {
		var r row
		if err := testDB(s).QueryRowContext(ctx, `SELECT box, integrative_ok, julianday(next_review_at) - julianday('now')
			FROM user_objective_mastery WHERE user_id = ? AND objective_id = ?`, user, obj).Scan(&r.box, &r.integ, &r.days); err != nil {
			t.Fatal(err)
		}
		return r
	}
	a, b, c := read(objs[0]), read(objs[1]), read(objs[2])
	if !a.integ || !c.integ || b.integ {
		t.Fatalf("integrative credit: a=%v b=%v c=%v (want a, c from item 1)", a.integ, b.integ, c.integ)
	}
	if a.box != 1 || b.box != 1 || c.box != 1 {
		t.Fatalf("boxes must not move: %d %d %d", a.box, b.box, c.box)
	}
	if b.days > 2 || c.days > 2 || a.days < 19 {
		t.Fatalf("item 2 failure pulls b and c forward only: a=%.1f b=%.1f c=%.1f days", a.days, b.days, c.days)
	}
	if due, err := s.SynthesisDue(ctx, user, time.Now()); err != nil || due {
		t.Fatalf("a synthesis this week blocks the next: %v %v", due, err)
	}
}
