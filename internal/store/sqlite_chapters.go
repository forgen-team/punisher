package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wooo-jin/punisher/internal/chapter"
	"github.com/wooo-jin/punisher/internal/learning"
)

// ChapterInfo is one of the user's chapters (map grouping).
type ChapterInfo struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	ConceptCount int       `json:"conceptCount"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ChapterSignals loads suggestion sources (a) and (b) and the names already on
// the user's map.
func (s *SQLiteStore) ChapterSignals(ctx context.Context, userID string) (proposed []learning.ProposedSignal, deepen []learning.DeepenSignal, covered map[string]bool, err error) {
	// (a) the user's proposed concepts; mentions = lectures the user opened that
	// name the concept (term block or inline link) + questions proposing it by name.
	if proposed, err = collect(s.c().query(ctx, `
		SELECT c.id, c.name, c.area,
			(SELECT COUNT(*) FROM lectures l JOIN user_lecture_progress p ON p.lecture_id = l.id AND p.user_id = $1
			 WHERE l.content LIKE '%' || c.id || '%')
			+ (SELECT COUNT(*) FROM lecture_questions q WHERE q.user_id = $1 AND q.proposed_concepts LIKE '%' || c.name || '%')
			+ (SELECT COUNT(*) FROM inquiries i WHERE i.user_id = $1 AND EXISTS (SELECT 1 FROM json_each(i.proposed_concept_ids) WHERE value = c.id))
		FROM concepts c
		WHERE c.status = 'proposed' AND c.proposed_by = $1
		ORDER BY c.created_at DESC LIMIT 50`, userID), func(r rowScanner) (learning.ProposedSignal, error) {
		var p learning.ProposedSignal
		err := r.Scan(&p.ID, &p.Name, &p.Area, &p.Mentions)
		return p, err
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("chapter proposed signals: %w", err)
	}

	// (b) finished concepts with deeper (depth 2-3) objectives still pending.
	if deepen, err = collect(s.c().query(ctx, `
		SELECT c.id, c.name, c.area, COUNT(o.id)
		FROM concepts c
		JOIN user_concept_states st ON st.concept_id = c.id AND st.user_id = $1 AND st.state IN ('completed', 'known')
		JOIN objectives o ON o.concept_id = c.id AND o.status = 'active' AND o.depth >= 2
		WHERE c.status = 'active' AND `+visibleTo("c", "$1")+`
		  AND NOT EXISTS (SELECT 1 FROM user_objective_states us
		                  WHERE us.user_id = $1 AND us.objective_id = o.id AND us.state IN ('completed', 'known'))
		GROUP BY c.id, c.name, c.area
		ORDER BY COUNT(o.id) DESC, c.id LIMIT 20`, userID), func(r rowScanner) (learning.DeepenSignal, error) {
		var d learning.DeepenSignal
		err := r.Scan(&d.ConceptID, &d.Name, &d.Area, &d.Remaining)
		return d, err
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("chapter deepen signals: %w", err)
	}

	list, err := collect(s.c().query(ctx, `
		SELECT c.name FROM concepts c WHERE c.status = 'active' AND `+visibleTo("c", "$1"), userID), scanString)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("covered names: %w", err)
	}
	covered = make(map[string]bool, len(list))
	for _, n := range list {
		covered[chapter.NormalizeName(n)] = true
	}
	return proposed, deepen, covered, nil
}

// ChapterContext loads what chapter.Validate needs for the user.
func (s *SQLiteStore) ChapterContext(ctx context.Context, userID string) (chapter.Context, error) {
	c := chapter.Context{UserID: userID, Visible: map[string]string{}, OwnProposed: map[string]string{}}
	type row struct{ id, name, status string }
	list, err := collect(s.c().query(ctx, `
		SELECT c.id, c.name, c.status FROM concepts c
		WHERE c.status IN ('active', 'proposed') AND `+visibleTo("c", "$1"), userID), func(r rowScanner) (row, error) {
		var x row
		err := r.Scan(&x.id, &x.name, &x.status)
		return x, err
	})
	if err != nil {
		return c, fmt.Errorf("chapter context: %w", err)
	}
	for _, x := range list {
		if x.status == "proposed" {
			c.OwnProposed[x.id] = x.name // visibleTo keeps only the user's own proposed concepts
		} else {
			c.Visible[x.id] = x.name
		}
	}
	return c, nil
}

// SaveChapterDraft stores a validated draft for preview.
func (s *SQLiteStore) SaveChapterDraft(ctx context.Context, userID string, input any, d chapter.Draft, model string) (string, error) {
	in, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	id := NewID()
	_, err = s.c().exec(ctx, `
		INSERT INTO chapter_drafts (id, user_id, input, draft, model) VALUES ($1, $2, $3, $4, $5)`,
		id, userID, string(in), string(raw), model)
	if err != nil {
		return "", fmt.Errorf("save chapter draft: %w", err)
	}
	return id, nil
}

// ChapterDraft loads one of the user's drafts (applied=true when already opened).
func (s *SQLiteStore) ChapterDraft(ctx context.Context, userID, draftID string) (d chapter.Draft, input json.RawMessage, applied bool, err error) {
	var raw json.RawMessage
	var appliedAt *time.Time
	err = s.c().row(ctx, `
		SELECT draft, input, applied_at FROM chapter_drafts WHERE user_id = $1 AND id = $2`, userID, draftID).
		Scan(rawJSON(&raw), rawJSON(&input), nts(&appliedAt))
	if errors.Is(err, sql.ErrNoRows) {
		return d, nil, false, ErrNotFound
	}
	if err != nil {
		return d, nil, false, fmt.Errorf("load chapter draft: %w", err)
	}
	err = json.Unmarshal(raw, &d)
	return d, input, appliedAt != nil, err
}

// ErrDraftApplied: the draft was already opened as a chapter.
var ErrDraftApplied = errors.New("chapter draft already applied")

// ApplyChapter opens a validated draft as the user's chapter in one
// transaction: creates new concepts (owner = user, origin = chapter),
// promotes the user's proposed concepts, adds objectives and prerequisite
// edges. Shared (seed) concepts are only referenced, never modified.
func (s *SQLiteStore) ApplyChapter(ctx context.Context, userID, draftID string, d chapter.Draft, input json.RawMessage) (string, error) {
	chapterID := NewID()
	err := s.tx(ctx, func(tx conn) error {
		res, err := tx.exec(ctx, `UPDATE chapter_drafts SET applied_at = NOW()
			WHERE id = $1 AND user_id = $2 AND applied_at IS NULL`, draftID, userID)
		if err != nil {
			return fmt.Errorf("lock draft: %w", err)
		}
		if affected(res) == 0 {
			return ErrDraftApplied
		}
		if _, err := tx.exec(ctx, `INSERT INTO chapters (id, user_id, title, summary, input) VALUES ($1, $2, $3, $4, $5)`,
			chapterID, userID, d.Title, d.Summary, nonNullJSON(input, "{}")); err != nil {
			return fmt.Errorf("insert chapter: %w", err)
		}
		for _, c := range d.Concepts {
			if err := applyChapterConcept(ctx, tx, userID, chapterID, c); err != nil {
				return err
			}
		}
		_, err = tx.exec(ctx, `UPDATE chapter_drafts SET chapter_id = $1 WHERE id = $2`, chapterID, draftID)
		return err
	})
	if err != nil {
		return "", err
	}
	return chapterID, nil
}

func applyChapterConcept(ctx context.Context, tx conn, userID, chapterID string, c chapter.Concept) error {
	switch c.Action {
	case chapter.ActionReuse:
		return nil
	case chapter.ActionCreate:
		if _, err := tx.exec(ctx, `
			INSERT INTO concepts (id, name, area, depth, summary, status, origin, owner_user_id, chapter_id)
			VALUES ($1, $2, $3, $4, $5, 'active', 'chapter', $6, $7)`,
			c.ID, c.Name, c.Area, c.Depth, c.Summary, userID, chapterID); err != nil {
			return fmt.Errorf("create chapter concept %s: %w", c.ID, err)
		}
	case chapter.ActionPromote:
		res, err := tx.exec(ctx, `
			UPDATE concepts SET status = 'active', origin = 'chapter', owner_user_id = $2, chapter_id = $3, updated_at = NOW()
			WHERE id = $1 AND status = 'proposed' AND proposed_by = $2`, c.ID, userID, chapterID)
		if err != nil {
			return fmt.Errorf("promote %s: %w", c.ID, err)
		}
		if affected(res) == 0 {
			return fmt.Errorf("promote %s: %w", c.ID, ErrNotFound)
		}
	default:
		return fmt.Errorf("unknown chapter action %q", c.Action)
	}
	var have int
	if err := tx.row(ctx, `SELECT COALESCE(MAX(ordinal), 0) FROM objectives WHERE concept_id = $1`, c.ID).Scan(&have); err != nil {
		return err
	}
	if have == 0 {
		for i, o := range c.Objectives {
			if _, err := tx.exec(ctx, `
				INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES ($1, $2, $3, $4, $5)`,
				fmt.Sprintf("%s#%d", c.ID, i+1), c.ID, o.Text, o.Depth, i+1); err != nil {
				return fmt.Errorf("objective of %s: %w", c.ID, err)
			}
		}
	}
	for _, p := range c.Prerequisites {
		if _, err := tx.exec(ctx, `
			INSERT INTO concept_edges (from_id, to_id, kind, origin) VALUES ($1, $2, 'prerequisite', 'dynamic')
			ON CONFLICT (from_id, to_id, kind) DO NOTHING`, p, c.ID); err != nil {
			return fmt.Errorf("edge %s → %s: %w", p, c.ID, err)
		}
	}
	return nil
}

// ListChapters returns the user's chapters, newest first.
func (s *SQLiteStore) ListChapters(ctx context.Context, userID string) ([]ChapterInfo, error) {
	out, err := collect(s.c().query(ctx, `
		SELECT ch.id, ch.title, ch.summary,
			(SELECT COUNT(*) FROM concepts c WHERE c.chapter_id = ch.id AND c.owner_user_id = $1), ch.created_at
		FROM chapters ch WHERE ch.user_id = $1 ORDER BY ch.created_at DESC`, userID), func(r rowScanner) (ChapterInfo, error) {
		var c ChapterInfo
		err := r.Scan(&c.ID, &c.Title, &c.Summary, &c.ConceptCount, ts(&c.CreatedAt))
		return c, err
	})
	if err != nil {
		return nil, fmt.Errorf("list chapters: %w", err)
	}
	return out, nil
}
