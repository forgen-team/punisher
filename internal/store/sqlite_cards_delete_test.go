package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
)

// TestCardSoftDelete: 삭제는 표시만 한다 — 목록·복습 대상·개수에서 빠지고, 복습 기록은 남으며,
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
		t.Fatalf("counts after delete: due=%d done=%d", due, done)
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
	if act, err := s.Activity(ctx, testUser, tomorrow, 7); err != nil {
		t.Fatal(err)
	} else {
		for _, d := range act {
			if d.Cards != 0 {
				t.Fatalf("activity skips a deleted card's reviews: %+v", d)
			}
		}
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
