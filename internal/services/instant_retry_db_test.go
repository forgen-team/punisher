package services

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/wooo-jin/punisher/internal/store"
)

func lectureMarkupFor(objectiveIDs []string) string {
	return "@@lecture\ntitle: 재시도 테스트\nobjectives: " + strings.Join(objectiveIDs, ", ") + "\nnew_scope: x\n" +
		"@@section situation_question minutes=3 title=a\n@@markdown\n상황\n" +
		"@@section principles minutes=12 title=b\n@@markdown\n원리\n@@sources\n- [doc](https://example.org/a)\n" +
		"@@section walkthrough minutes=10 title=c\n@@code lang=bash\nls\n@@sources\n- [doc](https://example.org/b)\n" +
		"@@section judgment_next minutes=5 title=d\n@@markdown\n판단\n@@end"
}

// TestInstantRewritesOnceAfterBadMarkup: the first output has an unknown @@
// directive, the second is valid → one lecture saved, two attempts recorded,
// a "rewriting" status emitted between them.
func TestInstantRewritesOnceAfterBadMarkup(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	db := s.Store.(*store.SQLiteStore).DB() // SQLite port of the Pool() queries below
	var conceptID string
	if err := db.QueryRowContext(ctx, `
		SELECT c.id FROM concepts c WHERE c.status = 'active' AND c.owner_user_id IS NULL
		  AND EXISTS (SELECT 1 FROM objectives o WHERE o.concept_id = c.id AND o.status = 'active' AND o.depth = 1)
		ORDER BY c.id LIMIT 1`).Scan(&conceptID); err != nil {
		t.Skip(err)
	}
	tgt, err := s.instantTarget(ctx, user, conceptID, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	bad := "@@lecture\ntitle: 깨진 출력\n@@frobnicate\n본문\n@@end"
	fake := &sequenceLLM{texts: []string{bad, lectureMarkupFor(tgt.Plan.ObjectiveIDs)}}
	s.LLM = fake
	job, _, err := s.Store.CreateJob(ctx, store.NewJob{
		IdempotencyKey: "test-retry:" + user, Kind: "instant", UserID: user, ConceptID: conceptID,
		ObjectiveKey: "test-retry:" + user, Depth: tgt.Plan.Depth, Reason: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	var events []Event
	s.runInstant("k", user, job.ID, tgt, func(e Event) { events = append(events, e) })

	var lectures int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lectures WHERE generation_job_id = ?`, job.ID).Scan(&lectures)
	var status string
	var attempts int
	var raw []byte
	if err := db.QueryRowContext(ctx, `SELECT status, attempts, attempt_log FROM generation_jobs WHERE id = ?`, job.ID).
		Scan(&status, &attempts, &raw); err != nil {
		t.Fatal(err)
	}
	var log []store.AttemptRecord
	_ = json.Unmarshal(raw, &log)
	if lectures != 1 || status != "succeeded" || attempts != 2 || len(log) != 2 ||
		log[0].Result != "parse_error" || !strings.Contains(log[0].Message, "frobnicate") || log[1].Result != "ok" {
		t.Fatalf("lectures=%d status=%s attempts=%d log=%+v", lectures, status, attempts, log)
	}
	if len(fake.users) != 2 || !strings.Contains(fake.users[1], "frobnicate") {
		t.Fatal("the rewrite must receive the parse error as revision notes")
	}
	rewriting, done := false, false
	for _, e := range events {
		if m, ok := e.Data.(map[string]string); ok && e.Type == EventStatus && m["stage"] == StageRewriting {
			rewriting = true
		}
		done = done || e.Type == EventDone
	}
	if !rewriting || !done {
		t.Fatalf("events must include rewriting status and done: %+v", events)
	}
}
