package store

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/migrations"
)

// SQLite port of witems postgres_i18n_test.go and postgres_locale_test.go.

func TestCheckItemJobKeyLanguage(t *testing.T) {
	k := CheckItemKey{LectureID: "L", Purpose: "check", PromptHash: "h", CreatedFor: "u"}
	if got := k.JobKey(); got != "check_items:L:check:h:u" {
		t.Fatalf("ko key must stay the pre-i18n key: %s", got)
	}
	k.Language = learning.LocaleKO
	if got := k.JobKey(); got != "check_items:L:check:h:u" {
		t.Fatalf("explicit ko key: %s", got)
	}
	k.Language = learning.LocaleEN
	if got := k.JobKey(); got != "check_items:L:check:h:u|en" {
		t.Fatalf("en key: %s", got)
	}
}

func setLocale(t *testing.T, s *SQLiteStore, user, locale string) {
	t.Helper()
	if _, err := s.UpdateLearningSettings(context.Background(), user, LearningSettingsPatch{Locale: &locale}); err != nil {
		t.Fatal(err)
	}
}

func testLecture(conceptID, objectiveID, lang, user string) NewLecture {
	return NewLecture{ConceptID: conceptID, ObjectiveIDs: []string{objectiveID}, ObjectiveKey: objectiveID, Depth: 1,
		Kind: learning.JobInstant, Status: "published", Title: "강의 " + lang, Content: []byte(`{}`), Sources: []byte(`[]`),
		RequestedBy: user, Language: lang}
}

// ko and en saves of one target at the same time: the single connection
// serializes MAX(version)+1 (witems takes an advisory lock), so both succeed
// with distinct versions.
func TestInsertLectureVersionAcrossLanguages(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	conceptID, objectiveID := masteryFixture(t, s, testUser)

	const rounds = 4
	var wg sync.WaitGroup
	errs := make(chan error, 2*rounds)
	for i := 0; i < rounds; i++ {
		for _, lang := range []string{learning.LocaleKO, learning.LocaleEN} {
			wg.Add(1)
			go func(lang string) {
				defer wg.Done()
				if _, err := s.InsertLecture(ctx, testLecture(conceptID, objectiveID, lang, testUser)); err != nil {
					errs <- err
				}
			}(lang)
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent insert: %v", err)
	}
	var n, distinct, maxV, ko, en int
	if err := s.c().row(ctx, `SELECT COUNT(*), COUNT(DISTINCT version), MAX(version),
		COUNT(CASE WHEN language = 'ko' THEN 1 END), COUNT(CASE WHEN language = 'en' THEN 1 END)
		FROM lectures WHERE concept_id = $1`, conceptID).Scan(&n, &distinct, &maxV, &ko, &en); err != nil {
		t.Fatal(err)
	}
	if n != 2*rounds || distinct != n || maxV != n || ko != rounds || en != rounds {
		t.Fatalf("versions must run across languages: n=%d distinct=%d max=%d ko=%d en=%d", n, distinct, maxV, ko, en)
	}
}

// Generation dedup / instant reuse take the language only; the check card
// (LangPrefer) takes the user's language first, else another.
func TestLectureLookupLanguageModes(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	conceptID, objectiveID := masteryFixture(t, s, user)
	ko, err := s.InsertLecture(ctx, testLecture(conceptID, objectiveID, learning.LocaleKO, user))
	if err != nil {
		t.Fatal(err)
	}
	if ko.Language != learning.LocaleKO {
		t.Fatalf("language not scanned: %q", ko.Language)
	}
	ids := []string{objectiveID}

	if _, err := s.LatestLectureForTarget(ctx, learning.JobInstant, conceptID, objectiveID, 1, learning.LocaleEN, []string{"published"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("en dedup must not see the ko lecture: %v", err)
	}
	if got, err := s.LatestLectureForTarget(ctx, learning.JobInstant, conceptID, objectiveID, 1, learning.LocaleKO, []string{"published"}); err != nil || got.ID != ko.ID {
		t.Fatalf("ko dedup: %v %v", got.ID, err)
	}
	if _, err := s.PublishedLectureCovering(ctx, user, conceptID, 1, ids, learning.LocaleEN, LangExact); !errors.Is(err, ErrNotFound) {
		t.Fatalf("en instant reuse must not reuse the ko lecture: %v", err)
	}
	if got, err := s.PublishedLectureCovering(ctx, user, conceptID, 1, ids, learning.LocaleEN, LangPrefer); err != nil || got.ID != ko.ID {
		t.Fatalf("prefer falls back to another language: %v %v", got.ID, err)
	}

	en, err := s.InsertLecture(ctx, testLecture(conceptID, objectiveID, learning.LocaleEN, user))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.PublishedLectureCovering(ctx, user, conceptID, 1, ids, learning.LocaleEN, LangPrefer); err != nil || got.ID != en.ID {
		t.Fatalf("en prefer: %v %v", got.ID, err)
	}
	if got, err := s.PublishedLectureCovering(ctx, user, conceptID, 1, ids, learning.LocaleKO, LangPrefer); err != nil || got.ID != ko.ID {
		t.Fatalf("ko prefer must keep the ko lecture: %v %v", got.ID, err)
	}
	if got, err := s.PublishedLectureCovering(ctx, user, conceptID, 1, ids, learning.LocaleEN, LangExact); err != nil || got.ID != en.ID {
		t.Fatalf("en exact: %v %v", got.ID, err)
	}

	// 이력(대상마다 최신 1개): the user's language.
	setLocale(t, s, user, learning.LocaleKO)
	list, err := s.ListConceptLectures(ctx, user, conceptID)
	if err != nil || len(list) != 1 || list[0].ID != ko.ID {
		t.Fatalf("ko history: %+v %v", list, err)
	}
	setLocale(t, s, user, learning.LocaleEN)
	if list, err = s.ListConceptLectures(ctx, user, conceptID); err != nil || len(list) != 1 || list[0].ID != en.ID {
		t.Fatalf("en history: %+v %v", list, err)
	}

	// 이어보기: reading both → the user's language one; only the other → it stays.
	for _, id := range []string{ko.ID, en.ID} {
		if err := s.UpdateProgress(ctx, user, id, "", 10); err != nil {
			t.Fatal(err)
		}
	}
	assertTodayResume(t, s, user, en.ID, true)
	assertTodayResume(t, s, user, ko.ID, false)
	setLocale(t, s, user, learning.LocaleKO)
	assertTodayResume(t, s, user, ko.ID, true)
	assertTodayResume(t, s, user, en.ID, false)
	if _, err := s.c().exec(ctx, `DELETE FROM user_lecture_progress WHERE user_id = $1 AND lecture_id = $2`, user, ko.ID); err != nil {
		t.Fatal(err)
	}
	assertTodayResume(t, s, user, en.ID, true)
}

func assertTodayResume(t *testing.T, s *SQLiteStore, user, lectureID string, want bool) {
	t.Helper()
	d, err := s.LoadToday(context.Background(), user, time.Now())
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	in := false
	for _, r := range d.Resume {
		in = in || r.LectureID == lectureID
	}
	if in != want {
		t.Fatalf("resume %s = %v, want %v", lectureID, in, want)
	}
}

// The active-job index includes the language (017 replaces the partial unique
// index): ko and en jobs of one target run together, a second job in the same
// language is blocked.
func TestActiveJobPerLanguage(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	conceptID, objectiveID := masteryFixture(t, s, user)
	newJob := func(lang string) NewJob {
		return NewJob{IdempotencyKey: "instant:" + user + ":" + RandomToken(), Kind: learning.JobInstant, UserID: user,
			ConceptID: conceptID, ObjectiveKey: objectiveID, Depth: 1, Reason: "test", Language: lang}
	}
	ko, created, err := s.CreateJob(ctx, newJob(learning.LocaleKO))
	if err != nil || !created || ko.Language != learning.LocaleKO {
		t.Fatalf("ko job: %+v %v %v", ko, created, err)
	}
	en, created, err := s.CreateJob(ctx, newJob(learning.LocaleEN))
	if err != nil || !created || en.Language != learning.LocaleEN {
		t.Fatalf("en job must run beside the ko job: %v %v", created, err)
	}
	dup, _, err := s.CreateJob(ctx, newJob(learning.LocaleEN))
	if !errors.Is(err, ErrJobActive) || dup.ID != en.ID {
		t.Fatalf("second en job must be blocked by the en job: %v %v", dup.ID, err)
	}
	if got, err := s.ActiveJobForTarget(ctx, objectiveID, 1, learning.LocaleKO); err != nil || got.ID != ko.ID {
		t.Fatalf("active ko: %v %v", got.ID, err)
	}
	if got, err := s.ActiveJobForTarget(ctx, objectiveID, 1, learning.LocaleEN); err != nil || got.ID != en.ID {
		t.Fatalf("active en: %v %v", got.ID, err)
	}
}

// An English item is also retired when only the objective's English text
// changed; Korean items follow the Korean text only.
func TestEnglishItemRetiresOnTextEnChange(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	conceptID, objectiveID := masteryFixture(t, s, testUser)
	if _, err := s.c().exec(ctx, `UPDATE objectives SET text_en = 'read path' WHERE id = $1`, objectiveID); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, lang := range []string{"ko", "en"} {
		ids[lang] = NewID()
		if _, err := s.c().exec(ctx, `INSERT INTO check_items (id, concept_id, objective_id, purpose, format, prompt,
			objective_text, objective_depth, language, objective_text_en)
			VALUES ($1, $2, $3, 'check', 'recall', '왜?', '읽기 경로', 1, $4, 'read path')`,
			ids[lang], conceptID, objectiveID, lang); err != nil {
			t.Fatal(err)
		}
	}
	run := func() {
		if err := s.tx(ctx, func(c conn) error { return markObjectiveChanges(ctx, c) }); err != nil {
			t.Fatal(err)
		}
	}
	status := func(id string) string {
		var st string
		if err := s.c().row(ctx, `SELECT status FROM check_items WHERE id = $1`, id).Scan(&st); err != nil {
			t.Fatal(err)
		}
		return st
	}
	run()
	if status(ids["ko"]) != "active" || status(ids["en"]) != "active" {
		t.Fatal("unchanged texts must keep both items")
	}
	if _, err := s.c().exec(ctx, `UPDATE objectives SET text_en = 'the read path' WHERE id = $1`, objectiveID); err != nil {
		t.Fatal(err)
	}
	run()
	if status(ids["ko"]) != "active" || status(ids["en"]) != "retired" {
		t.Fatalf("text_en change: ko=%s en=%s", status(ids["ko"]), status(ids["en"]))
	}
}

// Review items: the user's language first (even a seen one), another
// language only when the objective has none.
func TestReviewItemsPreferUserLanguage(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	conceptID, objectiveID := masteryFixture(t, s, user)
	insert := func(lang string) string {
		id := NewID()
		if _, err := s.c().exec(ctx, `INSERT INTO check_items (id, concept_id, objective_id, purpose, format, prompt,
			objective_text, objective_depth, language) VALUES ($1, $2, $3, 'review', 'recall', $4, '읽기 경로', 1, $5)`,
			id, conceptID, objectiveID, "문항 "+lang, lang); err != nil {
			t.Fatal(err)
		}
		return id
	}
	pick := func() string {
		got, err := s.ReviewItems(ctx, user, []string{objectiveID})
		if err != nil {
			t.Fatal(err)
		}
		return got[objectiveID].Item.ID
	}
	koID := insert(learning.LocaleKO)
	setLocale(t, s, user, learning.LocaleEN)
	if pick() != koID {
		t.Fatal("no en item: the ko item must be given")
	}
	enID := insert(learning.LocaleEN)
	if pick() != enID {
		t.Fatal("en user must get the en item")
	}
	setLocale(t, s, user, learning.LocaleKO)
	if pick() != koID {
		t.Fatal("ko user must get the ko item")
	}
}

// '안 본 문항 있음' counts only items in the user's language: an unseen ko
// item satisfies a ko user, an en user still gets a variant target.
func TestReviewVariantTargetsUserLanguage(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	conceptID, objectiveID := masteryFixture(t, s, user)
	lec, err := s.InsertLecture(ctx, testLecture(conceptID, objectiveID, learning.LocaleKO, user))
	if err != nil {
		t.Fatal(err)
	}
	due := time.Now().Add(-time.Hour)
	saveTestMastery(t, s, user, objectiveID, learning.Mastery{Box: 2, Best: learning.BestUnaided, NextReviewAt: &due})
	if _, err := s.c().exec(ctx, `INSERT INTO check_items (id, concept_id, objective_id, lecture_id, purpose, format, prompt,
		objective_text, objective_depth, language) VALUES ($1, $2, $3, $4, 'check', 'recall', '왜?', '읽기 경로', 1, 'ko')`,
		NewID(), conceptID, objectiveID, lec.ID); err != nil {
		t.Fatal(err)
	}
	targets := func() int {
		list, err := s.ReviewVariantTargets(ctx, user, time.Now().Add(24*time.Hour), 10)
		if err != nil {
			t.Fatal(err)
		}
		return len(list)
	}
	if n := targets(); n != 0 {
		t.Fatalf("ko user has an unseen ko item: %d targets", n)
	}
	setLocale(t, s, user, learning.LocaleEN)
	if n := targets(); n != 1 {
		t.Fatalf("en user has no unseen en item: %d targets", n)
	}
}

// localizedFixture is masteryFixture with an English overlay (name, summary,
// cases, objective) like the curriculum seed writes.
func localizedFixture(t *testing.T, s *SQLiteStore, user string) (conceptID, objectiveID string) {
	t.Helper()
	conceptID, objectiveID = masteryFixture(t, s, user)
	ctx := context.Background()
	if _, err := s.c().exec(ctx, `UPDATE concepts SET summary = '요약', practical_cases = '["사례"]',
		name_en = 'Cache', summary_en = 'Summary', practical_cases_en = '["Case"]' WHERE id = $1`, conceptID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.c().exec(ctx, `UPDATE objectives SET text_en = 'Read path' WHERE id = $1`, objectiveID); err != nil {
		t.Fatal(err)
	}
	return conceptID, objectiveID
}

// Screen reads follow the user's locale; English falls back to Korean when empty.
func TestScreenNamesFollowLocale(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	conceptID, _ := localizedFixture(t, s, user)

	check := func(locale, name, summary, objective, practical string) {
		t.Helper()
		d, err := s.GetConceptDetail(ctx, user, conceptID)
		if err != nil {
			t.Fatal(err)
		}
		if d.Name != name || d.Summary != summary || len(d.Objectives) != 1 || d.Objectives[0].Text != objective ||
			!strings.Contains(string(d.PracticalCases), practical) {
			t.Fatalf("%s detail: %q %q %+v %s", locale, d.Name, d.Summary, d.Objectives, d.PracticalCases)
		}
		today, err := s.LoadToday(ctx, user, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if c := today.Concepts[conceptID]; c.Name != name || c.Summary != summary || c.PracticalCase != practical {
			t.Fatalf("%s today: %+v", locale, c)
		}
		mastery, err := s.ConceptMastery(ctx, user, conceptID)
		if err != nil || len(mastery) != 1 || mastery[0].Text != objective {
			t.Fatalf("%s mastery: %+v %v", locale, mastery, err)
		}
	}
	check("ko", "캐시", "요약", "읽기 경로", "사례")
	setLocale(t, s, user, learning.LocaleEN)
	check("en", "Cache", "Summary", "Read path", "Case")

	// 영어가 비면 한국어로 대체.
	if _, err := s.c().exec(ctx, `UPDATE concepts SET name_en = '', practical_cases_en = '[]' WHERE id = $1`, conceptID); err != nil {
		t.Fatal(err)
	}
	check("en fallback", "캐시", "Summary", "Read path", "사례")
}

// Generation reads take the explicit language (the lecture's or the job's), not the user's.
func TestConceptInfoLanguage(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	conceptID, _ := localizedFixture(t, s, user)
	setLocale(t, s, user, learning.LocaleEN)

	ko, err := s.ConceptInfo(ctx, user, conceptID, learning.LocaleKO)
	if err != nil || ko.Name != "캐시" || ko.Objectives[0].Text != "읽기 경로" || ko.PracticalCases[0] != "사례" {
		t.Fatalf("ko info: %+v %v", ko, err)
	}
	en, err := s.ConceptInfo(ctx, user, conceptID, learning.LocaleEN)
	if err != nil || en.Name != "Cache" || en.Summary != "Summary" || en.Objectives[0].Text != "Read path" || en.PracticalCases[0] != "Case" {
		t.Fatalf("en info: %+v %v", en, err)
	}
	catalog, err := s.ConceptCatalog(ctx, user, learning.LocaleEN)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range catalog {
		if c.ID == conceptID {
			found = c.Name == "Cache"
		}
	}
	if !found {
		t.Fatal("en catalog name")
	}
	// 정본(변경 감지·스냅숏)은 한국어 그대로.
	var text string
	if err := s.c().row(ctx, `SELECT text FROM objectives WHERE concept_id = $1`, conceptID).Scan(&text); err != nil || text != "읽기 경로" {
		t.Fatalf("canonical objective text: %q %v", text, err)
	}
}

// Standalone only (design L2): a user who never saved settings gets the store
// default (the OS locale, config → SetDefaultLocale) in Go and in SQL reads;
// a saved locale wins.
func TestDefaultLocaleForUnsavedUser(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	conceptID, _ := localizedFixture(t, s, testUser)
	SetDefaultLocale(learning.LocaleEN)
	t.Cleanup(func() { SetDefaultLocale(learning.LocaleKO) })

	if st, err := s.GetLearningSettings(ctx, testUser); err != nil || st.Locale != learning.LocaleEN {
		t.Fatalf("unsaved settings locale = %q (%v)", st.Locale, err)
	}
	if got := s.LocaleFor(ctx, testUser); got != learning.LocaleEN {
		t.Fatalf("LocaleFor = %q", got)
	}
	if d, err := s.GetConceptDetail(ctx, testUser, conceptID); err != nil || d.Name != "Cache" {
		t.Fatalf("SQL default locale: %q %v", d.Name, err)
	}
	// First save of another field keeps the default locale; a saved ko wins afterwards.
	minutes := 30
	if st, err := s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{DailyMinutes: &minutes}); err != nil || st.Locale != learning.LocaleEN {
		t.Fatalf("first save locale = %q (%v)", st.Locale, err)
	}
	setLocale(t, s, testUser, learning.LocaleKO)
	if d, err := s.GetConceptDetail(ctx, testUser, conceptID); err != nil || d.Name != "캐시" {
		t.Fatalf("saved ko: %q %v", d.Name, err)
	}
	SetDefaultLocale(learning.LocaleKO)
	other := "someone-else"
	if got := s.LocaleFor(ctx, other); got != learning.LocaleKO {
		t.Fatalf("ko default = %q", got)
	}
}

// TestUpgradeFromV016 builds a v0.8 database (001–016, seed written the v0.8
// way, saved settings, a lecture, a check item and an active job), then applies
// 017: every row keeps its data and reads as Korean, the active-job index now
// includes the language, and the next SyncCurriculum fills the English overlay.
func TestUpgradeFromV016(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "017")); err != nil {
		t.Fatalf("016 migrate: %v", err)
	}
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.tx(ctx, func(c conn) error { return applySeedPre017(ctx, c, seed) }); err != nil {
		t.Fatal(err)
	}
	// v0.8's seed version had no English overlay in its hash.
	if _, err := old.c().exec(ctx, `UPDATE curriculum_versions SET version = 'v0.8-seed'`); err != nil {
		t.Fatal(err)
	}
	if _, err := old.c().exec(ctx, `INSERT INTO user_learning_settings (user_id, example_language, daily_minutes) VALUES ($1, 'go', 45)`, testUser); err != nil {
		t.Fatal(err)
	}
	c := seed.Concepts[0]
	objectiveID := curriculum.ObjectiveID(c.ID, 1)
	lectureID := NewID()
	if _, err := old.c().exec(ctx, `INSERT INTO lectures (id, concept_id, objective_ids, objective_key, depth, kind, status, version, title, content, published_at)
		VALUES ($1, $2, $3, $4, 1, 'library', 'published', 1, '옛 강의', '{}', NOW())`, lectureID, c.ID, []string{objectiveID}, objectiveID); err != nil {
		t.Fatal(err)
	}
	itemID := NewID()
	if _, err := old.c().exec(ctx, `INSERT INTO check_items (id, concept_id, objective_id, lecture_id, purpose, format, prompt, objective_text, objective_depth)
		VALUES ($1, $2, $3, $4, 'check', 'recall', '왜?', $5, $6)`, itemID, c.ID, objectiveID, lectureID, c.Objectives[0].Text, c.Objectives[0].Depth); err != nil {
		t.Fatal(err)
	}
	if _, err := old.c().exec(ctx, `INSERT INTO generation_jobs (id, idempotency_key, kind, user_id, concept_id, objective_key, depth, status, reason)
		VALUES ($1, 'old-job', 'instant', $2, $3, $4, 1, 'running', 'old')`, NewID(), testUser, c.ID, objectiveID); err != nil {
		t.Fatal(err)
	}
	var conceptsBefore int
	_ = old.c().row(ctx, `SELECT COUNT(*) FROM concepts`).Scan(&conceptsBefore)
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
	var conceptsAfter int
	_ = s.c().row(ctx, `SELECT COUNT(*) FROM concepts`).Scan(&conceptsAfter)
	if conceptsAfter != conceptsBefore {
		t.Fatalf("concepts %d → %d (no table rebuild, no row lost)", conceptsBefore, conceptsAfter)
	}
	// A saved settings row predates the language setting: it reads as ko (the
	// column default) whatever the OS default is.
	SetDefaultLocale(learning.LocaleEN)
	t.Cleanup(func() { SetDefaultLocale(learning.LocaleKO) })
	st, err := s.GetLearningSettings(ctx, testUser)
	if err != nil || st.Locale != learning.LocaleKO || st.ExampleLanguage != "go" || st.DailyMinutes != 45 {
		t.Fatalf("settings after 017 = %+v (%v)", st, err)
	}
	lec, err := s.GetLecture(ctx, testUser, lectureID)
	if err != nil || lec.Language != learning.LocaleKO || lec.Title != "옛 강의" {
		t.Fatalf("old lecture after 017 = %q %q (%v)", lec.Language, lec.Title, err)
	}
	items, err := s.ActiveCheckItems(ctx, testUser, CheckItemKey{LectureID: lectureID, Purpose: "check"})
	if err != nil || len(items) != 1 || items[0].ID != itemID || items[0].Language != learning.LocaleKO {
		t.Fatalf("old item after 017 = %+v (%v)", items, err)
	}
	if !indexExists(t, s, "generation_jobs_one_active_target") || !indexExists(t, s, "lectures_target_language_idx") {
		t.Fatal("017 indexes missing")
	}
	// The old running job is ko: an en job of the same target can start, a ko one cannot.
	if _, created, err := s.CreateJob(ctx, NewJob{IdempotencyKey: "new-en", Kind: learning.JobInstant, UserID: testUser,
		ConceptID: c.ID, ObjectiveKey: objectiveID, Depth: 1, Reason: "t", Language: learning.LocaleEN}); err != nil || !created {
		t.Fatalf("en job beside the old ko job: %v %v", created, err)
	}
	if _, _, err := s.CreateJob(ctx, NewJob{IdempotencyKey: "new-ko", Kind: learning.JobInstant, UserID: testUser,
		ConceptID: c.ID, ObjectiveKey: objectiveID, Depth: 1, Reason: "t", Language: learning.LocaleKO}); !errors.Is(err, ErrJobActive) {
		t.Fatalf("ko job must wait for the old ko job: %v", err)
	}

	// The seed version includes the English overlay, so the upgraded app re-syncs and fills it.
	applied, err := s.SyncCurriculum(ctx, seed)
	if err != nil || !applied {
		t.Fatalf("sync after 017: %v %v", applied, err)
	}
	var nameEn string
	if err := s.c().row(ctx, `SELECT name_en FROM concepts WHERE id = $1`, c.ID).Scan(&nameEn); err != nil || nameEn == "" || nameEn != c.NameEn {
		t.Fatalf("English overlay after sync: %q (want %q) %v", nameEn, c.NameEn, err)
	}
	if items, err := s.ActiveCheckItems(ctx, testUser, CheckItemKey{LectureID: lectureID, Purpose: "check"}); err != nil || len(items) != 1 {
		t.Fatalf("filling text_en keeps the ko item: %+v (%v)", items, err)
	}
}
