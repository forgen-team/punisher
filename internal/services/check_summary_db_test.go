package services

import (
	"context"
	"testing"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/store"
)

// TestCheckLectureSummary: a done item carries its lecture's '한 장 요약'
// (v2.2 §6) — not before the answer, and not at all for a lecture without one.
func TestCheckLectureSummary(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	fake.push("items", itemSetJSON(f, true))
	d, _, err := s.StartCheck(ctx, user, "check", f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	d = waitIdle(t, s, user, d.Session.ID)
	if d.Items[0].LectureSummary != "" {
		t.Fatal("no summary for a lecture without a summary block")
	}
	// jsonb_set(content, '{sections,1,blocks}', blocks || [summary]) → json_insert(…, '$[#]').
	if _, err := testDB(s).ExecContext(ctx, `UPDATE lectures SET content = json_insert(content, '$.sections[1].blocks[#]',
		json('{"anchor":"s2-b2","type":"summary","text":"- 읽기는 메모리에서\n- 쓰기는 나중에"}'))
		WHERE id = ?`, f.lectureID); err != nil {
		t.Fatal(err)
	}

	sess, _, err := s.Store.CreateCheckSession(ctx, store.NewCheckSession{UserID: user, Kind: "review", Grader: learning.GraderSelf,
		ConceptID: f.conceptID, SessionKey: "review:summary"})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{d.Items[0].ID}
	if _, err := s.Store.ActivateCheckSession(ctx, user, sess.ID, ids, store.TurnWrite{ItemID: ids[0], Role: "tutor", Move: "ask", Text: d.Items[0].Prompt}); err != nil {
		t.Fatal(err)
	}
	got, err := s.AnswerCheck(ctx, user, sess.ID, "메모리에 남아요", "")
	if err != nil || got.Items[0].LectureSummary != "" {
		t.Fatalf("summary before the item is done = %q (%v)", got.Items[0].LectureSummary, err)
	}
	got, err = s.SelfGradeCheck(ctx, user, sess.ID, "correct")
	if err != nil || got.Session.Status != "finished" {
		t.Fatalf("self grade = %+v (%v)", got.Session, err)
	}
	if want := "- 읽기는 메모리에서\n- 쓰기는 나중에"; got.Items[0].LectureSummary != want {
		t.Fatalf("done review item summary = %q, want %q", got.Items[0].LectureSummary, want)
	}
}
