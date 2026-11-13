package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
	"github.com/wooo-jin/punisher/internal/llm"
)

// pipelineLLM answers drafts (teacher system prompt) and reviews (reviewer
// system prompt) from separate queues and records the prompts.
type pipelineLLM struct {
	mu              sync.Mutex
	drafts, reviews []string
	draftUsers      []string
	reviewCalls     int
}

func (f *pipelineLLM) Create(_ context.Context, _ string, req llm.MessagesRequest) (llm.MessagesResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	queue := &f.drafts
	if strings.Contains(req.System, "기술 감수자") {
		queue = &f.reviews
		f.reviewCalls++
	} else {
		f.draftUsers = append(f.draftUsers, req.User)
	}
	if len(*queue) == 0 {
		return llm.MessagesResult{}, errors.New("no scripted reply")
	}
	text := (*queue)[0]
	*queue = (*queue)[1:]
	return llm.MessagesResult{Text: text, Model: "fake", Usage: llm.Usage{InputTokens: 10, OutputTokens: 5}}, nil
}

func (f *pipelineLLM) Stream(ctx context.Context, key string, req llm.MessagesRequest, _ func(string) error) (llm.MessagesResult, error) {
	return f.Create(ctx, key, req)
}

const (
	reviewPass  = `{"verdict":"pass","summary":"좋음","issues":[]}`
	reviewMajor = `{"verdict":"fail","summary":"오류","issues":[{"severity":"major","kind":"factual_error","section":"principles","quote":"q","message":"자리올림과 오버플로를 혼동","suggestion":"구분"}]}`
)

type instantRun struct {
	s      *Service
	user   string
	jobID  string
	fake   *pipelineLLM
	events []Event
}

// runInstantWith runs the instant pipeline for the first shared concept with
// scripted drafts and reviews (nothing is streamed to a client).
func runInstantWith(t *testing.T, drafts func(objectiveIDs []string) []string, reviews ...string) *instantRun {
	t.Helper()
	s, user := dbService(t)
	ctx := context.Background()
	var conceptID string
	if err := testDB(s).QueryRowContext(ctx, `
		SELECT c.id FROM concepts c WHERE c.status = 'active' AND c.owner_user_id IS NULL
		  AND EXISTS (SELECT 1 FROM objectives o WHERE o.concept_id = c.id AND o.status = 'active' AND o.depth = 1)
		ORDER BY c.id LIMIT 1`).Scan(&conceptID); err != nil {
		t.Skip(err)
	}
	tgt, err := s.instantTarget(ctx, user, conceptID, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	fake := &pipelineLLM{drafts: drafts(tgt.Plan.ObjectiveIDs), reviews: reviews}
	s.LLM = fake
	job, _, err := s.Store.CreateJob(ctx, store.NewJob{
		IdempotencyKey: "test-instant:" + user, Kind: "instant", UserID: user, ConceptID: conceptID,
		ObjectiveKey: "test-instant:" + user, Depth: tgt.Plan.Depth, Reason: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	// The database is a temp file per test (dbService), so no cleanup is needed.
	r := &instantRun{s: s, user: user, jobID: job.ID, fake: fake}
	s.runInstant("k", user, job.ID, tgt, func(e Event) { r.events = append(r.events, e) })
	return r
}

// stages lists "stage/attempt" of the stage events in order.
func (r *instantRun) stages() []string {
	var out []string
	for _, e := range r.events {
		if se, ok := e.Data.(StageEvent); ok && e.Type == EventStage {
			out = append(out, se.Stage+"/"+string(rune('0'+se.Attempt)))
		}
	}
	return out
}

func (r *instantRun) last() StageEvent {
	for i := len(r.events) - 1; i >= 0; i-- {
		if se, ok := r.events[i].Data.(StageEvent); ok {
			return se
		}
	}
	return StageEvent{}
}

func (r *instantRun) job(t *testing.T) models.GenerationJob {
	t.Helper()
	j, err := r.s.Store.GetJob(context.Background(), r.user, r.jobID)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func repeat(n int, text string) func([]string) []string {
	return func([]string) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = text
		}
		return out
	}
}

func valid(n int) func([]string) []string {
	return func(ids []string) []string { return repeat(n, lectureMarkupFor(ids))(ids) }
}

func TestInstantPublishesOnPass(t *testing.T) {
	r := runInstantWith(t, valid(1), reviewPass)
	want := "writing/0 checking/0 reviewing/0 published/0"
	if got := strings.Join(r.stages(), " "); got != want {
		t.Fatalf("stages = %s, want %s", got, want)
	}
	last := r.last()
	if last.LectureID == "" || last.MaxAttempts != InstantRewrites || last.Type != "stage" {
		t.Fatalf("published stage = %+v", last)
	}
	for _, e := range r.events {
		if e.Type == "delta" {
			t.Fatal("the draft text must not be streamed")
		}
	}
	if done := r.events[len(r.events)-1]; done.Type != EventDone {
		t.Fatalf("legacy done must follow the published stage: %+v", done)
	}
	lec, err := r.s.Store.GetLecture(context.Background(), r.user, last.LectureID)
	if err != nil {
		t.Fatal(err)
	}
	if lec.Status != "published" || lec.ReviewState != models.ReviewPassed || !lec.Reviewed || lec.EstimatedMinutes < 15 || lec.EstimatedMinutes > 30 {
		t.Fatalf("lecture = status %s review %s minutes %d", lec.Status, lec.ReviewState, lec.EstimatedMinutes)
	}
	if j := r.job(t); j.Status != "succeeded" || j.LectureID != lec.ID {
		t.Fatalf("job = %+v", j)
	}
}

func TestInstantRewritesAfterReviewFail(t *testing.T) {
	r := runInstantWith(t, valid(2), reviewMajor, reviewPass)
	want := "writing/0 checking/0 reviewing/0 rewriting/1 checking/1 reviewing/1 published/1"
	if got := strings.Join(r.stages(), " "); got != want {
		t.Fatalf("stages = %s, want %s", got, want)
	}
	if len(r.fake.draftUsers) != 2 || !strings.Contains(r.fake.draftUsers[1], "자리올림과 오버플로를 혼동") {
		t.Fatal("the rewrite must receive the review findings")
	}
	var log []store.AttemptRecord
	var raw []byte
	_ = testDB(r.s).QueryRowContext(context.Background(), `SELECT attempt_log FROM generation_jobs WHERE id = ?`, r.jobID).Scan(&raw)
	if json.Unmarshal(raw, &log) != nil || len(log) != 2 || log[0].Result != "review_failed" || log[1].Result != "ok" {
		t.Fatalf("attempt log = %s", raw)
	}
}

// TestInstantNotPublishedAfterRewrites: the first draft and both rewrites fail
// review → nothing is published, the stream ends with stage failed and short
// reasons, and the target stays free for '다시 만들기'.
func TestInstantNotPublishedAfterRewrites(t *testing.T) {
	r := runInstantWith(t, valid(3), reviewMajor, reviewMajor, reviewMajor)
	want := "writing/0 checking/0 reviewing/0 rewriting/1 checking/1 reviewing/1 rewriting/2 checking/2 reviewing/2 failed/2"
	if got := strings.Join(r.stages(), " "); got != want {
		t.Fatalf("stages = %s, want %s", got, want)
	}
	last := r.last()
	if last.LectureID != "" || len(last.Reasons) == 0 || len(last.Reasons) > 3 || last.Reasons[0] != reviewReasons["factual_error"] {
		t.Fatalf("failed stage = %+v", last)
	}
	for _, reason := range last.Reasons {
		if strings.Contains(reason, "자리올림과 오버플로를 혼동") {
			t.Fatal("reasons are summaries, never the reviewer's raw text")
		}
	}
	if e := r.events[len(r.events)-1]; e.Type != EventError {
		t.Fatalf("legacy error must follow the failed stage: %+v", e)
	}
	var published int
	_ = testDB(r.s).QueryRowContext(context.Background(), `SELECT COUNT(*) FROM lectures WHERE generation_job_id = ?`, r.jobID).Scan(&published)
	if published != 0 {
		t.Fatalf("a failed instant lecture must not be saved (%d rows)", published)
	}
	j := r.job(t)
	if j.Status != "failed" || j.Reason != JobReasonReviewFailed || j.LectureID != "" {
		t.Fatalf("job = %+v", j)
	}
	final := InstantFinalEvents(j)
	if se, ok := final[0].Data.(StageEvent); !ok || se.Stage != StageFailed || strings.Join(se.Reasons, "|") != strings.Join(last.Reasons, "|") {
		t.Fatalf("late subscriber events = %+v", final)
	}
}

// TestInstantBadDraftSkipsReview: an unparseable draft and one with a
// blocking machine finding (a meta phrase) are rewritten without a review call.
func TestInstantBadDraftSkipsReview(t *testing.T) {
	r := runInstantWith(t, func(ids []string) []string {
		good := lectureMarkupFor(ids)
		leak := strings.Replace(good, "자리올림과 넘침을 혼동했습니다.", "이 부분은 다른 강의에서 다룹니다.", 1)
		return []string{"@@lecture\ntitle: 깨진 출력\n@@frobnicate\n본문\n@@end", leak, good}
	}, reviewPass)
	want := "writing/0 checking/0 rewriting/1 checking/1 rewriting/2 checking/2 reviewing/2 published/2"
	if got := strings.Join(r.stages(), " "); got != want {
		t.Fatalf("stages = %s, want %s", got, want)
	}
	if r.fake.reviewCalls != 1 {
		t.Fatalf("review calls = %d, want 1 (only the clean draft)", r.fake.reviewCalls)
	}
	if !strings.Contains(r.fake.draftUsers[1], "frobnicate") || !strings.Contains(r.fake.draftUsers[2], "meta phrase") {
		t.Fatal("each rewrite receives the previous draft's problems")
	}
	if !strings.Contains(r.fake.draftUsers[2], "<<<DRAFT") || strings.Contains(r.fake.draftUsers[1], "<<<DRAFT") {
		t.Fatal("a parsed draft is revised in place; an unparseable one is rewritten from scratch")
	}
}

// TestInstantKeepsPassingDraft: a draft with only advisory findings (no
// @@pause) passes; its advisory rewrite gets a blocking review finding, so the
// earlier passing draft is published (and stays the newest version).
func TestInstantKeepsPassingDraft(t *testing.T) {
	r := runInstantWith(t, func(ids []string) []string {
		good := lectureMarkupFor(ids)
		return []string{strings.Replace(good, "@@pause\n위 쿼리는 무엇을 출력할까요?\n", "", 1), good}
	}, reviewPass, reviewMajor)
	want := "writing/0 checking/0 reviewing/0 rewriting/1 checking/1 reviewing/1 published/1"
	if got := strings.Join(r.stages(), " "); got != want {
		t.Fatalf("stages = %s, want %s", got, want)
	}
	if !strings.Contains(r.fake.draftUsers[1], "### 개선 권장") || !strings.Contains(r.fake.draftUsers[1], "@@pause") {
		t.Fatal("the advisory rewrite receives the advisory findings")
	}
	lec, err := r.s.Store.GetLecture(context.Background(), r.user, r.last().LectureID)
	if err != nil {
		t.Fatal(err)
	}
	var c lecture.Content
	if err := json.Unmarshal(lec.Content, &c); err != nil {
		t.Fatal(err)
	}
	if lec.Status != "published" || lecture.MeasureContent(c).Pauses != 0 {
		t.Fatalf("the first (pause-less, passing) draft must be the published one: status %s", lec.Status)
	}
	var newest string
	_ = testDB(r.s).QueryRowContext(context.Background(),
		`SELECT status FROM lectures WHERE generation_job_id = ? ORDER BY version DESC LIMIT 1`, r.jobID).Scan(&newest)
	if newest != "published" {
		t.Fatalf("the published version must be the newest, got %s", newest)
	}
	var log []store.AttemptRecord
	var raw []byte
	_ = testDB(r.s).QueryRowContext(context.Background(), `SELECT attempt_log FROM generation_jobs WHERE id = ?`, r.jobID).Scan(&raw)
	if json.Unmarshal(raw, &log) != nil || len(log) != 2 || log[0].Result != "ok" || !log[0].Published ||
		!strings.Contains(strings.Join(log[0].Advisory, " "), "@@pause") || log[1].Result != "review_failed" || log[1].Published ||
		log[1].Drift == "" {
		t.Fatalf("attempt log = %s", raw)
	}
}
