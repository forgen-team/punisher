package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/forgen-team/punisher/internal/curriculum"
)

// SyncCurriculum upserts the seed in a single transaction. It is a no-op when
// the seed version was already applied. Seed concepts/objectives missing from
// the seed are marked retired (never hard-deleted); seed aliases and edges are
// replaced wholesale. Dynamic (proposed) rows are left untouched.
// Returns true when the seed was applied.
func (s *SQLiteStore) SyncCurriculum(ctx context.Context, seed *curriculum.Seed) (bool, error) {
	applied := false
	err := s.tx(ctx, func(c conn) error {
		// Compare with the latest applied version (not "ever applied") so that
		// reverting the seed to an older version re-applies it.
		var latest string
		err := c.row(ctx, "SELECT version FROM curriculum_versions ORDER BY applied_at DESC LIMIT 1").Scan(&latest)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check curriculum version: %w", err)
		}
		if latest == seed.Version {
			return nil
		}
		if err := applySeed(ctx, c, seed); err != nil {
			return err
		}
		if err := markObjectiveChanges(ctx, c); err != nil {
			return err
		}
		applied = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return applied, nil
}

func applySeed(ctx context.Context, c conn, seed *curriculum.Seed) error {
	conceptIDs := make([]string, 0, len(seed.Concepts))
	var objectiveIDs []string

	for _, cc := range seed.Concepts {
		cases, err := json.Marshal(nonNil(cc.PracticalCases))
		if err != nil {
			return fmt.Errorf("marshal practical cases for %s: %w", cc.ID, err)
		}
		links, err := json.Marshal(nonNilLinks(cc.DocLinks))
		if err != nil {
			return fmt.Errorf("marshal doc links for %s: %w", cc.ID, err)
		}
		casesEn, err := json.Marshal(nonNil(cc.PracticalCasesEn))
		if err != nil {
			return fmt.Errorf("marshal English practical cases for %s: %w", cc.ID, err)
		}
		conceptIDs = append(conceptIDs, cc.ID)
		if _, err := c.exec(ctx, `
			INSERT INTO concepts (id, name, area, depth, summary, practical_cases, doc_links,
			                      status, origin, curriculum_version, name_en, summary_en, practical_cases_en)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'active', 'seed', $8, $9, $10, $11)
			ON CONFLICT (id) DO UPDATE SET
				name = excluded.name, area = excluded.area, depth = excluded.depth,
				summary = excluded.summary, practical_cases = excluded.practical_cases,
				doc_links = excluded.doc_links, status = 'active', origin = 'seed',
				curriculum_version = excluded.curriculum_version, name_en = excluded.name_en,
				summary_en = excluded.summary_en, practical_cases_en = excluded.practical_cases_en, updated_at = NOW()`,
			cc.ID, cc.Name, cc.Area, cc.Depth, cc.Summary, string(cases), string(links), seed.Version,
			cc.NameEn, cc.SummaryEn, string(casesEn)); err != nil {
			return fmt.Errorf("upsert concept %s: %w", cc.ID, err)
		}

		for i, o := range cc.Objectives {
			id := curriculum.ObjectiveID(cc.ID, i+1)
			objectiveIDs = append(objectiveIDs, id)
			if _, err := c.exec(ctx, `
				INSERT INTO objectives (id, concept_id, text, depth, ordinal, status, text_en)
				VALUES ($1, $2, $3, $4, $5, 'active', $6)
				ON CONFLICT (id) DO UPDATE SET
					text = excluded.text, depth = excluded.depth, ordinal = excluded.ordinal,
					status = 'active', text_en = excluded.text_en, updated_at = NOW()`,
				id, cc.ID, o.Text, o.Depth, i+1, o.TextEn); err != nil {
				return fmt.Errorf("upsert objective %s: %w", id, err)
			}
		}
	}

	if _, err := c.exec(ctx, `UPDATE concepts SET status = 'retired', updated_at = NOW()
		WHERE origin = 'seed' AND status <> 'retired' AND id NOT IN (SELECT value FROM json_each($1))`, conceptIDs); err != nil {
		return fmt.Errorf("retire concepts: %w", err)
	}
	if _, err := c.exec(ctx, `UPDATE objectives SET status = 'retired', updated_at = NOW()
		WHERE status <> 'retired' AND id NOT IN (SELECT value FROM json_each($1))
		  AND concept_id IN (SELECT id FROM concepts WHERE origin = 'seed')`, nonNil(objectiveIDs)); err != nil {
		return fmt.Errorf("retire objectives: %w", err)
	}

	if _, err := c.exec(ctx, `DELETE FROM concept_aliases WHERE origin = 'seed'`); err != nil {
		return fmt.Errorf("clear aliases: %w", err)
	}
	for _, a := range seed.Aliases {
		if _, err := c.exec(ctx, `
			INSERT INTO concept_aliases (alias, display, concept_id, origin)
			VALUES ($1, $2, $3, 'seed')
			ON CONFLICT (alias) DO UPDATE SET
				display = excluded.display, concept_id = excluded.concept_id, origin = 'seed'`,
			a.Alias, a.Display, a.ConceptID); err != nil {
			return fmt.Errorf("upsert alias %s: %w", a.Alias, err)
		}
	}

	if _, err := c.exec(ctx, `DELETE FROM concept_edges WHERE origin = 'seed'`); err != nil {
		return fmt.Errorf("clear edges: %w", err)
	}
	for _, e := range seed.Edges {
		if _, err := c.exec(ctx, `
			INSERT INTO concept_edges (from_id, to_id, kind, origin)
			VALUES ($1, $2, $3, 'seed')
			ON CONFLICT (from_id, to_id, kind) DO UPDATE SET origin = 'seed'`,
			e.From, e.To, e.Kind); err != nil {
			return fmt.Errorf("upsert edge %s→%s: %w", e.From, e.To, err)
		}
	}

	if _, err := c.exec(ctx, `INSERT INTO curriculum_versions (version, concept_count, edge_count, alias_count)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (version) DO UPDATE SET applied_at = NOW()`,
		seed.Version, len(seed.Concepts), len(seed.Edges), len(seed.Aliases)); err != nil {
		return fmt.Errorf("record curriculum version: %w", err)
	}
	return nil
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func nonNilLinks(v []curriculum.DocLink) []curriculum.DocLink {
	if v == nil {
		return []curriculum.DocLink{}
	}
	return v
}

// LatestCurriculumVersion returns the most recently applied seed version ("" if none).
func (s *SQLiteStore) LatestCurriculumVersion(ctx context.Context) (string, error) {
	var version string
	err := s.c().row(ctx,
		"SELECT version FROM curriculum_versions ORDER BY applied_at DESC LIMIT 1",
	).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return version, err
}
