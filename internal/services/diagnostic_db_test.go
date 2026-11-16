package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
)

func pretestChoiceJSON(obj, purpose string) string {
	return fmt.Sprintf(`{"objective_id":%q,"purpose":%q,"format":"choice","prompt":"같은 파일을 두 번째 읽으면 어디서 읽을까요?",
		"material":{"choices":["디스크에서 다시 읽는다","메모리의 페이지 캐시에서 읽는다","CPU 레지스터에서 읽는다"]},
		"rubric":{"answer":"","correct":1}}`, obj, purpose)
}

func pretestSetJSON(f checkFixture) string {
	return `{"items":[` + pretestChoiceJSON(f.objA, "pretest") + `,
		{"objective_id":"` + f.objB + `","purpose":"pretest","format":"predict","prompt":"write 직후 전원이 꺼지면 데이터는 어떻게 될까요?",
		"rubric":{"answer":"아직 디스크에 쓰지 않은 더티 페이지는 사라질 수 있다"}}]}`
}

func verifySetJSON(objectives ...string) string {
	items := []string{}
	for _, o := range objectives {
		items = append(items, fmt.Sprintf(`{"objective_id":%q,"purpose":"known_verify","format":"recall","prompt":"%s 를 자기 말로 설명해 보세요.",
			"rubric":{"must":["메모리에서 읽는다"],"answer":%q}}`, o, o, leakAnswer))
	}
	return `{"items":[` + strings.Join(items, ",") + `]}`
}

func startDiagnostic(t *testing.T, s *Service, user string, req DiagnosticRequest) *models.CheckDetail {
	t.Helper()
	d, _, err := s.StartDiagnostic(context.Background(), user, req)
	if err != nil {
		t.Fatalf("start %s: %v", req.Kind, err)
	}
	if d == nil {
		return nil
	}
	idle := waitIdle(t, s, user, d.Session.ID)
	return &idle
}

func conceptState(t *testing.T, s *Service, user, conceptID string) string {
	t.Helper()
	var state string
	if err := testDB(s).QueryRowContext(context.Background(), `SELECT state FROM user_concept_states WHERE user_id = ? AND concept_id = ?`,
		user, conceptID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	return state
}

func choose(t *testing.T, s *Service, user, id string, choice int) models.CheckDetail {
	t.Helper()
	d, err := s.AnswerChoice(context.Background(), user, id, &choice, false)
	if err != nil {
		t.Fatalf("choose %d: %v", choice, err)
	}
	return d
}

// TestPretestRecordsWithoutEvidence: a lecture's first open gets 2 predictions
// (one FastModel call), answers are recorded with no feedback and no mastery,
// never asked again, and appear on the check result as '처음 예측'.
func TestPretestRecordsWithoutEvidence(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	fake.push("items", pretestSetJSON(f))
	d := startDiagnostic(t, s, user, DiagnosticRequest{Kind: "pretest", LectureID: f.lectureID})
	if d == nil || d.Session.Kind != "pretest" || d.Session.Status != "active" || len(d.Items) != 2 || d.Session.Grader != learning.GraderNone {
		t.Fatalf("pretest = %+v", d)
	}
	if fake.count("items") != 1 || !strings.Contains(fake.users["items"][0], "정확히 2개") {
		t.Fatal("one FastModel call with the pretest task")
	}
	if !d.Current.CanAnswer || d.Current.CanReveal || d.Current.CanSelfGrade || d.Items[0].Answer != "" {
		t.Fatalf("current = %+v item=%+v", d.Current, d.Items[0])
	}
	if _, err := s.AnswerCheck(ctx, user, d.Session.ID, "메모리", ""); !errors.Is(err, ErrInvalidCheck) {
		t.Fatalf("a choice item needs a choice: %v", err)
	}
	d2 := choose(t, s, user, d.Session.ID, 1)
	if d2.Session.Cursor != 1 || !d2.Items[0].Done || d2.Items[0].Outcome != learning.OutcomeRecorded || d2.Items[0].Answer != "" {
		t.Fatalf("after choice: cursor=%d item=%+v", d2.Session.Cursor, d2.Items[0])
	}
	d3, err := s.AnswerChoice(ctx, user, d.Session.ID, nil, true)
	if err != nil || d3.Session.Status != "finished" {
		t.Fatalf("dont know: %v %+v", err, d3.Session)
	}
	if lastTurn(d3).Move != "dont_know" || lastTurn(d3).Verdict != "" || fake.count("tutor") != 0 {
		t.Fatal("'모르겠어요' is recorded without grading")
	}
	for _, obj := range []string{f.objA, f.objB} {
		if m := masteryOf(t, s, user, f.conceptID, obj); m.UpdatedAt != nil || m.Level != 0 {
			t.Fatalf("pretest must not be evidence: %+v", m)
		}
	}
	if again, _, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: "pretest", LectureID: f.lectureID}); err != nil || again != nil {
		t.Fatalf("asked once per objective: %+v %v", again, err)
	}

	fake.push("items", itemSetJSON(f, true))
	c, _, err := s.StartCheck(ctx, user, "check", f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	cd := waitIdle(t, s, user, c.Session.ID)
	if len(cd.Predictions) != 2 {
		t.Fatalf("predictions = %+v", cd.Predictions)
	}
	byObj := map[string]models.Prediction{}
	for _, p := range cd.Predictions {
		byObj[p.ObjectiveID] = p
	}
	if byObj[f.objA].Answer != "메모리의 페이지 캐시에서 읽는다" || !byObj[f.objB].DontKnow {
		t.Fatalf("predictions by objective = %+v", byObj)
	}
}

// TestPretestOffAndFirstOpenOnly: pretest_enabled off → nothing (no LLM call);
// a lecture read before gets none; the instant-stream pretest goes on in the lecture.
func TestPretestOffAndFirstOpenOnly(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	off, on := false, true
	if _, err := s.Store.UpdateLearningSettings(ctx, user, store.LearningSettingsPatch{PretestEnabled: &off}); err != nil {
		t.Fatal(err)
	}
	if st, _ := s.Store.GetLearningSettings(ctx, user); st.PretestEnabled {
		t.Fatal("pretest_enabled saved off")
	}
	if d, _, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: "pretest", LectureID: f.lectureID}); err != nil || d != nil || fake.count("items") != 0 {
		t.Fatalf("off: %+v %v", d, err)
	}
	if _, err := s.Store.UpdateLearningSettings(ctx, user, store.LearningSettingsPatch{PretestEnabled: &on}); err != nil {
		t.Fatal(err)
	}

	job, _, err := s.Store.CreateJob(ctx, store.NewJob{IdempotencyKey: "instant:" + user, Kind: learning.JobInstant, UserID: user,
		ConceptID: f.conceptID, ObjectiveKey: f.objA + "," + f.objB, Depth: 1, Reason: "instant"})
	if err != nil {
		t.Fatal(err)
	}
	fake.push("items", pretestSetJSON(f))
	d := startDiagnostic(t, s, user, DiagnosticRequest{Kind: "pretest", JobID: job.ID})
	if d == nil || len(d.Items) != 2 {
		t.Fatalf("instant pretest = %+v", d)
	}
	cont, created, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: "pretest", LectureID: f.lectureID})
	if err != nil || created || cont == nil || cont.Session.ID != d.Session.ID {
		t.Fatalf("the lecture continues the instant pretest: %+v created=%v %v", cont, created, err)
	}
	if _, err := s.AbandonCheck(ctx, user, d.Session.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.UpdateProgress(ctx, user, f.lectureID, "s1", 10); err != nil {
		t.Fatal(err)
	}
	if again, _, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: "pretest", LectureID: f.lectureID}); err != nil || again != nil {
		t.Fatalf("a lecture read before gets no pretest: %+v %v", again, err)
	}
}

// TestKnownVerifyPassAndFail: one tutor-graded item per objective, no
// narrowing; a pass is best=unaided box 1, a failure offers '다시 배우기'; the
// concept stays known either way.
func TestKnownVerifyPassAndFail(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	if _, err := s.MarkKnown(ctx, user, f.conceptID, false); err != nil {
		t.Fatal(err)
	}
	fake.push("items", verifySetJSON(f.objA, f.objB))
	d := startDiagnostic(t, s, user, DiagnosticRequest{Kind: "known_verify", ConceptID: f.conceptID})
	if d == nil || d.Session.Kind != "known_verify" || len(d.Items) != 2 || d.Session.Grader != learning.GraderLLM {
		t.Fatalf("verify = %+v", d)
	}
	fake.push("tutor", tutorJSON("correct", "next", "맞아요.", 0.9))
	d2 := answer(t, s, user, d.Session.ID, "메모리에서 읽어요")
	if d2.Session.Cursor != 1 || d2.Items[0].Outcome != learning.OutcomeUnaidedCorrect {
		t.Fatalf("after pass: %+v", d2.Items[0])
	}
	if !strings.Contains(fake.users["tutor"][0], "허용된 tutor_move\nnext") {
		t.Fatal("a known verification allows no narrowing question")
	}
	a := masteryOf(t, s, user, f.conceptID, f.objA)
	if a.Box != 1 || a.Best != learning.BestUnaided || a.NextReviewAt == nil {
		t.Fatalf("pass mastery = %+v", a)
	}
	fake.push("tutor", tutorJSON("incorrect", "wrap", "핵심이 빠졌어요.", 0.9))
	d3 := answer(t, s, user, d.Session.ID, "잘 모르겠어요")
	if d3.Session.Status != "finished" {
		t.Fatalf("finished? %+v", d3.Session)
	}
	var result struct {
		Items []store.ResultEntry `json:"items"`
	}
	if err := json.Unmarshal(d3.Session.Result, &result); err != nil || len(result.Items) != 2 {
		t.Fatalf("result = %s (%v)", d3.Session.Result, err)
	}
	if result.Items[0].Relearn || !result.Items[1].Relearn || result.Items[1].Outcome != learning.OutcomeFailed {
		t.Fatalf("relearn only for the failed objective: %+v", result.Items)
	}
	if state := conceptState(t, s, user, f.conceptID); state != learning.Known {
		t.Fatalf("the concept stays known: %s", state)
	}
}

// TestBulkKnownQueueMixesIntoReview: bulk '이미 앎' queues the objectives; the
// scheduler writes their known_verify items; the day's review mixes two of them
// in (within throughput), and a pass/failure leaves the queue.
// (SQLite keeps result JSON compact: `"relearn":true`, witems JSONB prints `"relearn": true`.)
func TestBulkKnownQueueMixesIntoReview(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	s.Cfg.OwnerUserID = user
	if _, err := s.MarkKnown(ctx, user, f.conceptID, true); err != nil {
		t.Fatal(err)
	}
	if sum, err := s.ReviewSummary(ctx, user); err != nil || sum.Verify != 0 || sum.Due != 0 {
		t.Fatalf("no items yet: %+v %v", sum, err)
	}
	if _, _, err := s.StartReview(ctx, user); !errors.Is(err, ErrNoReviews) {
		t.Fatalf("nothing to mix yet: %v", err)
	}
	fake.push("items", verifySetJSON(f.objA, f.objB))
	rep, err := s.PregenerateKnownVerify(ctx)
	if err != nil || rep.Generated != 1 {
		t.Fatalf("pregenerate = %+v %v", rep, err)
	}
	if rep, err := s.PregenerateKnownVerify(ctx); err != nil || rep.Generated+rep.Skipped != 0 {
		t.Fatalf("a day's verifications are ready: %+v %v", rep, err)
	}
	sum, err := s.ReviewSummary(ctx, user)
	if err != nil || sum.Verify != 2 || sum.EstimatedMinutes != 2 {
		t.Fatalf("summary = %+v %v", sum, err)
	}
	d, _, err := s.StartReview(ctx, user)
	if err != nil || d.Session.Kind != "review" || len(d.Items) != 2 {
		t.Fatalf("review = %+v %v", d.Session, err)
	}
	selfGrade(t, s, user, d.Session.ID, "메모리에서 읽어요", "correct")
	end := selfGrade(t, s, user, d.Session.ID, "모르겠어요", "incorrect")
	if end.Session.Status != "finished" {
		t.Fatalf("review end = %+v", end.Session)
	}
	first, second := end.Items[0].ObjectiveID, end.Items[1].ObjectiveID
	if m := masteryOf(t, s, user, f.conceptID, first); m.Box != 1 || m.Best != learning.BestUnaided {
		t.Fatalf("passed = %+v", m)
	}
	var queued int
	if err := testDB(s).QueryRowContext(ctx, `SELECT COUNT(*) FROM user_objective_mastery WHERE user_id = ? AND verify_queued`, user).Scan(&queued); err != nil || queued != 0 {
		t.Fatalf("both left the queue: %d %v", queued, err)
	}
	if !strings.Contains(string(end.Session.Result), `"relearn":true`) || second == "" {
		t.Fatalf("the failed verification offers relearn: %s", end.Session.Result)
	}
	if sum, err := s.ReviewSummary(ctx, user); err != nil || sum.Verify != 0 {
		t.Fatalf("today's verifications are used: %+v %v", sum, err)
	}
}

// placementFixture adds two more of the user's concepts and a chapter whose
// draft lists them (with the fixture concept) in chapter order.
// (SQLite: temporary database, ids from store.NewID(), no cleanup.)
func placementFixture(t *testing.T, s *Service, user string, f checkFixture) (string, []string) {
	t.Helper()
	ctx := context.Background()
	db := testDB(s)
	ids := []string{f.conceptID}
	for i, depth := range []int{2, 1} {
		id := fmt.Sprintf("%s-p%d", f.conceptID, i)
		ids = append(ids, id)
		if _, err := db.ExecContext(ctx, `INSERT INTO concepts (id, name, area, depth, owner_user_id) VALUES (?, ?, 'os', ?, ?)`,
			id, "배치 "+id, depth, user); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES (?, ?, '설명한다', 1, 1)`,
			id+"#1", id); err != nil {
			t.Fatal(err)
		}
	}
	chapterID := store.NewID()
	if _, err := db.ExecContext(ctx, `INSERT INTO chapters (id, user_id, title) VALUES (?, ?, '챕터')`, chapterID, user); err != nil {
		t.Fatal(err)
	}
	draft := fmt.Sprintf(`{"title":"챕터","concepts":[{"id":%q},{"id":%q},{"id":%q}]}`, ids[0], ids[1], ids[2])
	if _, err := db.ExecContext(ctx, `INSERT INTO chapter_drafts (id, user_id, draft, chapter_id) VALUES (?, ?, ?, ?)`,
		store.NewID(), user, draft, chapterID); err != nil {
		t.Fatal(err)
	}
	return chapterID, ids
}

// TestPlacementCandidatesAndConfirm: easy → hard choice items, answers graded
// by code without evidence; right answers are candidates, and confirming one
// makes it known (no verification queue, StartingPoint untouched).
func TestPlacementCandidatesAndConfirm(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	chapterID, ids := placementFixture(t, s, user, f)
	easyOrder := []string{f.objA, ids[2] + "#1", ids[1] + "#1"} // 깊이 1(챕터 순) → 깊이 2
	items := []string{}
	for _, o := range []string{ids[1] + "#1", f.objA, ids[2] + "#1"} {
		items = append(items, pretestChoiceJSON(o, "placement"))
	}
	fake.push("items", `{"items":[`+strings.Join(items, ",")+`]}`)
	d := startDiagnostic(t, s, user, DiagnosticRequest{Kind: "placement", ChapterID: chapterID})
	if d == nil || len(d.Items) != 3 || d.Placement == nil || d.Placement.ChapterID != chapterID {
		t.Fatalf("placement = %+v", d)
	}
	for i, it := range d.Items {
		if it.ObjectiveID != easyOrder[i] {
			t.Fatalf("item %d = %s, want %s (easy → hard)", i, it.ObjectiveID, easyOrder[i])
		}
	}
	choose(t, s, user, d.Session.ID, 1)
	choose(t, s, user, d.Session.ID, 0)
	end := choose(t, s, user, d.Session.ID, 1)
	if end.Session.Status != "finished" || end.Placement == nil || len(end.Placement.Candidates) != 2 {
		t.Fatalf("candidates = %+v", end.Placement)
	}
	if end.Placement.Candidates[0].ConceptID != f.conceptID || end.Placement.Candidates[1].ConceptID != ids[1] {
		t.Fatalf("candidates in asking order: %+v", end.Placement.Candidates)
	}
	if m := masteryOf(t, s, user, f.conceptID, f.objA); m.UpdatedAt != nil {
		t.Fatalf("placement is not evidence: %+v", m)
	}
	if _, err := s.MarkKnown(ctx, user, f.conceptID, false); err != nil {
		t.Fatal(err)
	}
	if state := conceptState(t, s, user, f.conceptID); state != learning.Known {
		t.Fatalf("confirmed known: %s", state)
	}
	var queued int
	if err := testDB(s).QueryRowContext(ctx, `SELECT COUNT(*) FROM user_objective_mastery WHERE user_id = ?`, user).Scan(&queued); err != nil || queued != 0 {
		t.Fatalf("a confirmed candidate is not queued: %d %v", queued, err)
	}

	if _, err := s.MarkKnown(ctx, user, ids[1], false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: "placement", ChapterID: chapterID}); !errors.Is(err, ErrNoPlacement) {
		t.Fatalf("fewer than 3 open concepts: %v", err)
	}
}
