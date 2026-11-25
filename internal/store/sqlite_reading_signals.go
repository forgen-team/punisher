package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// ReadingSignal is one block's reading record: time it was the block in view
// and the term popovers opened from it (concept id → times).
type ReadingSignal struct {
	Anchor    string         `json:"anchor"`
	DwellMs   int64          `json:"dwellMs"`
	TermOpens int            `json:"termOpens"`
	Terms     map[string]int `json:"terms,omitempty"`
}

// AddReadingSignals adds a batch to the user's per-block totals of a lecture.
func (s *SQLiteStore) AddReadingSignals(ctx context.Context, userID, lectureID string, batch []ReadingSignal) error {
	return s.tx(ctx, func(c conn) error {
		for _, in := range batch {
			terms := map[string]int{}
			var raw string
			var dwell int64
			var opens int
			err := c.row(ctx, `SELECT dwell_ms, term_opens, terms FROM lecture_reading_signals
				WHERE user_id = $1 AND lecture_id = $2 AND anchor = $3`, userID, lectureID, in.Anchor).Scan(&dwell, &opens, &raw)
			switch {
			case errors.Is(err, sql.ErrNoRows):
			case err != nil:
				return fmt.Errorf("reading signals: %w", err)
			default:
				if err := json.Unmarshal([]byte(raw), &terms); err != nil {
					terms = map[string]int{}
				}
			}
			for id, n := range in.Terms {
				terms[id] += n
				opens += n
			}
			encoded, err := json.Marshal(terms)
			if err != nil {
				return err
			}
			if _, err := c.exec(ctx, `INSERT INTO lecture_reading_signals (user_id, lecture_id, anchor, dwell_ms, term_opens, terms, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (user_id, lecture_id, anchor) DO UPDATE SET dwell_ms = excluded.dwell_ms, term_opens = excluded.term_opens,
					terms = excluded.terms, updated_at = excluded.updated_at`,
				userID, lectureID, in.Anchor, dwell+in.DwellMs, opens, string(encoded), clockNow()); err != nil {
				return fmt.Errorf("save reading signals: %w", err)
			}
		}
		return nil
	})
}

// LectureReadingSignals are the user's per-block reading totals of a lecture, by anchor.
func (s *SQLiteStore) LectureReadingSignals(ctx context.Context, userID, lectureID string) ([]ReadingSignal, error) {
	list, err := collect(s.c().query(ctx, `SELECT anchor, dwell_ms, term_opens, terms FROM lecture_reading_signals
		WHERE user_id = $1 AND lecture_id = $2 ORDER BY anchor`, userID, lectureID), func(row rowScanner) (ReadingSignal, error) {
		var r ReadingSignal
		var raw string
		if err := row.Scan(&r.Anchor, &r.DwellMs, &r.TermOpens, &raw); err != nil {
			return r, err
		}
		if err := json.Unmarshal([]byte(raw), &r.Terms); err != nil {
			r.Terms = nil
		}
		return r, nil
	})
	if err != nil {
		return nil, fmt.Errorf("lecture reading signals: %w", err)
	}
	return list, nil
}

// GapRecord is what the reader-gap agreement of one lecture is computed from:
// its content, the reader gaps of its latest review that has them, and the
// learner's questions and term popovers per block anchor.
type GapRecord struct {
	LectureID, ConceptID, PromptVersion string
	Content                             json.RawMessage
	ReaderGaps                          json.RawMessage
	Questions                           map[string]int
	TermOpens                           map[string]int
}

// GapRecords returns a record for every lecture with a reviewed reader-gap
// list and at least one learner signal (any user: the standalone app has one).
func (s *SQLiteStore) GapRecords(ctx context.Context) ([]GapRecord, error) {
	recs, err := collect(s.c().query(ctx, `
		SELECT l.id, l.concept_id, l.prompt_version, l.content, r.reader_gaps FROM lectures l
		JOIN lecture_reviews r ON r.id = (SELECT r2.id FROM lecture_reviews r2 WHERE r2.lecture_id = l.id AND r2.reader_gaps <> '[]'
			ORDER BY r2.created_at DESC LIMIT 1)
		WHERE EXISTS (SELECT 1 FROM lecture_questions q WHERE q.lecture_id = l.id AND q.anchor <> '')
		   OR EXISTS (SELECT 1 FROM lecture_reading_signals g WHERE g.lecture_id = l.id AND g.term_opens > 0)
		ORDER BY l.created_at, l.id`), func(row rowScanner) (GapRecord, error) {
		var g GapRecord
		err := row.Scan(&g.LectureID, &g.ConceptID, &g.PromptVersion, rawJSON(&g.Content), rawJSON(&g.ReaderGaps))
		return g, err
	})
	if err != nil {
		return nil, fmt.Errorf("gap records: %w", err)
	}
	type count struct {
		anchor string
		n      int
	}
	scan := func(row rowScanner) (count, error) {
		var c count
		err := row.Scan(&c.anchor, &c.n)
		return c, err
	}
	for i := range recs {
		recs[i].Questions, recs[i].TermOpens = map[string]int{}, map[string]int{}
		qs, err := collect(s.c().query(ctx, `SELECT anchor, COUNT(*) FROM lecture_questions WHERE lecture_id = $1 AND anchor <> '' GROUP BY anchor`, recs[i].LectureID), scan)
		if err != nil {
			return nil, fmt.Errorf("gap questions: %w", err)
		}
		for _, c := range qs {
			recs[i].Questions[c.anchor] = c.n
		}
		ts, err := collect(s.c().query(ctx, `SELECT anchor, SUM(term_opens) FROM lecture_reading_signals WHERE lecture_id = $1 AND term_opens > 0 GROUP BY anchor`, recs[i].LectureID), scan)
		if err != nil {
			return nil, fmt.Errorf("gap term opens: %w", err)
		}
		for _, c := range ts {
			recs[i].TermOpens[c.anchor] = c.n
		}
	}
	return recs, nil
}
