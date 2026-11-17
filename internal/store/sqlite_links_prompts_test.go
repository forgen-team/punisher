package store

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/migrations"
)

// rawLecture inserts a published instant lecture owned by user (content may be "").
func rawLecture(t *testing.T, s *SQLiteStore, user, conceptID, title, content string) string {
	t.Helper()
	if content == "" {
		content = "{}"
	}
	id := NewID()
	if _, err := s.c().exec(context.Background(), `
		INSERT INTO lectures (id, concept_id, objective_key, depth, kind, status, title, requested_by, content)
		VALUES ($1, $2, $3, 1, 'instant', 'published', $4, $5, $6)`,
		id, conceptID, "test:"+user+":"+title, title, user, content); err != nil {
		t.Fatalf("insert lecture: %v", err)
	}
	return id
}

func twoConcepts(t *testing.T, s *SQLiteStore) (string, string) {
	t.Helper()
	ids, err := collect(s.c().query(context.Background(), `SELECT id FROM concepts WHERE status = 'active' ORDER BY id LIMIT 2`), scanString)
	if err != nil || len(ids) < 2 {
		t.Fatalf("need two seeded concepts: %v %v", ids, err)
	}
	return ids[0], ids[1]
}

func TestDeeperLinksDirectAndViaJob(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	cA, cB := twoConcepts(t, s)
	parent := rawLecture(t, s, user, cA, "원 강의", "")
	child := rawLecture(t, s, user, cB, "더 깊게 1", "")

	link := DeeperLink{UserID: user, ParentLecture: parent, ChildLectureID: child, ChildConceptID: cB, Anchor: "principles-2"}
	for i := 0; i < 2; i++ { // second call is a no-op (no duplicate)
		if err := s.LinkDeeper(ctx, link); err != nil {
			t.Fatalf("link: %v", err)
		}
	}
	if err := s.LinkDeeper(ctx, DeeperLink{UserID: user, ParentLecture: parent, ChildLectureID: parent, ChildConceptID: cA}); err != nil {
		t.Fatalf("self link should be a no-op: %v", err)
	}
	if err := s.LinkDeeper(ctx, DeeperLink{UserID: "someone-else", ParentLecture: parent, ChildLectureID: child, ChildConceptID: cB}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unreadable parent err = %v, want ErrNotFound", err)
	}

	time.Sleep(3 * time.Millisecond) // links are ordered by link time (ms precision)
	job, _, err := s.CreateJob(ctx, NewJob{IdempotencyKey: "test-link", Kind: "question", UserID: user, ConceptID: cB})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.LinkDeeper(ctx, DeeperLink{UserID: user, ParentLecture: parent, ChildJobID: job.ID, ChildConceptID: cB, Kind: LinkRegeneratedKind}); err != nil {
		t.Fatalf("link job: %v", err)
	}
	_, children, err := s.LectureLinks(ctx, user, parent)
	if err != nil || len(children) != 1 {
		t.Fatalf("before job finishes children = %+v, %v (pending job must be hidden)", children, err)
	}
	time.Sleep(3 * time.Millisecond)
	child2 := rawLecture(t, s, user, cB, "더 깊게 2", "")
	if err := s.ResolveLinkJob(ctx, job.ID, child2); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	parents, children, err := s.LectureLinks(ctx, user, parent)
	if err != nil {
		t.Fatalf("links: %v", err)
	}
	if len(parents) != 0 || len(children) != 2 || children[0].ID != child || children[1].ID != child2 ||
		children[0].Anchor != "principles-2" || children[1].LinkKind != LinkRegeneratedKind {
		t.Fatalf("parent view: parents=%+v children=%+v", parents, children)
	}
	parents, _, err = s.LectureLinks(ctx, user, child2)
	if err != nil || len(parents) != 1 || parents[0].ID != parent || parents[0].Title != "원 강의" {
		t.Fatalf("child view parents = %+v, %v", parents, err)
	}

	if err := s.UpdateProgress(ctx, user, child, "principles", 20); err != nil {
		t.Fatalf("progress: %v", err)
	}
	list, _, err := s.ListMyLectures(ctx, user, "in_progress", 20, "")
	if err != nil || len(list) != 1 || list[0].ParentLectureID != parent {
		t.Fatalf("my lectures = %+v, %v", list, err)
	}
	d, err := s.LoadToday(ctx, user, time.Now())
	if err != nil || len(d.Resume) != 1 || d.Resume[0].ParentLectureID != parent {
		t.Fatalf("today resume = %+v, %v", d.Resume, err)
	}
}

func TestPromptOverrideSaveResetRestore(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	const id = "teacher.system"
	if _, err := s.SavePromptOverride(ctx, user, id, "v1", "save"); err != nil {
		t.Fatal(err)
	}
	o, err := s.SavePromptOverride(ctx, user, id, "v2", "save")
	if err != nil || o.UpdatedAt.IsZero() {
		t.Fatalf("save = %+v %v", o, err)
	}
	got, err := s.PromptOverrides(ctx, user)
	if err != nil || got[id].Body != "v2" || len(got) != 1 {
		t.Fatalf("overrides = %+v, %v", got, err)
	}
	if err := s.ResetPromptOverride(ctx, user, id); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetPromptOverride(ctx, user, id); err != nil { // no override: no-op
		t.Fatal(err)
	}
	if got, _ := s.PromptOverrides(ctx, user); len(got) != 0 {
		t.Fatalf("reset must remove the override: %+v", got)
	}
	hist, err := s.PromptHistory(ctx, user, id)
	if err != nil || len(hist) != 3 || hist[0].Action != "reset" || hist[0].Body != "v2" || hist[2].Body != "v1" {
		t.Fatalf("history = %+v, %v", hist, err)
	}
	e, err := s.PromptHistoryEntryByID(ctx, user, hist[2].ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SavePromptOverride(ctx, user, id, e.Body, "restore"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.PromptOverrides(ctx, user); got[id].Body != "v1" {
		t.Fatalf("restored = %+v", got)
	}
	if _, err := s.PromptHistoryEntryByID(ctx, "someone-else", hist[2].ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user's history must be hidden: %v", err)
	}
	for i := 0; i < PromptHistoryLimit+5; i++ {
		if _, err := s.SavePromptOverride(ctx, user, id, "b"+strconv.Itoa(i), "save"); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	_ = s.c().row(ctx, `SELECT COUNT(*) FROM prompt_override_history WHERE user_id = $1`, user).Scan(&n)
	if n != PromptHistoryLimit {
		t.Fatalf("history rows = %d, want %d", n, PromptHistoryLimit)
	}

	if g, err := s.UserGloss(ctx, user, "nope", "h1"); err != nil || g != "" {
		t.Fatalf("missing user gloss = %q %v", g, err)
	}
	c, _ := twoConcepts(t, s)
	for _, g := range []string{"첫 설명", "고친 설명"} {
		if err := s.SaveUserGloss(ctx, user, c, "h1", g, "haiku"); err != nil {
			t.Fatal(err)
		}
	}
	if g, err := s.UserGloss(ctx, user, c, "h1"); err != nil || g != "고친 설명" {
		t.Fatalf("user gloss = %q %v", g, err)
	}
}

// 다 본 학습 요청: 요청한 주제 개념(resolved[0])의 강의를 끝까지 보면 완료다(witems cf8a322).
func TestStudyRequestLearnedByTopicLectureAndDismiss(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	topic, prereq := twoConcepts(t, s)
	lectureID := rawLecture(t, s, user, topic, "테스트 강의", "")

	req, _, err := s.CreateStudyRequest(ctx, models.StudyRequest{UserID: user, Source: "manual", Query: "테스트 요청", Context: []byte(`{}`),
		ResolvedConceptIDs: []string{}, Status: "unresolved", IdempotencyKey: "k:" + user})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.RecordResolution(ctx, req.ID, []string{topic, prereq}, prereq, ""); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	assertLearned(t, s, req.ID, false)
	if err := s.UpdateProgress(ctx, user, lectureID, "principles", 80); err != nil {
		t.Fatalf("progress: %v", err)
	}
	assertLearned(t, s, req.ID, false)
	if _, err := s.CompleteLecture(ctx, user, lectureID); err != nil {
		t.Fatalf("complete: %v", err)
	}
	assertLearned(t, s, req.ID, true)

	if err := s.SetStudyRequestDismissed(ctx, user, req.ID, true); err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	got, err := s.GetStudyRequest(ctx, user, req.ID)
	if err != nil || got.Status != "dismissed" {
		t.Fatalf("dismissed status = %q, %v", got.Status, err)
	}
	if err := s.SetStudyRequestDismissed(ctx, user, req.ID, false); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got, _ = s.GetStudyRequest(ctx, user, req.ID); got.Status != "resolved" {
		t.Fatalf("restored status = %q, want resolved", got.Status)
	}
	if err := s.SetStudyRequestDismissed(ctx, "someone-else", req.ID, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user's request = %v, want ErrNotFound", err)
	}
}

// 선행 개념만 배웠을 때는 완료가 아니다(주제 개념 기준).
func TestStudyRequestNotLearnedByPrerequisiteOnly(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	topic, prereq := twoConcepts(t, s)
	req, _, err := s.CreateStudyRequest(ctx, models.StudyRequest{UserID: testUser, Source: "manual", Query: "선행만", Context: []byte(`{}`),
		ResolvedConceptIDs: []string{topic, prereq}, Status: "resolved", IdempotencyKey: "k2"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.SetConceptState(ctx, testUser, prereq, "completed"); err != nil {
		t.Fatalf("prereq completed: %v", err)
	}
	assertLearned(t, s, req.ID, false)
	if _, err := s.SetConceptState(ctx, testUser, topic, "known"); err != nil {
		t.Fatalf("topic known: %v", err)
	}
	assertLearned(t, s, req.ID, true)
}

func assertLearned(t *testing.T, s *SQLiteStore, id string, want bool) {
	t.Helper()
	got, err := s.GetStudyRequest(context.Background(), testUser, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	list, err := s.ListStudyRequests(context.Background(), testUser, "", 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %v", list, err)
	}
	if got.Learned != want || list[0].Learned != want {
		t.Fatalf("learned get=%v list=%v, want %v", got.Learned, list[0].Learned, want)
	}
}

// v010Migrations is the migration set shipped in v0.1.0 (001–003).
func v010Migrations(t *testing.T) fs.FS {
	t.Helper()
	m := fstest.MapFS{}
	for _, name := range []string{"001_init.sql", "002_lecture_generation.sql", "003_learning_polish.sql"} {
		raw, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		m[name] = &fstest.MapFile{Data: raw}
	}
	return m
}

// TestUpgradeFromV010 builds a v0.1.0 database (migrations 001–003 + data),
// then applies the current migrations: existing rows survive, new tables
// work, and the 004 backfill links a "더 깊게" lecture to the lecture it was
// opened from.
func TestUpgradeFromV010(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, v010Migrations(t)); err != nil {
		t.Fatalf("v0.1.0 migrate: %v", err)
	}
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	// The seed upsert alone: today's SyncCurriculum also runs markObjectiveChanges
	// on the 011 tables, which a v0.1.0 database does not have yet.
	if err := old.tx(ctx, func(c conn) error { return applySeedPre017(ctx, c, seed) }); err != nil {
		t.Fatal(err)
	}
	cA, cB := twoConcepts(t, old)
	content := `{"sections":[{"anchor":"principles","blocks":[{"anchor":"principles-1","type":"term","term":"B","conceptId":"` + cB + `"}]}]}`
	parent := rawLecture(t, old, testUser, cA, "원 강의", content)
	unrelated := rawLecture(t, old, testUser, cA, "용어 없는 강의", "")
	for _, id := range []string{parent, unrelated} {
		if _, err := old.c().exec(ctx, `INSERT INTO user_lecture_progress (user_id, lecture_id, anchor, percent) VALUES ($1, $2, 'principles', 40)`, testUser, id); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(3 * time.Millisecond)
	child := rawLecture(t, old, testUser, cB, "더 깊게", "")
	if _, err := old.c().exec(ctx, `INSERT INTO user_learning_settings (user_id, example_language) VALUES ($1, 'go')`, testUser); err != nil {
		t.Fatal(err)
	}
	if _, err := old.c().exec(ctx, `INSERT INTO study_requests (id, user_id, source, query, resolved_concept_ids, status, idempotency_key)
		VALUES ($1, $2, 'manual', '옛 요청', $3, 'resolved', 'old')`, NewID(), testUser, []string{cB, cA}); err != nil {
		t.Fatal(err)
	}
	var conceptsBefore, objectivesBefore int
	_ = old.c().row(ctx, `SELECT COUNT(*) FROM concepts`).Scan(&conceptsBefore)
	_ = old.c().row(ctx, `SELECT COUNT(*) FROM objectives`).Scan(&objectivesBefore)
	if _, err := old.c().exec(ctx, `INSERT INTO user_concept_states (user_id, concept_id, state) VALUES ($1, $2, 'known')`, testUser, cA); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(ctx, migrations.Files); err != nil {
		t.Fatalf("upgrade migrate: %v", err)
	}
	if err := s.Migrate(ctx, migrations.Files); err != nil {
		t.Fatalf("second migrate (no-op): %v", err)
	}
	var applied int
	_ = s.c().row(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&applied)
	all, _ := fs.Glob(migrations.Files, "*.sql")
	if applied != len(all) {
		t.Fatalf("schema_migrations = %d, want %d", applied, len(all))
	}

	if lang := s.ExampleLanguageFor(ctx, testUser); lang != "go" {
		t.Fatalf("settings lost: %q", lang)
	}
	if lec, err := s.GetLecture(ctx, testUser, parent); err != nil || lec.Title != "원 강의" || lec.Progress == nil {
		t.Fatalf("lecture/progress lost: %+v %v", lec, err)
	}
	parents, _, err := s.LectureLinks(ctx, testUser, child)
	if err != nil || len(parents) != 1 || parents[0].ID != parent {
		t.Fatalf("backfilled parents = %+v, %v", parents, err)
	}
	if list, err := s.ListStudyRequests(ctx, testUser, "", 10); err != nil || len(list) != 1 || list[0].Learned {
		t.Fatalf("old study request = %+v %v", list, err)
	}
	if _, err := s.SavePromptOverride(ctx, testUser, "teacher.system", "x", "save"); err != nil {
		t.Fatalf("new table after upgrade: %v", err)
	}
	// 007: concepts was rebuilt (CHECK origin += chapter) with foreign keys off —
	// every row, child row and user state survives and foreign keys are back on.
	var conceptsAfter, objectivesAfter, fkOn int
	_ = s.c().row(ctx, `SELECT COUNT(*) FROM concepts`).Scan(&conceptsAfter)
	_ = s.c().row(ctx, `SELECT COUNT(*) FROM objectives`).Scan(&objectivesAfter)
	_ = s.c().row(ctx, `PRAGMA foreign_keys`).Scan(&fkOn)
	if conceptsAfter != conceptsBefore || objectivesAfter != objectivesBefore || fkOn != 1 {
		t.Fatalf("after 007: concepts %d→%d objectives %d→%d foreign_keys=%d",
			conceptsBefore, conceptsAfter, objectivesBefore, objectivesAfter, fkOn)
	}
	if d, err := s.GetConceptDetail(ctx, testUser, cA); err != nil || d.UserState != "known" || len(d.Objectives) == 0 {
		t.Fatalf("concept state/objectives lost by rebuild: %+v %v", d, err)
	}
	if fk, _ := collect(s.c().query(ctx, `PRAGMA foreign_key_check`), func(r rowScanner) (string, error) {
		var table, parent string
		var rowid, fkid any
		err := r.Scan(&table, &rowid, &parent, &fkid)
		return table, err
	}); len(fk) != 0 {
		t.Fatalf("foreign key violations after upgrade: %v", fk)
	}
	if _, err := s.c().exec(ctx, `INSERT INTO concepts (id, name, area, depth, origin, owner_user_id) VALUES ('t.chapter', 'c', 'os', 1, 'chapter', $1)`, testUser); err != nil {
		t.Fatalf("origin=chapter after 007: %v", err)
	}
	if _, err := s.c().exec(ctx, `INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES ('x#1', 'no-such-concept', 't', 1, 1)`); err == nil {
		t.Fatal("foreign keys must be enforced after the migration")
	}
	if st, err := s.GetLearningSettings(ctx, testUser); err != nil || st.DailyMinutes != 30 || st.ExampleLanguage != "go" {
		t.Fatalf("settings after 007: %+v %v", st, err)
	}

	// 006: reader_gaps on reviews (old reviews default to []).
	gaps := []byte(`[{"section":"principles","quote":"q","unknown":"u"}]`)
	if err := s.InsertReview(ctx, parent, models.LectureReview{Verdict: "fail", Reviewer: "llm", Attempt: 1, ReaderGaps: gaps}); err != nil {
		t.Fatalf("review with reader_gaps: %v", err)
	}
	if lec, err := s.GetLecture(ctx, testUser, parent); err != nil || len(lec.Reviews) != 1 || string(lec.Reviews[0].ReaderGaps) != string(gaps) {
		t.Fatalf("reader_gaps round trip: %+v %v", lec.Reviews, err)
	}

	raw, _ := migrations.Files.ReadFile("004_lecture_links.sql")
	stmt := string(raw)[strings.Index(string(raw), "INSERT INTO lecture_links"):]
	if _, err := s.c().exec(ctx, stmt); err != nil {
		t.Fatalf("backfill rerun: %v", err)
	}
	var n int
	_ = s.c().row(ctx, `SELECT COUNT(*) FROM lecture_links`).Scan(&n)
	if n != 1 {
		t.Fatalf("links after backfill rerun = %d, want 1 (idempotent)", n)
	}
}
