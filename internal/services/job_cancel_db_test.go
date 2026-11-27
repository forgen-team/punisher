package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/claudecli"
	"github.com/forgen-team/punisher/internal/store"
)

// hangingClaude stands in for the claude CLI: a generation call starts a
// sleeping child (the grandchild of the server — like an MCP helper), records
// both pids and then hangs, so only a kill of the whole process group ends it.
const hangingClaude = `#!/bin/sh
case "$1" in
  --help) echo "--system-prompt --tools --setting-sources --strict-mcp-config --mcp-config --no-session-persistence --include-partial-messages"; exit 0;;
  --version) echo "9.9.9 (Claude Code)"; exit 0;;
esac
cat > /dev/null
sleep 300 &
echo "$$ $!" > "__PIDS__.tmp" && mv "__PIDS__.tmp" "__PIDS__"
sleep 300
`

// cancelFixture is a service whose AI is the real claudecli client running hangingClaude.
type cancelFixture struct {
	s         *Service
	user      string
	conceptID string
	pids      string
}

func newCancelFixture(t *testing.T) *cancelFixture {
	t.Helper()
	s, user := dbService(t)
	dir := t.TempDir()
	f := &cancelFixture{s: s, user: user, pids: filepath.Join(dir, "pids")}
	bin := filepath.Join(dir, "claude")
	if err := os.WriteFile(bin, []byte(strings.ReplaceAll(hangingClaude, "__PIDS__", f.pids)), 0o755); err != nil {
		t.Fatal(err)
	}
	s.LLM = claudecli.New(claudecli.Config{Bin: bin, TempBase: filepath.Join(dir, "tmp"), Concurrency: 2,
		DefaultTimeout: 2 * time.Minute, MaxTimeout: 10 * time.Minute})
	if err := testDB(s).QueryRowContext(context.Background(), `
		SELECT c.id FROM concepts c WHERE c.status = 'active' AND c.owner_user_id IS NULL
		  AND EXISTS (SELECT 1 FROM objectives o WHERE o.concept_id = c.id AND o.status = 'active' AND o.depth = 1)
		ORDER BY c.id LIMIT 1`).Scan(&f.conceptID); err != nil {
		t.Skip(err)
	}
	return f
}

// start starts an instant lecture and waits until the fake CLI and its child run.
func (f *cancelFixture) start(t *testing.T) (InstantResponse, []int) {
	t.Helper()
	_ = os.Remove(f.pids)
	resp, err := f.s.StartInstant(context.Background(), f.user, InstantRequest{ConceptID: f.conceptID, Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != InstantStarted {
		t.Fatalf("status %q, want started", resp.Status)
	}
	var pids []int
	waitFor(t, "the fake CLI to start its child", func() bool {
		raw, err := os.ReadFile(f.pids)
		if err != nil {
			return false
		}
		pids = nil
		for _, field := range strings.Fields(string(raw)) {
			pid, err := strconv.Atoi(field)
			if err != nil {
				t.Fatal(err)
			}
			pids = append(pids, pid)
		}
		return len(pids) == 2
	})
	return resp, pids
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func alive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// drain reads the job's stream until it ends and returns its events.
func drain(t *testing.T, h *Hub, jobID string) []Event {
	t.Helper()
	replay, ch, cancel, ok := h.Subscribe(jobID)
	defer cancel()
	if !ok {
		t.Fatalf("job %s not in the hub", jobID)
	}
	events := append([]Event(nil), replay...)
	timeout := time.After(15 * time.Second)
	for {
		select {
		case e, open := <-ch:
			if !open {
				return events
			}
			events = append(events, e)
		case <-timeout:
			t.Fatal("the cancelled job's stream did not end")
		}
	}
}

func TestCancelInstantKillsCLIProcessGroup(t *testing.T) {
	f := newCancelFixture(t)
	ctx := context.Background()
	resp, pids := f.start(t)
	for _, pid := range pids {
		if !alive(pid) {
			t.Fatalf("pid %d should run before the cancel", pid)
		}
	}

	job, err := f.s.CancelJob(ctx, f.user, resp.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "failed" || job.Reason != store.JobReasonCancelled {
		t.Fatalf("cancelled job: status %q reason %q", job.Status, job.Reason)
	}

	// The CLI and the sleeping child it spawned both end (process group killed).
	waitFor(t, "the CLI process group to die", func() bool { return !alive(pids[0]) && !alive(pids[1]) })

	// The stream ends with the cancel reason; the runner finishing does not overwrite the row.
	events := drain(t, f.s.Hub, resp.JobID)
	last := events[len(events)-1]
	if last.Type != EventError || last.Data.(map[string]string)["code"] != store.JobReasonCancelled {
		t.Fatalf("last event %+v", last)
	}
	stage := events[len(events)-2].Data.(StageEvent)
	if stage.Stage != StageFailed || len(stage.Reasons) != 1 || stage.Reasons[0] != ReasonCancelled {
		t.Fatalf("failed stage %+v", stage)
	}
	after, err := f.s.Store.GetJob(ctx, f.user, resp.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != "failed" || after.Reason != store.JobReasonCancelled || after.LectureID != "" {
		t.Fatalf("job after the runner ended: %+v", after)
	}
	var lectures int
	if err := testDB(f.s).QueryRowContext(ctx, `SELECT COUNT(*) FROM lectures WHERE generation_job_id = ?`, resp.JobID).Scan(&lectures); err != nil {
		t.Fatal(err)
	}
	if lectures != 0 {
		t.Fatalf("a cancelled job saved %d lectures", lectures)
	}
	final := InstantFinalEvents(after)
	if final[0].Data.(StageEvent).Reasons[0] != ReasonCancelled {
		t.Fatalf("final events of a cancelled job: %+v", final)
	}

	// A finished job cannot be cancelled again (HTTP 409).
	if _, err := f.s.CancelJob(ctx, f.user, resp.JobID); !errors.Is(err, ErrJobNotActive) {
		t.Fatalf("second cancel: %v, want ErrJobNotActive", err)
	}
	// Another user's job is not found (HTTP 404).
	if _, err := f.s.CancelJob(ctx, "someone-else", resp.JobID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign cancel: %v, want ErrNotFound", err)
	}
}

func TestCancelledInstantCanStartAgain(t *testing.T) {
	f := newCancelFixture(t)
	ctx := context.Background()
	first, _ := f.start(t)
	if _, err := f.s.CancelJob(ctx, f.user, first.JobID); err != nil {
		t.Fatal(err)
	}
	// Right after the cancel (the old runner may still be winding down) the
	// same concept starts a new job instead of attaching to the cancelled one.
	second, pids := f.start(t)
	if second.JobID == first.JobID {
		t.Fatal("the restart attached to the cancelled job")
	}
	if _, err := f.s.CancelJob(ctx, f.user, second.JobID); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the second CLI to die", func() bool { return !alive(pids[0]) && !alive(pids[1]) })
	drain(t, f.s.Hub, first.JobID)
	drain(t, f.s.Hub, second.JobID)
}

func TestCancelAfterCommitIsTooLate(t *testing.T) {
	var r jobRuns
	ctx, end := r.begin(context.Background(), "j1")
	defer end()
	if !r.commit("j1") {
		t.Fatal("an uncancelled job must commit")
	}
	if _, err := r.mark("j1"); !errors.Is(err, ErrJobNotActive) {
		t.Fatalf("cancel while saving: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("a committing job's context must stay alive")
	}

	ctx2, end2 := r.begin(context.Background(), "j2")
	defer end2()
	if found, err := r.mark("j2"); !found || err != nil {
		t.Fatalf("mark: %v %v", found, err)
	}
	if r.commit("j2") {
		t.Fatal("a cancelled job must not save its result")
	}
	r.cancelRun("j2")
	if !errors.Is(context.Cause(ctx2), ErrJobCancelled) {
		t.Fatalf("cause %v", context.Cause(ctx2))
	}
}

func TestFinishJobKeepsCancelledState(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	job, _, err := s.Store.CreateJob(ctx, store.NewJob{IdempotencyKey: "cancel-keep", Kind: "instant", UserID: user,
		ObjectiveKey: "cancel-keep", Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := s.Store.CancelJob(ctx, user, job.ID); !ok || err != nil {
		t.Fatalf("cancel: %v %v", ok, err)
	}
	if err := s.Store.FinishJob(ctx, job.ID, store.JobOutcome{Status: "failed", Reason: JobReasonLLMError, Error: "context canceled",
		InputTokens: 7, OutputTokens: 3}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Store.GetJob(ctx, user, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "failed" || got.Reason != store.JobReasonCancelled || got.Error != "" || got.InputTokens != 7 || got.OutputTokens != 3 {
		t.Fatalf("cancelled row after FinishJob: %+v", got)
	}
	if ok, err := s.Store.CancelJob(ctx, user, job.ID); ok || err != nil {
		t.Fatalf("second cancel: %v %v", ok, err)
	}
}
