package services

import (
	"context"
	"testing"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/store"
)

// TestReanchorInstantPretest: the pretest written during an instant
// generation (no anchors) points to the answering blocks when the lecture is
// first opened; an item nothing answers stays unanchored.
func TestReanchorInstantPretest(t *testing.T) {
	s, _, user, f := checkService(t)
	ctx := context.Background()
	job, _, err := s.Store.CreateJob(ctx, store.NewJob{IdempotencyKey: "reanchor:" + user, Kind: "instant", UserID: user,
		ConceptID: f.conceptID, ObjectiveKey: "k", Depth: 1, Reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	content := `{"schemaVersion":1,"title":"t","sections":[
		{"anchor":"principles","kind":"principles","title":"원리","blocks":[
			{"anchor":"principles-1","type":"keypoint","kind":"definition","text":"**유니크 제약**은 같은 이메일 값의 두 번째 INSERT 를 DB 가 거부하게 한다."},
			{"anchor":"principles-2","type":"markdown","text":"외래 키는 부모 행이 없는 자식 행을 막는다."},
			{"anchor":"principles-3","type":"answer","text":"유니크 제약 이메일 거부"}]}]}`
	if _, err := testDB(s).ExecContext(ctx, `UPDATE lectures SET content = ?, generation_job_id = ? WHERE id = ?`, content, job.ID, f.lectureID); err != nil {
		t.Fatal(err)
	}
	items, err := s.Store.SaveDiagnosticItems(ctx, store.DiagnosticItems{CreatedFor: user, Items: []check.Item{
		{ObjectiveID: f.objA, Purpose: check.PurposePretest, Format: check.FormatPredict, Prompt: "같은 이메일로 두 번 INSERT 하면 유니크 제약이 있을 때 어떻게 될까요?",
			Rubric: check.Rubric{Answer: "두 번째 INSERT 가 거부된다"}},
		{ObjectiveID: f.objB, Purpose: check.PurposePretest, Format: check.FormatPredict, Prompt: "캐시 적중률은 어떻게 재나요?"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	s.reanchorPretest(ctx, user, f.lectureID)
	got, err := s.Store.CheckItemsByID(ctx, user, []string{items[0].ID, items[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Rubric.Anchor != "principles-1" || got[1].Rubric.Anchor != "" {
		t.Fatalf("anchors = %q, %q", got[0].Rubric.Anchor, got[1].Rubric.Anchor)
	}
}
