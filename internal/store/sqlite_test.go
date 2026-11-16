package store

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/migrations"
)

const testUser = "local"

// newTestStore opens a fresh SQLite file in t.TempDir(), applies the
// migrations and syncs the embedded curriculum seed.
func newTestStore(t *testing.T) (*SQLiteStore, *curriculum.Seed) {
	t.Helper()
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "punisher.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(ctx, migrations.Files); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		t.Fatalf("load seed: %v", err)
	}
	applied, err := s.SyncCurriculum(ctx, seed)
	if err != nil || !applied {
		t.Fatalf("sync curriculum: applied=%v err=%v", applied, err)
	}
	return s, seed
}

// firstConcept returns an active seed concept with at least one objective.
func firstConcept(t *testing.T, s *SQLiteStore) models.ConceptInfo {
	t.Helper()
	ctx := context.Background()
	list, err := s.ListConcepts(ctx, testUser, ConceptFilter{Status: "active"})
	if err != nil || len(list) == 0 {
		t.Fatalf("list concepts: %d %v", len(list), err)
	}
	for _, c := range list {
		if c.ObjectiveCount > 0 {
			info, err := s.ConceptInfo(ctx, testUser, c.ID)
			if err != nil {
				t.Fatalf("concept info: %v", err)
			}
			return info
		}
	}
	t.Fatal("no concept with objectives")
	return models.ConceptInfo{}
}

func insertTestLecture(t *testing.T, s *SQLiteStore, c models.ConceptInfo, kind, status string) models.Lecture {
	t.Helper()
	ids := []string{c.Objectives[0].ID}
	l, err := s.InsertLecture(context.Background(), NewLecture{
		ConceptID: c.ID, ObjectiveIDs: ids, ObjectiveKey: learning.ObjectiveKey(ids), Depth: c.Objectives[0].Depth,
		Kind: kind, Status: status, Title: "테스트 강의", Content: json.RawMessage(`{"sections":[]}`),
		RequestedBy: testUser, EstimatedMinutes: 7,
	})
	if err != nil {
		t.Fatalf("insert lecture: %v", err)
	}
	return l
}

func TestMigrateIsIdempotentAndSeedSyncsOnce(t *testing.T) {
	s, seed := newTestStore(t)
	ctx := context.Background()
	if err := s.Migrate(ctx, migrations.Files); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	applied, err := s.SyncCurriculum(ctx, seed)
	if err != nil || applied {
		t.Fatalf("re-sync same seed: applied=%v err=%v (want no-op)", applied, err)
	}
	version, err := s.LatestCurriculumVersion(ctx)
	if err != nil || version != seed.Version {
		t.Fatalf("version = %q %v, want %q", version, err, seed.Version)
	}
	all, err := s.ListConcepts(ctx, testUser, ConceptFilter{Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != len(seed.Concepts) {
		t.Fatalf("active concepts = %d, want %d", len(all), len(seed.Concepts))
	}
	graph, err := s.LoadGraph(ctx, testUser)
	if err != nil || len(graph.Edges) == 0 {
		t.Fatalf("graph edges = %v %v", graph, err)
	}
	if err := s.Ping(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestSeedRetiresRemovedConcepts(t *testing.T) {
	s, seed := newTestStore(t)
	ctx := context.Background()
	trimmed := *seed
	removed := trimmed.Concepts[len(trimmed.Concepts)-1].ID
	trimmed.Concepts = trimmed.Concepts[:len(trimmed.Concepts)-1]
	var edges []curriculum.Edge
	for _, e := range trimmed.Edges {
		if e.From != removed && e.To != removed {
			edges = append(edges, e)
		}
	}
	trimmed.Edges = edges
	var aliases []curriculum.Alias
	for _, a := range trimmed.Aliases {
		if a.ConceptID != removed {
			aliases = append(aliases, a)
		}
	}
	trimmed.Aliases = aliases
	trimmed.Version = "test-trimmed"
	if applied, err := s.SyncCurriculum(ctx, &trimmed); err != nil || !applied {
		t.Fatalf("sync trimmed: %v %v", applied, err)
	}
	d, err := s.GetConceptDetail(ctx, testUser, removed)
	if err != nil {
		t.Fatalf("detail of retired: %v", err)
	}
	if d.Status != "retired" {
		t.Fatalf("status = %s, want retired", d.Status)
	}
	if _, err := s.SetConceptState(ctx, testUser, removed, learning.Known); !errors.Is(err, ErrConceptRetired) {
		t.Fatalf("set state on retired = %v, want ErrConceptRetired", err)
	}
}

func TestConceptDetailStateAndResolve(t *testing.T) {
	s, seed := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)

	d, err := s.GetConceptDetail(ctx, testUser, c.ID)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if d.UserState != learning.NotStarted || len(d.Objectives) != len(c.Objectives) || len(d.Aliases) == 0 {
		t.Fatalf("unexpected detail: state=%s objectives=%d aliases=%d", d.UserState, len(d.Objectives), len(d.Aliases))
	}
	if _, err := s.GetConceptDetail(ctx, testUser, "no-such-concept"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown concept = %v", err)
	}

	state, err := s.SetConceptState(ctx, testUser, c.ID, learning.Known)
	if err != nil || state != learning.Known {
		t.Fatalf("set known: %s %v", state, err)
	}
	done, err := s.DoneSet(ctx, testUser, c.ID)
	if err != nil || len(done) != len(c.Objectives) {
		t.Fatalf("done set = %d %v, want %d", len(done), err, len(c.Objectives))
	}
	state, err = s.SetConceptState(ctx, testUser, c.ID, learning.NotStarted)
	if err != nil || state != learning.NotStarted {
		t.Fatalf("reset: %s %v", state, err)
	}
	if done, _ := s.DoneSet(ctx, testUser, c.ID); len(done) != 0 {
		t.Fatalf("reset kept %d objective states", len(done))
	}

	alias := seed.Aliases[0]
	ids, err := s.ResolveConcepts(ctx, testUser, alias.Alias)
	if err != nil || len(ids) == 0 || ids[0] != alias.ConceptID {
		t.Fatalf("resolve %q = %v %v, want %s", alias.Alias, ids, err, alias.ConceptID)
	}
	idx, err := s.AliasIndex(ctx, testUser)
	if err != nil || idx[alias.Alias] != alias.ConceptID {
		t.Fatalf("alias index = %v", err)
	}
	if _, err := s.ConceptCatalog(ctx, testUser); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RelatedConcepts(ctx, testUser, c.ID); err != nil {
		t.Fatal(err)
	}
	known, err := s.KnownConceptIDs(ctx, testUser)
	if err != nil || !known[c.ID] {
		t.Fatalf("known ids: %v", err)
	}
}

func TestLectureVersionsCoveringAndComplete(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)

	v1 := insertTestLecture(t, s, c, "library", "published")
	v2 := insertTestLecture(t, s, c, "library", "published")
	if v1.Version != 1 || v2.Version != 2 {
		t.Fatalf("versions = %d, %d, want 1, 2", v1.Version, v2.Version)
	}
	if v1.PublishedAt == nil || len(v1.ObjectiveIDs) != 1 || string(v1.Content) != `{"sections":[]}` {
		t.Fatalf("lecture round trip: %+v", v1)
	}

	got, err := s.PublishedLectureCovering(ctx, testUser, c.ID, v1.Depth, v1.ObjectiveIDs)
	if err != nil || got.ID != v2.ID {
		t.Fatalf("covering = %v %v, want v2", got.ID, err)
	}
	if _, err := s.PublishedLectureCovering(ctx, testUser, c.ID, v1.Depth, []string{"nope#1"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("covering missing objective = %v", err)
	}
	latest, err := s.LatestLectureForTarget(ctx, "library", c.ID, v1.ObjectiveKey, v1.Depth, []string{"published"})
	if err != nil || latest.ID != v2.ID {
		t.Fatalf("latest = %v %v", latest.ID, err)
	}
	if err := s.InsertReview(ctx, v2.ID, models.LectureReview{Verdict: "pass", Reviewer: "llm", Attempt: 1}); err != nil {
		t.Fatal(err)
	}

	list, err := s.ListConceptLectures(ctx, testUser, c.ID)
	if err != nil || len(list) != 1 || list[0].ID != v2.ID || list[0].ProgressState != "not_started" {
		t.Fatalf("concept lectures = %+v %v (want only newest)", list, err)
	}

	if err := s.UpdateProgress(ctx, testUser, v2.ID, "intro", 40); err != nil {
		t.Fatal(err)
	}
	lec, err := s.GetLecture(ctx, testUser, v2.ID)
	if err != nil || lec.Progress == nil || lec.Progress.Percent != 40 || len(lec.Reviews) != 1 {
		t.Fatalf("get lecture = %+v %v", lec.Progress, err)
	}

	res, err := s.CompleteLecture(ctx, testUser, v2.ID)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	wantState := learning.Completed
	if len(c.Objectives) > 1 {
		wantState = learning.InProgress
	}
	if res.ConceptState != wantState || len(res.ObjectivesCompleted) != 1 {
		t.Fatalf("complete result = %+v, want state %s", res, wantState)
	}
	// Completed lecture progress never goes backwards.
	if err := s.UpdateProgress(ctx, testUser, v2.ID, "intro", 10); err != nil {
		t.Fatal(err)
	}
	lec, _ = s.GetLecture(ctx, testUser, v2.ID)
	if lec.Progress.Percent != 100 || lec.Progress.CompletedAt == nil {
		t.Fatalf("progress after complete = %+v", lec.Progress)
	}
	done, _, err := s.ListMyLectures(ctx, testUser, "completed", 10, "")
	if err != nil || len(done) != 1 || done[0].ID != v2.ID {
		t.Fatalf("completed lectures = %+v %v", done, err)
	}

	if _, err := s.InsertFeedback(ctx, testUser, v2.ID, "understood", "", "좋아요", ""); err != nil {
		t.Fatal(err)
	}
	if fb, err := s.LatestFeedback(ctx, testUser); err != nil || fb != "understood" {
		t.Fatalf("latest feedback = %q %v", fb, err)
	}

	instant := insertTestLecture(t, s, c, "instant", "published")
	if _, err := s.GetLecture(ctx, "someone-else", instant.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("instant lecture visible to other user: %v", err)
	}
}

func TestMyLecturesPagination(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)
	for i := 0; i < 3; i++ {
		l := insertTestLecture(t, s, c, "library", "published")
		if err := s.UpdateProgress(ctx, testUser, l.ID, "", 10); err != nil {
			t.Fatal(err)
		}
		time.Sleep(3 * time.Millisecond) // distinct last_seen_at (ms precision)
	}
	page1, next, err := s.ListMyLectures(ctx, testUser, "in_progress", 2, "")
	if err != nil || len(page1) != 2 || next == "" {
		t.Fatalf("page1 = %d next=%q %v", len(page1), next, err)
	}
	page2, next2, err := s.ListMyLectures(ctx, testUser, "in_progress", 2, next)
	if err != nil || len(page2) != 1 || next2 != "" {
		t.Fatalf("page2 = %d next=%q %v", len(page2), next2, err)
	}
	seen := map[string]bool{}
	for _, l := range append(page1, page2...) {
		if seen[l.ID] {
			t.Fatalf("duplicate across pages: %s", l.ID)
		}
		seen[l.ID] = true
	}
	if !page1[0].LastSeenAt.After(page2[0].LastSeenAt) {
		t.Fatalf("not newest first")
	}
	if _, _, err := s.ListMyLectures(ctx, testUser, "in_progress", 2, "!!!"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("bad cursor = %v", err)
	}
}

func TestDismissExcludesFromResumeAndRestores(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)
	lec := insertTestLecture(t, s, c, "instant", "published")

	if err := s.UpdateProgress(ctx, testUser, lec.ID, "principles", 30); err != nil {
		t.Fatalf("progress: %v", err)
	}
	assertResume(t, s, lec.ID, true)

	if err := s.DismissLecture(ctx, testUser, lec.ID); err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	assertResume(t, s, lec.ID, false)
	d, _ := s.GetConceptDetail(ctx, testUser, c.ID)
	if d.UserState != learning.NotStarted {
		t.Fatalf("concept state after dismiss = %s, want not_started", d.UserState)
	}
	dismissed, err := s.DismissedConceptIDs(ctx, testUser)
	if err != nil || !dismissed[c.ID] {
		t.Fatalf("dismissed concepts = %v, %v", dismissed, err)
	}
	if _, err := s.GetLecture(ctx, testUser, lec.ID); err != nil {
		t.Fatalf("lecture deleted by dismiss: %v", err)
	}

	if err := s.RestoreLecture(ctx, testUser, lec.ID); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertResume(t, s, lec.ID, true)

	if err := s.DismissLecture(ctx, testUser, lec.ID); err != nil {
		t.Fatalf("dismiss again: %v", err)
	}
	if err := s.UpdateProgress(ctx, testUser, lec.ID, "walkthrough", 60); err != nil {
		t.Fatalf("progress again: %v", err)
	}
	assertResume(t, s, lec.ID, true)

	if err := s.DismissLecture(ctx, testUser, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("dismiss unknown = %v, want ErrNotFound", err)
	}
}

func assertResume(t *testing.T, s *SQLiteStore, lectureID string, want bool) {
	t.Helper()
	ctx := context.Background()
	d, err := s.LoadToday(ctx, testUser, time.Now())
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	inToday := false
	for _, r := range d.Resume {
		inToday = inToday || r.LectureID == lectureID
	}
	list, _, err := s.ListMyLectures(ctx, testUser, "in_progress", 20, "")
	if err != nil {
		t.Fatalf("my lectures: %v", err)
	}
	inList := false
	for _, l := range list {
		inList = inList || l.ID == lectureID
	}
	if inToday != want || inList != want {
		t.Fatalf("resume visible: today=%v list=%v, want %v", inToday, inList, want)
	}
}

func TestJobsIdempotencyActiveTargetAndUsage(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)
	key := learning.ObjectiveKey([]string{c.Objectives[0].ID})

	job, created, err := s.CreateJob(ctx, NewJob{IdempotencyKey: "k1", Kind: "instant", UserID: testUser,
		ConceptID: c.ID, ObjectiveKey: key, Depth: 1, Input: json.RawMessage(`{"a":1}`)})
	if err != nil || !created || job.Status != "pending" || string(job.Input) != `{"a":1}` {
		t.Fatalf("create job = %+v %v %v", job, created, err)
	}
	again, created, err := s.CreateJob(ctx, NewJob{IdempotencyKey: "k1", Kind: "instant", UserID: testUser, ObjectiveKey: key, Depth: 1})
	if err != nil || created || again.ID != job.ID {
		t.Fatalf("idempotent create = %v %v %v", again.ID, created, err)
	}
	active, _, err := s.CreateJob(ctx, NewJob{IdempotencyKey: "k2", Kind: "library", UserID: testUser, ObjectiveKey: key, Depth: 1})
	if !errors.Is(err, ErrJobActive) || active.ID != job.ID {
		t.Fatalf("second active job = %v %v, want ErrJobActive with %s", active.ID, err, job.ID)
	}

	if err := s.StartJob(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	lec := insertTestLecture(t, s, c, "instant", "published")
	if err := s.FinishJob(ctx, job.ID, JobOutcome{Status: "succeeded", LectureID: lec.ID, Model: "sonnet", InputTokens: 100, OutputTokens: 50}); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetJob(ctx, testUser, job.ID)
	if err != nil || got.Status != "succeeded" || got.LectureID != lec.ID || got.Attempts != 1 || got.FinishedAt == nil {
		t.Fatalf("finished job = %+v %v", got, err)
	}
	if _, err := s.GetJob(ctx, "other", job.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("job visible to other user: %v", err)
	}
	// The target is free again once the job is finished.
	if _, created, err := s.CreateJob(ctx, NewJob{IdempotencyKey: "k3", Kind: "library", UserID: testUser, ObjectiveKey: key, Depth: 1}); err != nil || !created {
		t.Fatalf("job after finish = %v %v", created, err)
	}

	if err := s.RecordUsage(ctx, testUser, "question", c.ID, JobOutcome{Status: "succeeded", InputTokens: 7, OutputTokens: 3}); err != nil {
		t.Fatal(err)
	}
	lectures, tokens, err := s.UsageSince(ctx, testUser, time.Now().Add(-time.Hour))
	if err != nil || lectures != 2 || tokens != 160 {
		t.Fatalf("usage = %d lectures %d tokens %v, want 2/160", lectures, tokens, err)
	}
	if n, err := s.FailStaleJobs(ctx, -time.Minute); err != nil || n != 1 {
		t.Fatalf("fail stale = %d %v, want 1 (k3)", n, err)
	}
}

func TestStudyRequestsResolutionAndMap(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)

	req := models.StudyRequest{UserID: testUser, Source: "manual", Query: "새 개념", Status: "unresolved",
		IdempotencyKey: "q1", Context: json.RawMessage(`{}`)}
	saved, created, err := s.CreateStudyRequest(ctx, req)
	if err != nil || !created || len(saved.ResolvedConceptIDs) != 0 {
		t.Fatalf("create request = %+v %v %v", saved, created, err)
	}
	if again, created, err := s.CreateStudyRequest(ctx, req); err != nil || created || again.ID != saved.ID {
		t.Fatalf("idempotent request = %v %v", created, err)
	}
	pending, err := s.UnresolvedStudyRequests(ctx, testUser, 3, 10)
	if err != nil || len(pending) != 1 {
		t.Fatalf("unresolved = %d %v", len(pending), err)
	}

	proposed := models.ProposedConcept{ID: "test-new-concept", Name: "새 개념", Area: c.Area, Depth: 1,
		Summary: "요약", Aliases: []string{"New Concept"}, Prerequisites: []string{c.ID}, Objectives: []string{"설명할 수 있다"}}
	if _, err := s.ProposeConcept(ctx, testUser, proposed); err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := s.ProposeConcept(ctx, testUser, proposed); err != nil {
		t.Fatalf("propose again (idempotent): %v", err)
	}
	if err := s.RecordResolution(ctx, saved.ID, []string{proposed.ID}, proposed.ID, ""); err != nil {
		t.Fatal(err)
	}
	r, err := s.GetStudyRequest(ctx, testUser, saved.ID)
	if err != nil || r.Status != "resolved" || r.StartingConceptID != proposed.ID || r.ResolvedAt == nil || r.ResolveAttempts != 1 || r.Learned {
		t.Fatalf("resolved request = %+v %v", r, err)
	}

	graph, err := s.LoadGraph(ctx, testUser)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.LoadMapExtras(ctx, testUser, graph); err != nil {
		t.Fatalf("map extras: %v", err)
	}
	foundNode, foundEdge := false, false
	for _, n := range graph.Concepts {
		foundNode = foundNode || n.ID == proposed.ID
	}
	for _, e := range graph.Edges {
		foundEdge = foundEdge || (e.From == c.ID && e.To == proposed.ID)
	}
	if !foundNode || !foundEdge || len(graph.Requests) != 1 {
		t.Fatalf("map: node=%v edge=%v requests=%d", foundNode, foundEdge, len(graph.Requests))
	}

	today, err := s.LoadToday(ctx, testUser, time.Now())
	if err != nil || len(today.Requests) != 1 || !today.Concepts[proposed.ID].Proposed || !today.Concepts[proposed.ID].HasPending {
		t.Fatalf("today = %+v %v", today.Requests, err)
	}

	if _, err := s.SetConceptState(ctx, testUser, proposed.ID, learning.Known); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListStudyRequests(ctx, testUser, "", 10)
	if err != nil || len(list) != 1 || !list[0].Learned {
		t.Fatalf("learned flag = %+v %v", list, err)
	}

	if err := s.SaveGloss(ctx, c.ID, "짧은 설명", "haiku"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveGloss(ctx, c.ID, "덮어쓰기 안 됨", "haiku"); err != nil {
		t.Fatal(err)
	}
	if g, err := s.GetGloss(ctx, c.ID); err != nil || g != "짧은 설명" {
		t.Fatalf("gloss = %q %v", g, err)
	}
}

func TestNotesGoalsQuestionsSettings(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)
	lec := insertTestLecture(t, s, c, "library", "published")

	n, err := s.CreateNote(ctx, testUser, models.Note{LectureID: lec.ID, ConceptID: c.ID, Anchor: "a1", Kind: "note", Body: "메모"})
	if err != nil || n.ID == "" || n.CreatedAt.IsZero() {
		t.Fatalf("create note = %+v %v", n, err)
	}
	body := "고친 메모"
	if n, err = s.UpdateNote(ctx, testUser, n.ID, nil, nil, &body); err != nil || n.Body != body || n.Anchor != "a1" {
		t.Fatalf("update note = %+v %v", n, err)
	}
	if notes, err := s.ListNotes(ctx, testUser, lec.ID, ""); err != nil || len(notes) != 1 {
		t.Fatalf("list notes = %d %v", len(notes), err)
	}
	if err := s.DeleteNote(ctx, testUser, n.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteNote(ctx, testUser, n.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete twice = %v", err)
	}

	g, err := s.CreateGoal(ctx, testUser, "목표", []string{c.ID})
	if err != nil || g.Status != "active" || len(g.TargetConceptIDs) != 1 {
		t.Fatalf("create goal = %+v %v", g, err)
	}
	status := "achieved"
	if g, err = s.UpdateGoal(ctx, testUser, g.ID, nil, nil, &status); err != nil || g.Status != status || len(g.TargetConceptIDs) != 1 {
		t.Fatalf("update goal = %+v %v", g, err)
	}
	if goals, err := s.ListGoals(ctx, testUser, ""); err != nil || len(goals) != 1 {
		t.Fatalf("list goals = %d %v", len(goals), err)
	}
	if err := s.DeleteGoal(ctx, testUser, g.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetGoal(ctx, testUser, g.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted goal = %v", err)
	}

	q, err := s.CreateQuestion(ctx, testUser, models.Question{LectureID: lec.ID, Kind: "question", Question: "왜?"})
	if err != nil || q.Status != "pending" || q.LinkedConceptIDs == nil {
		t.Fatalf("create question = %+v %v", q, err)
	}
	q, err = s.AnswerQuestion(ctx, q.ID, "그래서", "sonnet", []string{c.ID}, nil)
	if err != nil || q.Status != "answered" || q.AnsweredAt == nil || len(q.LinkedConceptIDs) != 1 || string(q.ProposedConcepts) != "[]" {
		t.Fatalf("answer = %+v %v", q, err)
	}
	failed, err := s.CreateQuestion(ctx, testUser, models.Question{LectureID: lec.ID, Kind: "alt_explanation", Question: "다르게"})
	if err != nil {
		t.Fatal(err)
	}
	if failed, err = s.FailQuestion(ctx, failed.ID, "sonnet", "boom"); err != nil || failed.Status != "failed" {
		t.Fatalf("fail question = %+v %v", failed, err)
	}
	if list, err := s.ListQuestions(ctx, testUser, lec.ID); err != nil || len(list) != 2 {
		t.Fatalf("list questions = %d %v", len(list), err)
	}
	mine, next, err := s.ListMyQuestions(ctx, testUser, c.ID, 10, "")
	if err != nil || len(mine) != 2 || next != "" || mine[0].ConceptName == "" || mine[0].LectureTitle == "" {
		t.Fatalf("my questions = %+v %v", mine, err)
	}

	st, err := s.GetLearningSettings(ctx, testUser)
	if err != nil || st.ExampleLanguage != learning.DefaultExampleLanguage || st.UpdatedAt != nil {
		t.Fatalf("default settings = %+v %v", st, err)
	}
	if _, err := s.SaveLearningSettings(ctx, testUser, "python", 0); err != nil {
		t.Fatal(err)
	}
	if lang := s.ExampleLanguageFor(ctx, testUser); lang != "python" {
		t.Fatalf("example language = %q", lang)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 3, 4, 5, 6, 7, 123000000, time.UTC)
	got, err := decodeCursor(encodeCursor(at, "abc"))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.At.Equal(at) || got.ID != "abc" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestDecodeCursorRejectsGarbage(t *testing.T) {
	for _, in := range []string{"!!!", "bm9wZQ", "Zm9vfGJhcg"} {
		if _, err := decodeCursor(in); !errors.Is(err, ErrInvalidCursor) {
			t.Errorf("decodeCursor(%q) err = %v, want ErrInvalidCursor", in, err)
		}
	}
}

func TestRewritePlaceholdersAndNow(t *testing.T) {
	got := rewrite("SELECT $2, $1, $10 WHERE t < NOW()")
	want := "SELECT ?2, ?1, ?10 WHERE t < " + nowExpr
	if got != want {
		t.Fatalf("rewrite = %q, want %q", got, want)
	}
}

func TestNewIDIsUUIDv4(t *testing.T) {
	id := NewID()
	if len(id) != 36 || id[14] != '4' || id == NewID() {
		t.Fatalf("bad uuid %q", id)
	}
}
