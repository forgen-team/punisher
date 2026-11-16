package services

import (
	"context"
	"errors"
	"testing"
)

// TestMasteryBatch: the map's batch summary matches the single-concept payload
// and leaves out unknown concepts.
func TestMasteryBatch(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	checkBoth(t, s, fake, user, f)
	one, err := s.Mastery(ctx, user, f.conceptID)
	if err != nil {
		t.Fatal(err)
	}
	list, err := s.MasteryBatch(ctx, user, []string{"test.missing", f.conceptID})
	if err != nil || len(list) != 1 {
		t.Fatalf("batch = %+v (%v)", list, err)
	}
	if list[0] != one.ConceptMasterySummary {
		t.Fatalf("batch %+v != single %+v", list[0], one.ConceptMasterySummary)
	}
	if one.Level == 0 || one.MasteryTarget != 4 || one.UncheckedCount != 0 {
		t.Fatalf("checked concept summary = %+v", one.ConceptMasterySummary)
	}
}

// TestCheckInputCodes: each bad input carries its own code.
func TestCheckInputCodes(t *testing.T) {
	s, _, user, f := checkService(t)
	ctx := context.Background()
	code := func(err error) string {
		var ie *CheckInputError
		if !errors.As(err, &ie) || !errors.Is(err, ErrInvalidCheck) {
			t.Fatalf("not a coded input error: %v", err)
		}
		return ie.Code
	}
	if _, _, err := s.StartCheck(ctx, user, "review", f.lectureID); code(err) != CodeInvalidKind {
		t.Fatalf("kind: %v", err)
	}
	if _, err := s.AnswerCheck(ctx, user, "x", "  ", ""); code(err) != CodeAnswerEmpty {
		t.Fatalf("empty: %v", err)
	}
	long := make([]rune, MaxCheckAnswerRunes+1)
	for i := range long {
		long[i] = '가'
	}
	if _, err := s.AnswerCheck(ctx, user, "x", string(long), ""); code(err) != CodeAnswerTooLong {
		t.Fatalf("too long: %v", err)
	}
	if _, err := s.SelfGradeCheck(ctx, user, "x", "maybe"); code(err) != CodeInvalidVerdict {
		t.Fatalf("verdict: %v", err)
	}
	if _, err := testDB(s).ExecContext(ctx, `UPDATE objectives SET status = 'retired' WHERE concept_id = ?`, f.conceptID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.StartCheck(ctx, user, "check", f.lectureID); code(err) != CodeNoObjectives {
		t.Fatalf("no objectives: %v", err)
	}
}
