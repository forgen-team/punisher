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
)

const validMarkup = `@@lecture
title: t
objectives: c#1
new_scope: x
@@section situation_question minutes=3 title=a
상황
@@section principles minutes=12 title=b
원리
@@sources
- [doc](https://example.org/doc)
@@section walkthrough minutes=10 title=c
@@code lang=bash
ls
@@sources
- [doc](https://example.org/doc2)
@@section judgment_next minutes=5 title=d
판단
@@end`

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
	invalid := strings.Replace(validMarkup, "minutes=12", "minutes=1", 1) // total 19 → schema issue
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
		emit(Event{Type: EventDelta, Data: "a"})
		<-release
		emit(Event{Type: EventDelta, Data: "b"})
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
	if !strings.Contains(RenderSection(SectionForAnchor(c, "walkthrough")), "```bash") {
		t.Fatal("code should render fenced")
	}
}
