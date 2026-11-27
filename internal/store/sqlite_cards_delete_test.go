package store

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
)

func activityCards(days []ActivityDay) int {
	total := 0
	for _, d := range days {
		total += d.Cards
	}
	return total
}

// TestCardSoftDelete: 삭제는 표시만 한다 — 목록·복습 대상·개수에서 빠지고, 복습 기록은 남아 통계에 그대로 세며,
// 되돌리면 일정이 이어진다. 다른 사람의 카드·이미 지운 카드는 ErrNotFound, 지운 카드는 채점·수정이 안 된다.
func TestCardSoftDelete(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 25, 3, 0, 0, 0, time.UTC)
	tomorrow := now.Add(24 * time.Hour)
	note := NewID()
	if _, err := s.c().exec(ctx, `INSERT INTO notes (id, user_id, anchor, quote, body) VALUES ($1, $2, 'a', '인용', '본문')`, note, testUser); err != nil {
		t.Fatal(err)
	}
	keep, err := s.SaveCard(ctx, testUser, "note", note, "남길 카드", "답", now)
	if err != nil {
		t.Fatal(err)
	}
	ext, err := s.SaveExternalCard(ctx, testUser, ExternalCard{ExternalID: "atlas:c1", Question: "외부 질문", Answer: "외부 답"}, now)
	if err != nil {
		t.Fatal(err)
	}
	graded, err := s.GradeCard(ctx, testUser, ext.ID, learning.RatingGood, tomorrow)
	if err != nil {
		t.Fatal(err)
	}
	if due, done, _ := s.CardCounts(ctx, testUser, tomorrow); due != 1 || done != 1 {
		t.Fatalf("before delete: due=%d done=%d", due, done)
	}

	if err := s.DeleteCard(ctx, "other", ext.ID, tomorrow); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another user's card: %v", err)
	}
	if err := s.DeleteCard(ctx, testUser, ext.ID, tomorrow); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteCard(ctx, testUser, ext.ID, tomorrow); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleting twice: %v", err)
	}

	later := *graded.NextReviewAt // 지운 외부 카드가 복습 시각에 닿았다
	if list, _ := s.ListCards(ctx, testUser); len(list) != 1 || list[0].ID != keep.ID {
		t.Fatalf("list keeps only the live card: %+v", list)
	}
	if due, _ := s.DueCards(ctx, testUser, later, 10); len(due) != 1 || due[0].ID != keep.ID {
		t.Fatalf("due cards: %+v", due)
	}
	if due, done, _ := s.CardCounts(ctx, testUser, later); due != 1 || done != 0 {
		t.Fatalf("counts after delete: due=%d done=%d", due, done) // done 은 later 의 KST 날 채점 수 — 채점은 전날
	}
	if due, done, _ := s.CardCounts(ctx, testUser, tomorrow); due != 1 || done != 1 {
		t.Fatalf("counts on the graded day after delete: due=%d done=%d", due, done)
	}
	if days, _, err := s.ReviewForecast(ctx, testUser, now, 14); err != nil {
		t.Fatal(err)
	} else {
		total := 0
		for _, d := range days {
			total += d.Cards
		}
		if total != 1 {
			t.Fatalf("forecast counts the live card only: %d", total)
		}
	}
	// 복습 기록은 카드를 지워도 통계에 그대로 센다(지운 카드의 지난 복습도 한 일이다).
	if act, err := s.Activity(ctx, testUser, tomorrow, 7); err != nil {
		t.Fatal(err)
	} else if total := activityCards(act); total != 1 {
		t.Fatalf("activity keeps a deleted card's reviews: %d", total)
	}
	var logs int
	if err := s.c().row(ctx, `SELECT COUNT(*) FROM review_card_log WHERE card_id = $1`, ext.ID).Scan(&logs); err != nil || logs != 1 {
		t.Fatalf("review log stays: %d %v", logs, err)
	}
	if _, err := s.GradeCard(ctx, testUser, ext.ID, learning.RatingGood, later); !errors.Is(err, ErrNotFound) {
		t.Fatalf("grading a deleted card: %v", err)
	}
	q := "새 질문"
	if _, err := s.PatchExternalCard(ctx, testUser, "atlas:c1", CardPatch{Question: &q}, later); !errors.Is(err, ErrNotFound) {
		t.Fatalf("patching a deleted card: %v", err)
	}

	if _, err := s.RestoreCard(ctx, "other", ext.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another user's restore: %v", err)
	}
	if _, err := s.RestoreCard(ctx, testUser, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restoring nothing: %v", err)
	}
	back, err := s.RestoreCard(ctx, testUser, ext.ID)
	if err != nil || back.ID != ext.ID || back.Reps != 1 || back.NextReviewAt == nil || !back.NextReviewAt.Equal(later) {
		t.Fatalf("restored = %+v (%v)", back, err)
	}
	if _, err := s.RestoreCard(ctx, testUser, ext.ID); err != nil {
		t.Fatalf("a repeated restore is harmless: %v", err)
	}
	if list, _ := s.ListCards(ctx, testUser); len(list) != 2 {
		t.Fatalf("restored card is listed again: %+v", list)
	}
	if due, done, _ := s.CardCounts(ctx, testUser, later); due != 2 || done != 0 {
		// done 은 '오늘(later 의 KST 날)' 채점 수 — 채점은 어제(tomorrow)였다.
		t.Fatalf("counts after restore: due=%d done=%d", due, done)
	}
	if due, done, _ := s.CardCounts(ctx, testUser, tomorrow); due != 1 || done != 1 {
		t.Fatalf("counts on the graded day: due=%d done=%d", due, done)
	}
}

// TestCardSoftDeleteResave: 지운 카드의 출처를 다시 카드로 만들면(같은 노트·같은 외부 id) 새 행이 아니라
// 그 카드가 되살아난다 — 유일 키(user, source_kind, source_id)와 복습 기록이 이어진다.
func TestCardSoftDeleteResave(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 25, 3, 0, 0, 0, time.UTC)
	note := NewID()
	if _, err := s.c().exec(ctx, `INSERT INTO notes (id, user_id, anchor, quote, body) VALUES ($1, $2, 'a', '인용', '본문')`, note, testUser); err != nil {
		t.Fatal(err)
	}
	c, err := s.SaveCard(ctx, testUser, "note", note, "질문", "답", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteCard(ctx, testUser, c.ID, now); err != nil {
		t.Fatal(err)
	}
	if d, _ := s.CardDraftFor(ctx, testUser, "note", note); d.CardID != "" {
		t.Fatalf("a deleted card is not 'already made': %q", d.CardID)
	}
	again, err := s.SaveCard(ctx, testUser, "note", note, "고친 질문", "답", now)
	if err != nil || again.ID != c.ID || again.Question != "고친 질문" {
		t.Fatalf("re-save = %+v (%v)", again, err)
	}
	if list, _ := s.ListCards(ctx, testUser); len(list) != 1 {
		t.Fatalf("one live card: %+v", list)
	}

	e, err := s.SaveExternalCard(ctx, testUser, ExternalCard{ExternalID: "atlas:c9", Question: "q", Answer: "a"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteCard(ctx, testUser, e.ID, now); err != nil {
		t.Fatal(err)
	}
	e2, err := s.SaveExternalCard(ctx, testUser, ExternalCard{ExternalID: "atlas:c9", Question: "q2", Answer: "a"}, now)
	if err != nil || e2.ID != e.ID {
		t.Fatalf("external re-send revives the card: %+v (%v)", e2, err)
	}
	if list, _ := s.ListCards(ctx, testUser); len(list) != 2 {
		t.Fatalf("two live cards: %+v", list)
	}
}

// TestPurgeDeletedCards: 30일이 지난 삭제 카드는 행만 지우고 복습 기록은 남긴다. 29일은 그대로,
// 되돌린 카드·남의 카드·지우지 않은 카드는 건드리지 않는다. 기록이 가리키는 카드가 없어도 통계·개수 쿼리는
// 행을 잃지 않고, 같은 출처를 다시 카드로 만들 수 있으며, 외래 키 검사는 다시 켜져 있다.
func TestPurgeDeletedCards(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 12, 31, 3, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	mk := func(user, id string) ReviewCard {
		t.Helper()
		if user != testUser {
			if _, err := s.c().exec(ctx, `INSERT OR IGNORE INTO users (id) VALUES ($1)`, user); err != nil {
				t.Logf("users row: %v", err) // 사용자 표가 없으면 외래 키가 없다는 뜻 — 계속
			}
		}
		c, err := s.SaveExternalCard(ctx, user, ExternalCard{ExternalID: "atlas:" + id, Question: "q " + id, Answer: "a"}, now.Add(-90*day))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.GradeCard(ctx, user, c.ID, learning.RatingGood, now.Add(-60*day)); err != nil {
			t.Fatal(err)
		}
		return c
	}
	d29, d30, d31, restored, live, other := mk(testUser, "d29"), mk(testUser, "d30"), mk(testUser, "d31"), mk(testUser, "rest"), mk(testUser, "live"), mk("other", "o31")
	for c, ago := range map[string]time.Duration{d29.ID: 29, d30.ID: 30, d31.ID: 31, restored.ID: 40} {
		user := testUser
		if err := s.DeleteCard(ctx, user, c, now.Add(-ago*day)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.DeleteCard(ctx, "other", other.ID, now.Add(-31*day)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RestoreCard(ctx, testUser, restored.ID); err != nil { // 40일 전에 지웠지만 정리 전에 되돌림
		t.Fatal(err)
	}
	rowsOf := func() map[string]bool {
		t.Helper()
		ids, err := collect(s.c().query(ctx, `SELECT id FROM review_cards`), func(r rowScanner) (string, error) {
			var id string
			return id, r.Scan(&id)
		})
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]bool{}
		for _, id := range ids {
			out[id] = true
		}
		return out
	}
	logsBefore := 0
	_ = s.c().row(ctx, `SELECT COUNT(*) FROM review_card_log`).Scan(&logsBefore)
	statsBefore, _ := s.Activity(ctx, testUser, now, 120)
	_, doneBefore, _ := s.CardCounts(ctx, testUser, now.Add(-60*day))

	// 정리 전에 d29 가 30일째가 되는 날은 아직 아니다: now 에서는 d30·d31(과 남의 o31)만 지워진다.
	n, err := s.PurgeDeletedCards(ctx, now)
	if err != nil || n != 3 {
		t.Fatalf("purged %d (%v), want d30 d31 o31", n, err)
	}
	rows := rowsOf()
	for name, c := range map[string]ReviewCard{"29일": d29, "되돌린 카드": restored, "지우지 않은 카드": live} {
		if !rows[c.ID] {
			t.Errorf("%s was removed", name)
		}
	}
	for name, c := range map[string]ReviewCard{"30일": d30, "31일": d31, "남의 31일": other} {
		if rows[c.ID] {
			t.Errorf("%s card still there", name)
		}
	}
	if n, err := s.PurgeDeletedCards(ctx, now); err != nil || n != 0 {
		t.Fatalf("second purge removed %d (%v)", n, err)
	}
	// 하루 뒤 29일 카드가 30일을 채운다.
	if n, err := s.PurgeDeletedCards(ctx, now.Add(day)); err != nil || n != 1 || rowsOf()[d29.ID] {
		t.Fatalf("d29 after one more day: removed %d (%v)", n, err)
	}

	// 기록은 하나도 줄지 않았다 — 통계·개수는 그대로고, 고아 기록이 쿼리를 깨지 않는다.
	var logsAfter int
	if err := s.c().row(ctx, `SELECT COUNT(*) FROM review_card_log`).Scan(&logsAfter); err != nil || logsAfter != logsBefore {
		t.Fatalf("review log: %d -> %d (%v)", logsBefore, logsAfter, err)
	}
	statsAfter, err := s.Activity(ctx, testUser, now, 120)
	if err != nil || activityCards(statsAfter) != activityCards(statsBefore) || activityCards(statsAfter) != 5 {
		t.Fatalf("activity cards %d -> %d (%v)", activityCards(statsBefore), activityCards(statsAfter), err)
	}
	if _, done, err := s.CardCounts(ctx, testUser, now.Add(-60*day)); err != nil || done != doneBefore {
		t.Fatalf("done on the graded day %d -> %d (%v)", doneBefore, done, err)
	}
	if list, err := s.ListCards(ctx, testUser); err != nil || len(list) != 2 {
		t.Fatalf("list after purge: %+v (%v)", list, err)
	}

	// 지운 뒤 같은 외부 id 로 다시 보내면 새 카드(새 id)로 만들어지고 고아 기록과 부딪히지 않는다.
	again, err := s.SaveExternalCard(ctx, testUser, ExternalCard{ExternalID: "atlas:d31", Question: "다시", Answer: "a"}, now)
	if err != nil || again.ID == d31.ID {
		t.Fatalf("re-send after purge = %+v (%v)", again, err)
	}
	// 정리가 끝나면 외래 키 검사는 다시 켜져 있다(없는 카드에 기록을 달 수 없다).
	if _, err := s.c().exec(ctx, `INSERT INTO review_card_log (card_id, user_id, rating) VALUES ('nope', $1, 3)`, testUser); err == nil {
		t.Fatal("foreign keys are off after the purge")
	}
}

// TestMigrateToleratesPurgedCardLog: 정리로 생긴 고아 복습 기록이 있어도, 외래 키를 끄고 표를 다시 만드는
// 이후 마이그레이션(-- punisher:foreign_keys=off)이 외래 키 검사에서 실패하지 않는다. 다른 표의 위반은 여전히 실패.
func TestMigrateToleratesPurgedCardLog(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 12, 31, 3, 0, 0, 0, time.UTC)
	c, err := s.SaveExternalCard(ctx, testUser, ExternalCard{ExternalID: "atlas:m1", Question: "q", Answer: "a"}, now.Add(-90*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.GradeCard(ctx, testUser, c.ID, learning.RatingGood, now.Add(-60*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteCard(ctx, testUser, c.ID, now.Add(-40*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if n, err := s.PurgeDeletedCards(ctx, now); err != nil || n != 1 {
		t.Fatalf("purge = %d (%v)", n, err)
	}
	next := fstest.MapFS{"998_after_purge.sql": {Data: []byte("-- punisher:foreign_keys=off\nCREATE TABLE IF NOT EXISTS t998 (id TEXT);\n")}}
	if err := s.Migrate(ctx, next); err != nil {
		t.Fatalf("a fk-off migration after a purge: %v", err)
	}
}
