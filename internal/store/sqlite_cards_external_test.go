package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/forgen-team/punisher/migrations"
)

// TestUpgradeFromV031ExternalCards: 032 rebuilds review_cards keeping the
// cards, their schedule, the review log (its foreign key) and the due index;
// then external cards can be stored (and 033 soft delete works on the old card).
func TestUpgradeFromV031ExternalCards(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 25, 3, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "032")); err != nil {
		t.Fatalf("031 migrate: %v", err)
	}
	note := NewID()
	if _, err := old.c().exec(ctx, `INSERT INTO notes (id, user_id, anchor, quote, body) VALUES ($1, $2, 'a', '인용', '본문')`, note, testUser); err != nil {
		t.Fatal(err)
	}
	// 031 의 표에는 새 열이 없으니 지금 코드(SaveCard·GradeCard) 대신 SQL 로 넣는다.
	cardID := NewID()
	if _, err := old.c().exec(ctx, `INSERT INTO review_cards (id, user_id, source_kind, source_id, question, answer, reps, next_review_at)
		VALUES ($1, $2, 'note', $3, '옛 질문', '옛 답', 1, $4)`, cardID, testUser, note, now); err != nil {
		t.Fatal(err)
	}
	if _, err := old.c().exec(ctx, `INSERT INTO review_card_log (card_id, user_id, rating) VALUES ($1, $2, 3)`, cardID, testUser); err != nil {
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
	for _, idx := range []string{"review_cards_due", "review_cards_external"} {
		if !indexExists(t, s, idx) {
			t.Errorf("index %s missing after 032", idx)
		}
	}
	list, err := s.ListCards(ctx, testUser)
	if err != nil || len(list) != 1 || list[0].Question != "옛 질문" || list[0].Reps != 1 || list[0].Stale || list[0].ExternalID != "" {
		t.Fatalf("old card after 032 = %+v (%v)", list, err)
	}
	var logs int
	if err := s.c().row(ctx, `SELECT COUNT(*) FROM review_card_log WHERE card_id = $1`, cardID).Scan(&logs); err != nil || logs != 1 {
		t.Fatalf("review log kept: %d %v", logs, err)
	}
	// 지우면 표시만 하고(033) 로그는 남는다 — 되돌리면 일정이 이어진다.
	if err := s.DeleteCard(ctx, testUser, cardID, now); err != nil {
		t.Fatal(err)
	}
	if err := s.c().row(ctx, `SELECT COUNT(*) FROM review_card_log WHERE card_id = $1`, cardID).Scan(&logs); err != nil || logs != 1 {
		t.Fatalf("log stays with a deleted card: %d %v", logs, err)
	}
	if _, err := s.SaveExternalCard(ctx, testUser, ExternalCard{ExternalID: "atlas:c1", Question: "q", Answer: "a"}, now); err != nil {
		t.Fatalf("external card after upgrade: %v", err)
	}
}

func TestExternalCards(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 25, 3, 0, 0, 0, time.UTC)
	in := ExternalCard{ExternalID: "atlas:01HCARD", SourceURL: "http://127.0.0.1:4720/#/cards?card=01HCARD",
		Question: "수납: 본인부담 상한을 넘으면?", Answer: "상한을 넘은 금액은 환급한다.\n근거: payment/copay.ts:10-14"}
	c, err := s.SaveExternalCard(ctx, testUser, in, now)
	if err != nil || c.SourceKind != CardSourceExternal || c.SourceID != in.ExternalID || c.ExternalID != in.ExternalID ||
		c.SourceURL != in.SourceURL || c.Stale || c.NextReviewAt == nil || !c.NextReviewAt.After(now) {
		t.Fatalf("external card = %+v (%v)", c, err)
	}
	if _, err := s.GradeCard(ctx, testUser, c.ID, 3, now.Add(48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	// 같은 externalId 로 다시 보내면 같은 카드를 고친다(일정은 그대로).
	in.Answer, in.Stale = "새 답", true
	again, err := s.SaveExternalCard(ctx, testUser, in, now.Add(72*time.Hour))
	if err != nil || again.ID != c.ID || again.Answer != "새 답" || !again.Stale || again.Reps != 1 {
		t.Fatalf("resend = %+v (%v)", again, err)
	}
	if list, _ := s.ListCards(ctx, testUser); len(list) != 1 {
		t.Fatalf("resend made %d cards", len(list))
	}
	// 다른 사용자는 같은 externalId 로 따로 갖는다.
	if other, err := s.SaveExternalCard(ctx, "someone", in, now); err != nil || other.ID == c.ID {
		t.Fatalf("other user = %+v (%v)", other, err)
	}

	off, text := false, "고친 질문"
	p, err := s.PatchExternalCard(ctx, testUser, in.ExternalID, CardPatch{Stale: &off, Question: &text}, now)
	if err != nil || p.Stale || p.Question != text || p.Answer != "새 답" {
		t.Fatalf("patch by external id = %+v (%v)", p, err)
	}
	on := true
	if p, err = s.PatchCard(ctx, testUser, c.ID, CardPatch{Stale: &on}, now); err != nil || !p.Stale {
		t.Fatalf("patch by id = %+v (%v)", p, err)
	}
	if _, err := s.PatchCard(ctx, "someone", c.ID, CardPatch{Stale: &on}, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("patch of another user's card: %v", err)
	}
	if _, err := s.PatchExternalCard(ctx, testUser, "atlas:none", CardPatch{Stale: &on}, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("patch of a missing external id: %v", err)
	}
	blank := " "
	if _, err := s.PatchCard(ctx, testUser, c.ID, CardPatch{Answer: &blank}, now); !errors.Is(err, ErrInvalidCard) {
		t.Fatalf("blank answer: %v", err)
	}
}

func TestExternalCardValidation(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	now := time.Now()
	for _, in := range []ExternalCard{
		{ExternalID: "", Question: "q", Answer: "a"},
		{ExternalID: "no-app", Question: "q", Answer: "a"},
		{ExternalID: "atlas:a/b", Question: "q", Answer: "a"},
		{ExternalID: "atlas:x", SourceURL: "javascript:alert(1)", Question: "q", Answer: "a"},
		{ExternalID: "atlas:x", SourceURL: "/relative", Question: "q", Answer: "a"},
		{ExternalID: "atlas:x", Question: "", Answer: "a"},
	} {
		if _, err := s.SaveExternalCard(ctx, testUser, in, now); !errors.Is(err, ErrInvalidCard) {
			t.Errorf("%+v: %v", in, err)
		}
	}
	note := NewID()
	if _, err := s.c().exec(ctx, `INSERT INTO notes (id, user_id, anchor, quote, body) VALUES ($1, $2, 'a', '인용', '본문')`, note, testUser); err != nil {
		t.Fatal(err)
	}
	c, err := s.SaveCard(ctx, testUser, "note", note, "q", "a", now)
	if err != nil {
		t.Fatal(err)
	}
	on := true
	if _, err := s.PatchCard(ctx, testUser, c.ID, CardPatch{Stale: &on}, now); !errors.Is(err, ErrInvalidCard) {
		t.Fatalf("stale on a note card: %v", err)
	}
}
