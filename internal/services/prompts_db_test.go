package services

import (
	"context"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/forgen-team/punisher/migrations"
)

// dbService is a Service on a fresh SQLite file (migrated, curriculum seeded)
// with no LLM (any LLM call would panic, proving previews/targets make none).
func dbService(t *testing.T) (*Service, string) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "punisher.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx, migrations.Files); err != nil {
		t.Fatal(err)
	}
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SyncCurriculum(ctx, seed); err != nil {
		t.Fatal(err)
	}
	return &Service{Store: st, Keys: GatewayKeys{}, Hub: NewHub()}, "local"
}

// firstConceptWithObjectives returns the first active concept (by id) that has objectives.
func firstConceptWithObjectives(t *testing.T, s *Service) string {
	t.Helper()
	list, err := s.Store.ListConcepts(context.Background(), "local", store.ConceptFilter{Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, c := range list {
		if c.ObjectiveCount > 0 {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		t.Skip("no concept with objectives")
	}
	sort.Strings(ids)
	return ids[0]
}

func TestLibraryIgnoresOverridesInstantUsesThem(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	if _, err := s.Store.SavePromptOverride(ctx, user, prompts.BlockTeacherSystem, "내 강사 원칙", "save"); err != nil {
		t.Fatal(err)
	}
	conceptID := firstConceptWithObjectives(t, s)

	lib, err := s.libraryTarget(ctx, user, conceptID)
	if err != nil {
		t.Fatal(err)
	}
	if lib.Prompts.Overridden(prompts.BlockTeacherSystem) || lib.promptVersion() != prompts.Version || lib.Owner != "" {
		t.Fatalf("library target must use defaults: version=%s owner=%q", lib.promptVersion(), lib.Owner)
	}

	inst, err := s.instantTarget(ctx, user, conceptID, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(prompts.Version) + `\+u:[0-9a-f]{8}$`)
	if !inst.Prompts.Overridden(prompts.BlockTeacherSystem) || !re.MatchString(inst.promptVersion()) || inst.Owner != user {
		t.Fatalf("instant target must use overrides: version=%s", inst.promptVersion())
	}
	if inst.Prompts.Text(prompts.BlockTeacherSystem, nil) != "내 강사 원칙" {
		t.Fatal("override text not used")
	}

	// Reset → back to defaults for the next generation.
	if err := s.Store.ResetPromptOverride(ctx, user, prompts.BlockTeacherSystem); err != nil {
		t.Fatal(err)
	}
	inst, err = s.instantTarget(ctx, user, conceptID, 0, "")
	if err != nil || inst.promptVersion() != prompts.Version {
		t.Fatalf("after reset version = %s, %v", inst.promptVersion(), err)
	}
}

func TestPreviewAssemblesWithoutLLM(t *testing.T) {
	s, user := dbService(t)
	draft := "초안 지시 {{concept_name}} {{nope}}"
	for _, b := range prompts.Blocks {
		var d *string
		if b.ID == prompts.BlockExplainSystem {
			d = &draft
		}
		p, err := s.PreviewPrompt(context.Background(), user, b.ID, d)
		if err != nil {
			t.Fatalf("%s: %v", b.ID, err)
		}
		if strings.TrimSpace(p.System) == "" || strings.TrimSpace(p.User) == "" || p.ConceptName == "" {
			t.Fatalf("%s preview incomplete: %+v", b.ID, p)
		}
		if b.ID == prompts.BlockExplainSystem {
			if !strings.HasPrefix(p.System, "초안 지시 "+p.ConceptName) || len(p.UnknownVars) != 1 || p.UnknownVars[0] != "nope" {
				t.Fatalf("draft not applied: %q %v", p.System, p.UnknownVars)
			}
		}
	}
	if _, err := s.PreviewPrompt(context.Background(), user, "nope", nil); err != ErrUnknownPrompt {
		t.Fatalf("unknown block err = %v", err)
	}
}

// recordingLLM returns a fixed lecture (a passing review to the reviewer) and
// records the system prompts of the drafts.
type recordingLLM struct {
	text    string
	systems []string
}

func (r *recordingLLM) Create(_ context.Context, _ string, req llm.MessagesRequest) (llm.MessagesResult, error) {
	if strings.Contains(req.System, "기술 감수자") {
		return llm.MessagesResult{Text: reviewPass, Model: "fake"}, nil
	}
	r.systems = append(r.systems, req.System)
	return llm.MessagesResult{Text: r.text, Model: "fake"}, nil
}

func (r *recordingLLM) Stream(ctx context.Context, key string, req llm.MessagesRequest, onDelta func(string) error) (llm.MessagesResult, error) {
	res, err := r.Create(ctx, key, req)
	_ = onDelta(res.Text)
	return res, err
}

// TestInstantLectureRecordsUserPromptVersion: a saved instant lecture carries
// Version+"+u:<hash>" and was generated with the user's teacher block.
func TestInstantLectureRecordsUserPromptVersion(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	fake := &recordingLLM{}
	s.LLM = fake
	if _, err := s.Store.SavePromptOverride(ctx, user, prompts.BlockTeacherSystem, "내 강사 원칙(테스트)", "save"); err != nil {
		t.Fatal(err)
	}
	conceptID := firstConceptWithObjectives(t, s)
	tgt, err := s.instantTarget(ctx, user, conceptID, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	fake.text = lectureMarkupFor(tgt.Plan.ObjectiveIDs) // valid for this target → no automatic rewrite
	job, _, err := s.Store.CreateJob(ctx, store.NewJob{
		IdempotencyKey: "test-version:" + user, Kind: "instant", UserID: user, ConceptID: conceptID,
		ObjectiveKey: "test-version:" + user, Depth: tgt.Plan.Depth, Reason: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	s.runInstant("k", user, job.ID, tgt, func(Event) {})

	finished, err := s.Store.GetJob(ctx, user, job.ID)
	if err != nil || finished.LectureID == "" {
		t.Fatalf("job = %+v %v", finished, err)
	}
	saved, err := s.Store.GetLecture(ctx, user, finished.LectureID)
	if err != nil {
		t.Fatalf("saved lecture: %v", err)
	}
	version := saved.PromptVersion
	if want := tgt.Prompts.Version(prompts.InstantBlocks...); version != want || !strings.Contains(version, "+u:") {
		t.Fatalf("prompt_version = %q, want %q", version, want)
	}
	if len(fake.systems) != 1 || fake.systems[0] != "내 강사 원칙(테스트)" {
		t.Fatalf("system prompts = %q", fake.systems)
	}
}

// TestStandaloneLibraryUsesOwnerPrompts: with LibraryUsesUserPrompts the
// library target carries the owner's blocks but stays a library lecture.
func TestStandaloneLibraryUsesOwnerPrompts(t *testing.T) {
	s, user := dbService(t)
	s.Cfg.LibraryUsesUserPrompts = true
	ctx := context.Background()
	conceptID := firstConceptWithObjectives(t, s)
	if _, err := s.Store.SavePromptOverride(ctx, user, prompts.BlockTeacherSystem, "단독판 강사 원칙", "save"); err != nil {
		t.Fatal(err)
	}
	lib, err := s.libraryTarget(ctx, user, conceptID)
	if err != nil {
		t.Fatal(err)
	}
	if !lib.Prompts.Overridden(prompts.BlockTeacherSystem) || lib.Owner != "" || !strings.Contains(lib.promptVersion(), "+u:") {
		t.Fatalf("standalone library target: overridden=%v owner=%q version=%s",
			lib.Prompts.Overridden(prompts.BlockTeacherSystem), lib.Owner, lib.promptVersion())
	}
}

// TestStandaloneLibraryCheckItemsUseOwnerPrompts: the check.items cache key
// follows the same rule as LibraryUsesUserPrompts — a library lecture's item
// set is built with the owner's edited block (keyed to that user like
// witems), and the default block keeps the shared key.
func TestStandaloneLibraryCheckItemsUseOwnerPrompts(t *testing.T) {
	s, user := dbService(t)
	s.Cfg.LibraryUsesUserPrompts = true
	ctx := context.Background()
	if k := s.checkItemKey(ctx, user, models.Lecture{ID: "lib-lecture"}); k.PromptHash != "" || k.CreatedFor != "" {
		t.Fatalf("default block must keep the shared key: %+v", k)
	}
	if _, err := s.Store.SavePromptOverride(ctx, user, prompts.BlockCheckItems, "단독판 출제 원칙", "save"); err != nil {
		t.Fatal(err)
	}
	k := s.checkItemKey(ctx, user, models.Lecture{ID: "lib-lecture"})
	if k.PromptHash == "" || k.CreatedFor != user || k.PromptHash != s.UserPrompts(ctx, user).Hash(prompts.BlockCheckItems) {
		t.Fatalf("edited block must key the set to the owner: %+v", k)
	}
}

// userRecordingLLM answers a library job (outline → draft → review) and records
// every call's user prompt.
type userRecordingLLM struct {
	lecture string
	users   []string
}

func (r *userRecordingLLM) Create(_ context.Context, _ string, req llm.MessagesRequest) (llm.MessagesResult, error) {
	r.users = append(r.users, req.User)
	switch {
	case strings.Contains(req.System, "기술 감수자"):
		return llm.MessagesResult{Text: reviewPass, Model: "fake"}, nil
	case strings.Contains(req.User, "단독판 개요 지시"):
		return llm.MessagesResult{Text: `{"sections":[]}`, Model: "fake"}, nil
	}
	return llm.MessagesResult{Text: r.lecture, Model: "fake"}, nil
}

func (r *userRecordingLLM) Stream(ctx context.Context, key string, req llm.MessagesRequest, onDelta func(string) error) (llm.MessagesResult, error) {
	res, err := r.Create(ctx, key, req)
	_ = onDelta(res.Text)
	return res, err
}

// TestStandaloneLibraryKeepsFactualRules (quality design v2.2 §3.1, 단독판
// LibraryUsesUserPrompts): a library lecture built with the owner's edited
// outline / format / review blocks still carries the fixed factual-accuracy
// rules in every generation prompt — code appends them after the user's
// blocks, so an edit can never drop them.
func TestStandaloneLibraryKeepsFactualRules(t *testing.T) {
	s, user := dbService(t)
	s.Cfg.LibraryUsesUserPrompts = true
	ctx := context.Background()
	for block, text := range map[string]string{
		prompts.BlockOutlineTask:   "단독판 개요 지시",
		prompts.BlockLectureFormat: "단독판 본문 형식",
		prompts.BlockReviewTask:    "단독판 검수 지시",
	} {
		if _, err := s.Store.SavePromptOverride(ctx, user, block, text, "save"); err != nil {
			t.Fatal(err)
		}
	}
	conceptID := firstConceptWithObjectives(t, s)
	lib, err := s.libraryTarget(ctx, user, conceptID)
	if err != nil {
		t.Fatal(err)
	}
	fake := &userRecordingLLM{lecture: lectureMarkupFor(lib.Plan.ObjectiveIDs)}
	s.LLM = fake
	job, _, err := s.Store.CreateJob(ctx, store.NewJob{
		IdempotencyKey: "test-library-rules:" + user, Kind: "library", ConceptID: conceptID,
		ObjectiveKey: lib.Plan.ObjectiveKey, Depth: lib.Plan.Depth, Reason: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.runLibraryJob(ctx, "k", job.ID, lib, nil); err != nil {
		t.Fatal(err)
	}
	if len(fake.users) != 3 {
		t.Fatalf("outline, draft and review expected: %d calls", len(fake.users))
	}
	for i, want := range []string{"단독판 개요 지시", "단독판 본문 형식", "단독판 검수 지시"} {
		u := fake.users[i]
		if !strings.Contains(u, want) {
			t.Errorf("call %d must use the owner's block %q", i, want)
		}
		if !strings.Contains(u, "## 사실 정확성 규칙") || !strings.Contains(u, "자리올림/오버플로") {
			t.Errorf("call %d (%s) lacks the factual-accuracy rules", i, want)
		}
	}
}
