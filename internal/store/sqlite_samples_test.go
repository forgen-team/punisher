package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSeedSample(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	var ids, texts []string
	rows, err := collect(s.c().query(ctx, `SELECT id, text FROM objectives WHERE concept_id = 'net.network-layers' AND status = 'active' ORDER BY ordinal`),
		func(r rowScanner) ([2]string, error) {
			var p [2]string
			err := r.Scan(&p[0], &p[1])
			return p, err
		})
	if err != nil || len(rows) == 0 {
		t.Fatalf("objectives: %v %v", rows, err)
	}
	for _, r := range rows {
		ids, texts = append(ids, r[0]), append(texts, r[1])
	}
	sm := Sample{ConceptID: "net.network-layers", ObjectiveIDs: ids, ObjectiveTexts: texts, Depth: 1, Title: "맛보기",
		Content: json.RawMessage(`{"schemaVersion":1,"title":"맛보기","sections":[]}`), ContentHash: "h", ReviewModel: "m",
		CheckItems: []SampleCheckItem{
			{ObjectiveID: ids[0], Purpose: "check", Format: "recall", Prompt: "q?", Rubric: json.RawMessage(`{"must":["a"]}`)},
			{Purpose: "check", Format: "integrative", Prompt: "q2?", Rubric: json.RawMessage(`{"must":["b"]}`)},
		}}

	changed := sm
	changed.ObjectiveTexts = append([]string{"다른 문장"}, texts[1:]...)
	if added, err := s.SeedSample(ctx, changed); err != nil || added {
		t.Fatalf("changed curriculum must not seed: %v %v", added, err)
	}
	if added, err := s.SeedSample(ctx, sm); err != nil || !added {
		t.Fatalf("seed: %v %v", added, err)
	}
	if added, err := s.SeedSample(ctx, sm); err != nil || added {
		t.Fatalf("a concept with a published lecture is left alone: %v %v", added, err)
	}
	var id, kind, verdict string
	var items int
	if err := s.c().row(ctx, `SELECT l.id, l.kind, r.verdict, (SELECT COUNT(*) FROM check_items ci WHERE ci.lecture_id = l.id)
		FROM lectures l JOIN lecture_reviews r ON r.lecture_id = l.id WHERE l.concept_id = 'net.network-layers'`).Scan(&id, &kind, &verdict, &items); err != nil {
		t.Fatal(err)
	}
	if kind != "library" || verdict != "pass" || items != 2 {
		t.Fatalf("seeded: kind=%s review=%s items=%d", kind, verdict, items)
	}
	dumped, err := s.DumpSample(ctx, id)
	if err != nil || dumped.Title != "맛보기" || len(dumped.CheckItems) != 2 || dumped.CheckItems[1].ObjectiveID != "" || dumped.Language != "ko" {
		t.Fatalf("dump = %+v %v", dumped, err)
	}

	// The English sample of the same concept is its own lecture (i18n L9).
	en := sm
	en.Title, en.Language = "Sample", "en"
	if added, err := s.SeedSample(ctx, en); err != nil || !added {
		t.Fatalf("en sample beside the ko one: %v %v", added, err)
	}
	var lang string
	var enItems int
	if err := s.c().row(ctx, `SELECT l.language, (SELECT COUNT(*) FROM check_items ci WHERE ci.lecture_id = l.id AND ci.language = 'en')
		FROM lectures l WHERE l.concept_id = 'net.network-layers' AND l.title = 'Sample'`).Scan(&lang, &enItems); err != nil || lang != "en" || enItems != 2 {
		t.Fatalf("en sample: %q items=%d %v", lang, enItems, err)
	}
}
