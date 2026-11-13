package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
)

var validMarkup = lectureMarkupFor([]string{"c#1"})

func testTarget() Target {
	return Target{
		Plan: learning.LecturePlan{ConceptID: "c", ObjectiveIDs: []string{"c#1"}, ObjectiveKey: "c#1", Depth: 1},
		Validation: lecture.ValidationContext{
			KnownConceptIDs: map[string]bool{"c": true}, PlannedObjectiveIDs: []string{"c#1"},
			AllowedObjectiveIDs: map[string]bool{"c#1": true},
		},
	}
}

type fakeSteps struct {
	drafts  []string
	reviews []ReviewOutcome
	genErr  error
	notes   []string
	gen     int
	rev     int
}

func (f *fakeSteps) funcs() LoopFuncs {
	return LoopFuncs{
		Generate: func(_ context.Context, notes string) (string, error) {
			f.notes = append(f.notes, notes)
			if f.genErr != nil && f.gen == len(f.drafts)-1 {
				return "", f.genErr
			}
			d := f.drafts[f.gen]
			f.gen++
			return d, nil
		},
		Review: func(context.Context, string) (ReviewOutcome, error) {
			r := f.reviews[f.rev]
			f.rev++
			return r, nil
		},
		Parse: testTarget().parseAndValidate,
	}
}

var (
	pass  = ReviewOutcome{Verdict: "pass", Issues: []ReviewIssue{{Severity: "minor", Message: "nit"}}}
	major = ReviewOutcome{Verdict: "pass", Issues: []ReviewIssue{{Severity: "major", Kind: "factual_error", Message: "틀림"}}}
)

func TestReviewLoopPublishesOnFirstPass(t *testing.T) {
	f := &fakeSteps{drafts: []string{validMarkup}, reviews: []ReviewOutcome{pass}}
	res, err := RunReviewLoop(context.Background(), 2, f.funcs())
	if err != nil || res.Status != "published" || len(res.Attempts) != 1 || res.Final != 0 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestReviewLoopRegeneratesOnceWithFindings(t *testing.T) {
	f := &fakeSteps{drafts: []string{validMarkup, validMarkup}, reviews: []ReviewOutcome{major, pass}}
	res, err := RunReviewLoop(context.Background(), 2, f.funcs())
	if err != nil || res.Status != "published" || res.Final != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if !strings.Contains(f.notes[1], "틀림") {
		t.Fatalf("regeneration must receive reviewer findings, got %q", f.notes[1])
	}
}

func TestReviewLoopNeedsRevisionAfterSecondFailure(t *testing.T) {
	invalid := strings.Replace(validMarkup, "minutes=9", "minutes=1", 1) // total 12 → schema issue
	f := &fakeSteps{drafts: []string{invalid, validMarkup}, reviews: []ReviewOutcome{pass, major}}
	res, err := RunReviewLoop(context.Background(), 2, f.funcs())
	if err != nil || res.Status != "needs_revision" || res.Final != 1 || len(res.Attempts) != 2 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if !strings.Contains(f.notes[1], "total minutes") {
		t.Fatalf("schema issues must be fed back: %q", f.notes[1])
	}
}

func TestReviewLoopLLMErrorFailsWithoutResult(t *testing.T) {
	boom := errors.New("429")
	f := &fakeSteps{drafts: []string{validMarkup, validMarkup}, reviews: []ReviewOutcome{major}, genErr: boom}
	_, err := RunReviewLoop(context.Background(), 2, f.funcs())
	if !errors.Is(err, boom) {
		t.Fatalf("LLM failure must propagate (job failed, nothing saved), got %v", err)
	}
}

func TestReviewLoopUnparseableIsUnusable(t *testing.T) {
	f := &fakeSteps{drafts: []string{"no markup", "@@lecture\ntitle: x"}}
	if _, err := RunReviewLoop(context.Background(), 2, f.funcs()); !errors.Is(err, ErrUnusableOutput) {
		t.Fatalf("want ErrUnusableOutput, got %v", err)
	}
}

func TestParseReviewNeverPassesGarbage(t *testing.T) {
	if r := ParseReview("I think it's fine"); !r.Blocking() {
		t.Fatal("unparseable review must block")
	}
	if r := ParseReview("```json\n{\"verdict\":\"pass\",\"issues\":[]}\n```"); r.Blocking() {
		t.Fatal("clean pass should not block")
	}
	if r := ParseReview(`{"verdict":"pass","issues":[{"severity":"critical"}]}`); !r.Blocking() {
		t.Fatal("critical issue must block even with verdict pass")
	}
}

func TestHubOneJobPerTargetAndReplay(t *testing.T) {
	h := NewHub()
	h.Linger = time.Second
	release := make(chan struct{})
	var runs int32
	run := func(emit func(Event)) {
		atomic.AddInt32(&runs, 1)
		emit(Event{Type: EventStage, Data: "a"})
		<-release
		emit(Event{Type: EventStage, Data: "b"})
		emit(Event{Type: EventDone, Data: "lec"})
	}

	var wg sync.WaitGroup
	ids := make([]string, 8)
	for i := range ids {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id, _ := h.Start("job-"+string(rune('a'+i)), "c#1@1", run)
			ids[i] = id
		}(i)
	}
	wg.Wait()
	for _, id := range ids[1:] {
		if id != ids[0] {
			t.Fatalf("concurrent starts for one target must share a job: %v", ids)
		}
	}

	// Wait until the first delta is published, then subscribe late.
	deadline := time.Now().Add(time.Second)
	var replay []Event
	var ch <-chan Event
	for time.Now().Before(deadline) {
		var ok bool
		replay, ch, _, ok = h.Subscribe(ids[0])
		if ok && len(replay) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(replay) != 1 {
		t.Fatalf("late subscriber should get replay, got %v", replay)
	}
	close(release)
	var live []Event
	for e := range ch {
		live = append(live, e)
	}
	if len(live) != 2 || live[1].Type != EventDone {
		t.Fatalf("live events=%v", live)
	}
	if atomic.LoadInt32(&runs) != 1 {
		t.Fatalf("generation ran %d times", runs)
	}
	if _, active := h.ActiveJob("c#1@1"); active {
		t.Fatal("target must be released after finish")
	}
	if _, started := h.Start("job-next", "c#1@1", func(func(Event)) {}); !started {
		t.Fatal("a new job may start after the previous one finished")
	}
}

func TestSanitizeResolveOutput(t *testing.T) {
	existing := map[string]bool{"os.process": true, "infra.pod": true}
	ids, p := SanitizeResolveOutput(ResolveOutput{ConceptIDs: []string{"infra.pod", "nope", "infra.pod"}}, existing)
	if len(ids) != 1 || ids[0] != "infra.pod" || p != nil {
		t.Fatalf("ids=%v p=%v", ids, p)
	}
	proposal := &models.ProposedConcept{ID: "infra.pod-eviction", Name: "Pod 축출", Area: "infra-ops", Depth: 2,
		Prerequisites: []string{"infra.pod", "ghost.concept"}, Objectives: []string{"축출 조건을 설명한다", " "}}
	ids, p = SanitizeResolveOutput(ResolveOutput{Proposed: proposal}, existing)
	if p == nil || len(ids) != 0 || len(p.Prerequisites) != 1 || p.Prerequisites[0] != "infra.pod" || len(p.Objectives) != 1 {
		t.Fatalf("proposal should be kept with existing prerequisites only: %+v", p)
	}
	bad := &models.ProposedConcept{ID: "infra.x", Name: "x", Area: "infra-ops", Prerequisites: []string{"ghost"}, Objectives: []string{"a"}}
	if _, p := SanitizeResolveOutput(ResolveOutput{Proposed: bad}, existing); p != nil {
		t.Fatal("proposal without an existing prerequisite must be rejected")
	}
	badArea := &models.ProposedConcept{ID: "x.y", Name: "x", Area: "cooking", Prerequisites: []string{"os.process"}, Objectives: []string{"a"}}
	if _, p := SanitizeResolveOutput(ResolveOutput{Proposed: badArea}, existing); p != nil {
		t.Fatal("unknown area must be rejected")
	}
	dup := &models.ProposedConcept{ID: "OS.Process", Name: "p", Area: "os", Prerequisites: []string{"infra.pod"}, Objectives: []string{"a"}}
	if ids, p := SanitizeResolveOutput(ResolveOutput{Proposed: dup}, existing); p != nil || len(ids) != 1 || ids[0] != "os.process" {
		t.Fatal("proposal of an existing id maps to that concept")
	}
}

func TestSectionForAnchor(t *testing.T) {
	c, err := lecture.ParseMarkup(validMarkup, "c", 1)
	if err != nil {
		t.Fatal(err)
	}
	if s := SectionForAnchor(c, "walkthrough-1"); s.Kind != lecture.SectionWalkthrough {
		t.Fatalf("got %s", s.Kind)
	}
	w := RenderSection(SectionForAnchor(c, "walkthrough"))
	if !strings.Contains(w, "```sql") || !strings.Contains(w, "[예측 질문] 위 쿼리는") || !strings.Contains(w, "[직접 해 보기]") {
		t.Fatalf("code should render fenced, material blocks labeled: %s", w)
	}
	if p := RenderSection(SectionForAnchor(c, "principles")); !strings.Contains(p, `[손계산 표 ] {"caption":"4비트 덧셈"`) {
		t.Fatalf("trace renders its data: %s", p)
	}
}

func TestReviewReaderGapsParsedAndFedBack(t *testing.T) {
	r := ParseReview(`{"verdict":"fail","summary":"s","reader_gaps":[{"section":"judgment_next","quote":"감독자라면",
		"unknown":"감독자(supervisor)","why":"웹 개발자는 systemd 역할을 모름","fix":"한 줄 정의"}],
		"issues":[{"severity":"major","kind":"assumed_knowledge","section":"judgment_next","quote":"감독자라면","message":"m"}]}`)
	if len(r.ReaderGaps) != 1 || !r.Blocking() {
		t.Fatalf("review = %+v", r)
	}
	notes := RevisionNotes(Attempt{Reviewed: true, Review: r})
	if !strings.Contains(notes, "[전제 지식 · judgment_next]") || !strings.Contains(notes, "major/assumed_knowledge") {
		t.Fatalf("notes = %s", notes)
	}
	if ParseReview(`{"verdict":"pass","summary":"ok"}`).ReaderGaps == nil {
		t.Fatal("reader_gaps must default to empty")
	}
}

func TestUnlearnedRefsKeepsOnlyNotDone(t *testing.T) {
	got := UnlearnedRefs([]store.ConceptRef{
		{ID: "a", Name: "A", State: learning.Completed},
		{ID: "b", Name: "B", State: learning.Known},
		{ID: "c", Name: "C", State: learning.InProgress},
		{ID: "d", Name: "D", State: "not_started"},
	})
	if len(got) != 2 || got[0].ID != "c" || got[1].ID != "d" {
		t.Fatalf("unlearned = %+v", got)
	}
}

func TestCatalogRefsSkipsTargetAndNeighbours(t *testing.T) {
	got := CatalogRefs([]store.ConceptRef{{ID: "t"}, {ID: "n"}, {ID: "x", Name: "X", Area: "os", State: learning.Known}},
		"t", []prompts.ConceptRef{{ID: "n"}})
	if len(got) != 1 || got[0].ID != "x" || !got[0].Learned {
		t.Fatalf("catalog = %+v", got)
	}
}

func TestReviewLoopAttemptLogKeepsFailedParse(t *testing.T) {
	f := &fakeSteps{drafts: []string{"@@lecture\ntitle: t\n@@frobnicate\n@@end", validMarkup}, reviews: []ReviewOutcome{pass}}
	res, err := RunReviewLoop(context.Background(), 2, f.funcs())
	if err != nil {
		t.Fatal(err)
	}
	log := res.AttemptLog()
	if len(log) != 2 || log[0].Attempt != 1 || log[0].Result != "parse_error" || !strings.Contains(log[0].Message, "frobnicate") ||
		log[1].Result != "ok" {
		t.Fatalf("attempt log = %+v", log)
	}
}
