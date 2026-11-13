package services

import (
	"context"
	"testing"
)

// TestCheckHistoryIncludesKnownVerify: '이미 앎' verification sessions are history
// entries with their concept (kind filter known_verify too); the pretest is not.
func TestCheckHistoryIncludesKnownVerify(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	fake.push("items", pretestSetJSON(f))
	if d := startDiagnostic(t, s, user, DiagnosticRequest{Kind: "pretest", LectureID: f.lectureID}); d == nil {
		t.Fatal("pretest session expected")
	}
	if _, err := s.MarkKnown(ctx, user, f.conceptID, false); err != nil {
		t.Fatal(err)
	}
	fake.push("items", verifySetJSON(f.objA, f.objB))
	v := startDiagnostic(t, s, user, DiagnosticRequest{Kind: "known_verify", ConceptID: f.conceptID})
	if v == nil {
		t.Fatal("known_verify session expected")
	}
	all, _, err := s.Store.ListCheckHistory(ctx, user, "", 10, "")
	if err != nil || len(all) != 1 {
		t.Fatalf("history = %+v (%v): only the verification, never the pretest", all, err)
	}
	if h := all[0]; h.ID != v.Session.ID || h.Kind != "known_verify" || h.ConceptID != f.conceptID || h.ConceptName != "페이지 캐시" {
		t.Fatalf("known_verify entry = %+v", h)
	}
	if only, _, err := s.Store.ListCheckHistory(ctx, user, "known_verify", 10, ""); err != nil || len(only) != 1 {
		t.Fatalf("kind=known_verify = %+v (%v)", only, err)
	}
	if none, _, err := s.Store.ListCheckHistory(ctx, user, "pretest", 10, ""); err != nil || len(none) != 0 {
		t.Fatalf("kind=pretest must stay out of the history: %+v (%v)", none, err)
	}
}
