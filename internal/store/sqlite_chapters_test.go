package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wooo-jin/punisher/internal/chapter"
	"github.com/wooo-jin/punisher/internal/curriculum"
	"github.com/wooo-jin/punisher/internal/learning"
)

func TestChapterApplyAndUserIsolation(t *testing.T) {
	// The standalone app has one user, but the visibleTo paths are the same as
	// witems: a second user must see none of the first user's chapter.
	s, _ := newTestStore(t)
	ctx := context.Background()
	userA, userB := testUser, "other-user"
	seedA, seedB := twoConcepts(t, s)
	proposedID := "infra.chapter-test-proposed"
	if _, err := s.c().exec(ctx, `
		INSERT INTO concepts (id, name, area, depth, summary, status, origin, proposed_by)
		VALUES ($1, '챕터 테스트 제안 개념', 'infra-ops', 2, 's', 'proposed', 'dynamic', $2)`, proposedID, userA); err != nil {
		t.Fatal(err)
	}

	cctxB, err := s.ChapterContext(ctx, userB)
	if err != nil {
		t.Fatal(err)
	}
	if _, leaked := cctxB.OwnProposed[proposedID]; leaked {
		t.Fatal("another user's proposed concept must not be promotable")
	}

	cctxA, err := s.ChapterContext(ctx, userA)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cctxA.OwnProposed[proposedID]; !ok {
		t.Fatal("own proposed concept missing from chapter context")
	}
	cctxA.Areas = curriculum.Areas
	obj := []chapter.Objective{{Text: "설명한다", Depth: 1}}
	raw := chapter.Draft{Title: "격리 테스트 챕터", Concepts: []chapter.Concept{
		{Key: "alpha", Name: "알파 개념 " + userA, Area: "os", Depth: 1, Objectives: obj, Prerequisites: []string{seedA}},
		{Key: "beta", Name: "베타 개념 " + userA, Area: "os", Depth: 1, Objectives: obj, Prerequisites: []string{"alpha"}},
		{Key: "gamma", Name: "감마 개념 " + userA, Area: "os", Depth: 2, Objectives: obj, Prerequisites: []string{"beta", seedB}},
		{ProposedID: proposedID, Area: "infra-ops", Depth: 2, Objectives: obj, Prerequisites: []string{"gamma"}},
		{ExistingID: seedA},
	}}
	draft, problems := chapter.Validate(raw, cctxA)
	if len(problems) > 0 {
		t.Fatalf("problems: %v", problems)
	}
	draftID, err := s.SaveChapterDraft(ctx, userA, map[string]any{"freeText": "test"}, draft, "fake")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := s.ChapterDraft(ctx, userB, draftID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another user's draft must be hidden: %v", err)
	}
	stored, input, applied, err := s.ChapterDraft(ctx, userA, draftID)
	if err != nil || applied {
		t.Fatalf("draft: %v applied=%v", err, applied)
	}
	chapterID, err := s.ApplyChapter(ctx, userA, draftID, stored, input)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := s.ApplyChapter(ctx, userA, draftID, stored, json.RawMessage(`{}`)); !errors.Is(err, ErrDraftApplied) {
		t.Fatalf("second apply = %v, want ErrDraftApplied", err)
	}

	prefix := chapter.OwnerPrefix(userA)
	alpha, gamma := prefix+".alpha", prefix+".gamma"

	// User A sees the chapter concepts (grouped by chapter) and the promoted one.
	listA, err := s.ListConcepts(ctx, userA, ConceptFilter{Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	inA := map[string]string{}
	for _, c := range listA {
		inA[c.ID] = c.ChapterID
	}
	if inA[alpha] != chapterID || inA[gamma] != chapterID || inA[proposedID] != chapterID {
		t.Fatalf("user A map missing chapter concepts: alpha=%q gamma=%q promoted=%q", inA[alpha], inA[gamma], inA[proposedID])
	}
	if info, err := s.ConceptInfo(ctx, userA, alpha); err != nil || len(info.Objectives) != 1 {
		t.Fatalf("concept info A: %+v %v", info, err)
	}
	edgesA, err := s.PrerequisiteEdges(ctx, userA)
	if err != nil || !hasEdge(edgesA, seedA, alpha) || !hasEdge(edgesA, gamma, proposedID) {
		t.Fatalf("user A edges missing chapter prerequisites (%v)", err)
	}
	todayA, err := s.LoadToday(ctx, userA, time.Now())
	if err != nil || !todayA.Concepts[alpha].Private || todayA.Concepts[alpha].PendingDepths != 1 {
		t.Fatalf("today A: %+v %v", todayA.Concepts[alpha], err)
	}
	chapters, err := s.ListChapters(ctx, userA)
	if err != nil || len(chapters) != 1 || chapters[0].ConceptCount != 4 {
		t.Fatalf("chapters = %+v %v", chapters, err)
	}

	// User B sees none of it.
	listB, _ := s.ListConcepts(ctx, userB, ConceptFilter{})
	for _, c := range listB {
		if strings.HasPrefix(c.ID, prefix) || c.ID == proposedID {
			t.Fatalf("user B sees %s", c.ID)
		}
	}
	if _, err := s.GetConceptDetail(ctx, userB, alpha); !errors.Is(err, ErrNotFound) {
		t.Fatalf("detail for B = %v", err)
	}
	if _, err := s.ConceptInfo(ctx, userB, alpha); !errors.Is(err, ErrNotFound) {
		t.Fatalf("info for B = %v", err)
	}
	knownB, _ := s.KnownConceptIDs(ctx, userB)
	aliasB, _ := s.AliasIndex(ctx, userB)
	catalogB, _ := s.ConceptCatalog(ctx, userB)
	todayB, _ := s.LoadToday(ctx, userB, time.Now())
	edgesB, _ := s.PrerequisiteEdges(ctx, userB)
	graphB, _ := s.LoadGraph(ctx, userB)
	if knownB[alpha] || aliasB[alpha] != "" || todayB.Concepts[alpha].ID != "" || hasEdge(edgesB, seedA, alpha) {
		t.Fatal("user B leaks chapter concepts via known ids / alias index / today / edges")
	}
	for _, c := range catalogB {
		if c.ID == alpha || c.ID == proposedID {
			t.Fatalf("user B catalog leaks %s", c.ID)
		}
	}
	for _, e := range graphB.Edges {
		if e.To == alpha {
			t.Fatal("user B graph leaks chapter edges")
		}
	}
	if ids, _ := s.ResolveConcepts(ctx, userB, alpha); len(ids) != 0 {
		t.Fatalf("user B resolves %v", ids)
	}
	if _, err := s.SetConceptState(ctx, userB, alpha, "known"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("user B can set state on A's concept: %v", err)
	}
	if chB, _ := s.ListChapters(ctx, userB); len(chB) != 0 {
		t.Fatalf("user B chapters = %+v", chB)
	}
}

func hasEdge(edges []learning.PrereqEdge, from, to string) bool {
	for _, e := range edges {
		if e.From == from && e.To == to {
			return true
		}
	}
	return false
}

func TestDailyMinutesSetting(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	st, err := s.GetLearningSettings(ctx, user)
	if err != nil || st.DailyMinutes != 30 || st.ExampleLanguage != "typescript" {
		t.Fatalf("defaults = %+v %v", st, err)
	}
	if st, err = s.SaveLearningSettings(ctx, user, "", 45); err != nil || st.DailyMinutes != 45 || st.ExampleLanguage != "typescript" {
		t.Fatalf("save daily = %+v %v", st, err)
	}
	if st, err = s.SaveLearningSettings(ctx, user, "go", 0); err != nil || st.DailyMinutes != 45 || st.ExampleLanguage != "go" {
		t.Fatalf("save lang keeps daily = %+v %v", st, err)
	}
}
