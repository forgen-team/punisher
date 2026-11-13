package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wooo-jin/punisher/internal/services"
)

func TestParseConceptIDs(t *testing.T) {
	ids, msg := parseConceptIDs(" a, b,,a ,c")
	if msg != "" || strings.Join(ids, "|") != "a|b|c" {
		t.Fatalf("ids = %v (%q)", ids, msg)
	}
	if _, msg := parseConceptIDs(" , "); msg == "" {
		t.Fatal("an empty list must be rejected")
	}
	many := make([]string, maxMasteryBatch+1)
	for i := range many {
		many[i] = fmt.Sprintf("c%d", i)
	}
	if _, msg := parseConceptIDs(strings.Join(many, ",")); msg == "" {
		t.Fatal("more than the batch limit must be rejected")
	}
	if ids, msg := parseConceptIDs(strings.Join(many[:maxMasteryBatch], ",")); msg != "" || len(ids) != maxMasteryBatch {
		t.Fatalf("the limit itself is allowed: %d (%q)", len(ids), msg)
	}
}

func TestCheckErrorCode(t *testing.T) {
	coded := fmt.Errorf("answer: %w", &services.CheckInputError{Code: services.CodeAnswerEmpty, Msg: "answer is required"})
	if got := checkErrorCode(coded); got != "ANSWER_EMPTY" {
		t.Fatalf("coded = %s", got)
	}
	if got := checkErrorCode(fmt.Errorf("%w: x", services.ErrInvalidCheck)); got != ErrorBadRequest {
		t.Fatalf("uncoded = %s", got)
	}
}

func TestLectureNotReviewedIs409(t *testing.T) {
	rec := httptest.NewRecorder()
	writeCheckError(rec, httptest.NewRequest(http.MethodPost, "/v1/checks", nil), "start check", services.ErrLectureNotReviewed)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"LECTURE_NOT_REVIEWED"`) {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}
