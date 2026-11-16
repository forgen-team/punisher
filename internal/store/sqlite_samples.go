package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wooo-jin/punisher/internal/learning"
)

// Sample is a shipped lecture (curriculum/samples/*.json): the lecture row,
// its passing review and its check items (design v3 §9 O3).
type Sample struct {
	ConceptID        string            `json:"conceptId"`
	ObjectiveIDs     []string          `json:"objectiveIds"`
	ObjectiveTexts   []string          `json:"objectiveTexts"`
	Depth            int               `json:"depth"`
	Title            string            `json:"title"`
	Content          json.RawMessage   `json:"content"`
	ContentHash      string            `json:"contentHash"`
	Sources          json.RawMessage   `json:"sources"`
	Model            string            `json:"model"`
	PromptVersion    string            `json:"promptVersion"`
	EstimatedMinutes int               `json:"estimatedMinutes"`
	ReviewModel      string            `json:"reviewModel"`
	ReviewSummary    string            `json:"reviewSummary"`
	CheckItems       []SampleCheckItem `json:"checkItems"`
}

// SampleCheckItem is one check/review item of a sample lecture.
type SampleCheckItem struct {
	ObjectiveID string          `json:"objectiveId"`
	Purpose     string          `json:"purpose"`
	Format      string          `json:"format"`
	Prompt      string          `json:"prompt"`
	Material    json.RawMessage `json:"material"`
	Rubric      json.RawMessage `json:"rubric"`
	Model       string          `json:"model"`
}

// SeedSample inserts a sample as a published library lecture when its concept
// exists, its objectives are the concept's current ones (same ids and texts)
// and the concept has no published lecture yet. Returns whether it was added.
func (s *SQLiteStore) SeedSample(ctx context.Context, sm Sample) (bool, error) {
	added := false
	err := s.tx(ctx, func(c conn) error {
		var published int
		if err := c.row(ctx, `SELECT COUNT(*) FROM lectures WHERE concept_id = $1 AND status = 'published'`, sm.ConceptID).Scan(&published); err != nil {
			return err
		}
		if published > 0 {
			return nil
		}
		for i, id := range sm.ObjectiveIDs {
			var text string
			err := c.row(ctx, `SELECT text FROM objectives WHERE id = $1 AND concept_id = $2 AND status = 'active'`, id, sm.ConceptID).Scan(&text)
			if err != nil || i >= len(sm.ObjectiveTexts) || strings.TrimSpace(text) != strings.TrimSpace(sm.ObjectiveTexts[i]) {
				return nil // 커리큘럼이 바뀌었으면 넣지 않는다.
			}
		}
		id := NewID()
		if _, err := c.exec(ctx, `
			INSERT INTO lectures (id, concept_id, objective_ids, objective_key, depth, kind, status, version,
				title, content, content_hash, sources, model, prompt_version, estimated_minutes, published_at, minutes_estimated)
			VALUES ($1, $2, $3, $4, $5, 'library', 'published',
				(SELECT COALESCE(MAX(version), 0) + 1 FROM lectures WHERE concept_id = $2 AND objective_key = $4 AND depth = $5),
				$6, $7, $8, $9, $10, $11, $12, NOW(), 1)`,
			id, sm.ConceptID, sm.ObjectiveIDs, learning.ObjectiveKey(sm.ObjectiveIDs), sm.Depth,
			sm.Title, string(sm.Content), sm.ContentHash, nonNullJSON(sm.Sources, "[]"), sm.Model, sm.PromptVersion, sm.EstimatedMinutes); err != nil {
			return fmt.Errorf("insert sample lecture: %w", err)
		}
		if _, err := c.exec(ctx, `INSERT INTO lecture_reviews (id, lecture_id, verdict, model, summary) VALUES ($1, $2, 'pass', $3, $4)`,
			NewID(), id, sm.ReviewModel, sm.ReviewSummary); err != nil {
			return fmt.Errorf("insert sample review: %w", err)
		}
		for _, it := range sm.CheckItems {
			// 통합 문항(objective_id 빈 값)은 목표 없이 개념에 단다.
			if _, err := c.exec(ctx, `
				INSERT INTO check_items (id, concept_id, objective_id, lecture_id, purpose, format, prompt, material, rubric,
					objective_text, objective_depth, model)
				VALUES ($1, $10, NULLIF($9, ''), $2, $3, $4, $5, $6, $7,
					COALESCE((SELECT text FROM objectives WHERE id = $9), ''), COALESCE((SELECT depth FROM objectives WHERE id = $9), 0), $8)`,
				NewID(), id, it.Purpose, it.Format, it.Prompt, nonNullJSON(it.Material, "{}"), nonNullJSON(it.Rubric, "{}"), it.Model,
				it.ObjectiveID, sm.ConceptID); err != nil {
				return fmt.Errorf("insert sample check item: %w", err)
			}
		}
		added = true
		return nil
	})
	return added, err
}

// DumpSample reads a published lecture with its latest passing review and
// default check items as a Sample (the dev command `punisher samples dump`).
func (s *SQLiteStore) DumpSample(ctx context.Context, lectureID string) (Sample, error) {
	var sm Sample
	var content, sources, ids string
	err := s.c().row(ctx, `SELECT concept_id, objective_ids, depth, title, content, content_hash, sources, model, prompt_version, estimated_minutes
		FROM lectures WHERE id = $1 AND status = 'published'`, lectureID).Scan(&sm.ConceptID, &ids, &sm.Depth, &sm.Title, &content,
		&sm.ContentHash, &sources, &sm.Model, &sm.PromptVersion, &sm.EstimatedMinutes)
	if err != nil {
		return sm, fmt.Errorf("lecture %s: %w", lectureID, err)
	}
	sm.Content, sm.Sources = json.RawMessage(content), json.RawMessage(sources)
	if err := json.Unmarshal([]byte(ids), &sm.ObjectiveIDs); err != nil {
		return sm, err
	}
	for _, id := range sm.ObjectiveIDs {
		var text string
		if err := s.c().row(ctx, `SELECT text FROM objectives WHERE id = $1`, id).Scan(&text); err != nil {
			return sm, err
		}
		sm.ObjectiveTexts = append(sm.ObjectiveTexts, text)
	}
	if err := s.c().row(ctx, `SELECT model, summary FROM lecture_reviews WHERE lecture_id = $1 AND verdict = 'pass' ORDER BY created_at DESC LIMIT 1`,
		lectureID).Scan(&sm.ReviewModel, &sm.ReviewSummary); err != nil {
		return sm, fmt.Errorf("lecture %s has no passing review: %w", lectureID, err)
	}
	items, err := collect(s.c().query(ctx, `SELECT COALESCE(objective_id, ''), purpose, format, prompt, material, rubric, model FROM check_items
		WHERE lecture_id = $1 AND prompt_hash = '' AND status = 'active' AND purpose IN ('check', 'review')
		ORDER BY rowid`, lectureID), func(r rowScanner) (SampleCheckItem, error) {
		var it SampleCheckItem
		var material, rubric string
		err := r.Scan(&it.ObjectiveID, &it.Purpose, &it.Format, &it.Prompt, &material, &rubric, &it.Model)
		it.Material, it.Rubric = json.RawMessage(material), json.RawMessage(rubric)
		return it, err
	})
	sm.CheckItems = items
	return sm, err
}
