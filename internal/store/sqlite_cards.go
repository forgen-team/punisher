package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/forgen-team/punisher/internal/learning"
)

// Review cards (W5a): a note or a reading-thread answer the user chose to
// review. The question and answer are what the user confirmed (no LLM). Cards
// are scheduled by FSRS alone (they have no ladder) and self-graded 1·2·3.

// ReviewCard is a card with its schedule.
type ReviewCard struct {
	ID           string     `json:"id"`
	SourceKind   string     `json:"sourceKind"`
	SourceID     string     `json:"sourceId"`
	LectureID    string     `json:"lectureId"`
	ConceptID    string     `json:"conceptId"`
	Anchor       string     `json:"anchor"`
	Question     string     `json:"question"`
	Answer       string     `json:"answer"`
	Reps         int        `json:"reps"`
	Lapses       int        `json:"lapses"`
	NextReviewAt *time.Time `json:"nextReviewAt"`
	SuspendedAt  *time.Time `json:"suspendedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	// External cards (sourceKind "external", sent by another app such as atlas):
	// the link back to the source, the sender's id and "the evidence changed".
	SourceURL  string `json:"sourceUrl"`
	ExternalID string `json:"externalId"`
	Stale      bool   `json:"stale"`
}

// Card limits.
const (
	CardQuestionMax = 500
	CardAnswerMax   = 2000
	// CardFirstReviewDays: a new card is first asked the next day (reading it just now is not recall).
	CardFirstReviewDays = 1
)

// ErrInvalidCard: empty or too long question/answer, or an unknown source.
var ErrInvalidCard = errors.New("invalid review card")

// CardDraft is the prefilled question/answer of a source (the user edits it).
type CardDraft struct {
	SourceKind string `json:"sourceKind"`
	SourceID   string `json:"sourceId"`
	LectureID  string `json:"lectureId"`
	ConceptID  string `json:"conceptId"`
	Anchor     string `json:"anchor"`
	Question   string `json:"question"`
	Answer     string `json:"answer"`
	// CardID: the card already made from this source ("" = none yet).
	CardID string `json:"cardId"`
}

const cardColumns = `id, source_kind, source_id, COALESCE(lecture_id, ''), COALESCE(concept_id, ''), anchor, question, answer,
	reps, lapses, next_review_at, suspended_at, created_at, COALESCE(source_url, ''), COALESCE(external_id, ''), stale`

func scanCard(row rowScanner) (ReviewCard, error) {
	var c ReviewCard
	err := row.Scan(&c.ID, &c.SourceKind, &c.SourceID, &c.LectureID, &c.ConceptID, &c.Anchor, &c.Question, &c.Answer,
		&c.Reps, &c.Lapses, nts(&c.NextReviewAt), nts(&c.SuspendedAt), ts(&c.CreatedAt), &c.SourceURL, &c.ExternalID, &c.Stale)
	return c, err
}

// CardDraftFor reads a note (quote → question context, body → answer) or a
// thread root (quote + question → question, the latest answer → answer).
func (s *SQLiteStore) CardDraftFor(ctx context.Context, userID, kind, sourceID string) (CardDraft, error) {
	d := CardDraft{SourceKind: kind, SourceID: sourceID}
	var quote, body string
	var err error
	switch kind {
	case "note":
		err = s.c().row(ctx, `SELECT COALESCE(lecture_id, ''), COALESCE(concept_id, ''), anchor, quote, body FROM notes
			WHERE id = $1 AND user_id = $2`, sourceID, userID).Scan(&d.LectureID, &d.ConceptID, &d.Anchor, &quote, &body)
		d.Question, d.Answer = quote, body
	case "thread":
		var question string
		err = s.c().row(ctx, `SELECT COALESCE(q.lecture_id, ''), COALESCE(q.concept_id, ''), q.anchor, q.quote, q.question,
				COALESCE((SELECT f.answer FROM lecture_questions f WHERE (f.id = q.id OR f.parent_id = q.id) AND f.status = 'answered'
				          ORDER BY f.created_at DESC LIMIT 1), '')
			FROM lecture_questions q WHERE q.id = $1 AND q.user_id = $2 AND q.parent_id IS NULL`, sourceID, userID).
			Scan(&d.LectureID, &d.ConceptID, &d.Anchor, &quote, &question, &body)
		d.Question = strings.TrimSpace(question)
		if quote != "" {
			d.Question = strings.TrimSpace("“" + quote + "” — " + question)
		}
		d.Answer = body
	default:
		return d, ErrInvalidCard
	}
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, fmt.Errorf("card source: %w", err)
	}
	d.Question, d.Answer = clip(strings.TrimSpace(d.Question), CardQuestionMax), clip(strings.TrimSpace(d.Answer), CardAnswerMax)
	err = s.c().row(ctx, `SELECT id FROM review_cards WHERE user_id = $1 AND source_kind = $2 AND source_id = $3 AND deleted_at IS NULL`, userID, kind, sourceID).Scan(&d.CardID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return d, err
	}
	return d, nil
}

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// ValidCardText: both filled, within the limits.
func ValidCardText(question, answer string) bool {
	q, a := strings.TrimSpace(question), strings.TrimSpace(answer)
	return q != "" && a != "" && utf8.RuneCountInString(q) <= CardQuestionMax && utf8.RuneCountInString(a) <= CardAnswerMax
}

// SaveCard creates the card of a source (or updates its text when it exists).
// A new card is first due CardFirstReviewDays later. A card the user deleted
// (deleted_at set) comes back with its schedule and history.
func (s *SQLiteStore) SaveCard(ctx context.Context, userID, kind, sourceID, question, answer string, now time.Time) (ReviewCard, error) {
	if !ValidCardText(question, answer) {
		return ReviewCard{}, ErrInvalidCard
	}
	d, err := s.CardDraftFor(ctx, userID, kind, sourceID)
	if err != nil {
		return ReviewCard{}, err
	}
	due := learning.KSTDayStart(now).AddDate(0, 0, CardFirstReviewDays)
	return scanCard(s.c().row(ctx, `
		INSERT INTO review_cards (id, user_id, source_kind, source_id, lecture_id, concept_id, anchor, question, answer, next_review_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9, $10, $11, $11)
		ON CONFLICT (user_id, source_kind, source_id) DO UPDATE SET question = excluded.question, answer = excluded.answer,
			suspended_at = NULL, deleted_at = NULL, updated_at = excluded.updated_at
		RETURNING `+cardColumns,
		NewID(), userID, kind, sourceID, d.LectureID, d.ConceptID, d.Anchor, strings.TrimSpace(question), strings.TrimSpace(answer), due, now))
}

// DeleteCard marks a card deleted (deleted_at) — the row and its review log
// stay so RestoreCard can bring it back with its schedule. A deleted card is
// left out of lists, due cards and counts. ErrNotFound when not the user's or
// already deleted.
func (s *SQLiteStore) DeleteCard(ctx context.Context, userID, cardID string, now time.Time) error {
	res, err := s.c().exec(ctx, `UPDATE review_cards SET deleted_at = $3 WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`, cardID, userID, now)
	if err != nil {
		return err
	}
	if affected(res) == 0 {
		return ErrNotFound
	}
	return nil
}

// RestoreCard undoes DeleteCard and returns the card. A card that is not
// deleted is returned as it is (so a repeated undo is harmless). ErrNotFound
// when there is no such card of the user's.
func (s *SQLiteStore) RestoreCard(ctx context.Context, userID, cardID string) (ReviewCard, error) {
	c, err := scanCard(s.c().row(ctx, `UPDATE review_cards SET deleted_at = NULL WHERE id = $1 AND user_id = $2 RETURNING `+cardColumns, cardID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewCard{}, ErrNotFound
	}
	return c, err
}

// CardTrashDays: a deleted card can be restored for this long; after that the
// scheduler removes the card row for good (PurgeDeletedCards).
const CardTrashDays = 30

// PurgeDeletedCards removes the card rows deleted CardTrashDays or more before
// now and keeps their review log: the activity heat map, the day counts and the
// like are history of what the user did and must not shrink. A card restored in
// time has no deleted_at and is never touched. Returns the cards removed.
//
// review_card_log.card_id is ON DELETE CASCADE (the schema is not changed), so
// the delete runs on one pinned connection with foreign keys off — otherwise the
// log would go with the card. The log rows then point at a card that no longer
// exists; every query on the log reads it alone (no JOIN with review_cards), and
// the migration runner ignores exactly this orphan kind in its foreign key check.
func (s *SQLiteStore) PurgeDeletedCards(ctx context.Context, now time.Time) (int64, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return 0, fmt.Errorf("purge deleted cards: %w", err)
	}
	defer conn.Close()
	// PRAGMA foreign_keys is per connection and a no-op inside a transaction.
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return 0, fmt.Errorf("purge deleted cards: foreign keys off: %w", err)
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), `PRAGMA foreign_keys = ON`) }()
	res, err := conn.ExecContext(ctx, `DELETE FROM review_cards WHERE deleted_at IS NOT NULL AND deleted_at <= ?`,
		FormatTime(now.AddDate(0, 0, -CardTrashDays)))
	if err != nil {
		return 0, fmt.Errorf("purge deleted cards: %w", err)
	}
	return res.RowsAffected()
}

// DueCards returns the user's cards due by now (oldest due first, at most limit).
func (s *SQLiteStore) DueCards(ctx context.Context, userID string, now time.Time, limit int) ([]ReviewCard, error) {
	return collect(s.c().query(ctx, `SELECT `+cardColumns+` FROM review_cards
		WHERE user_id = $1 AND deleted_at IS NULL AND suspended_at IS NULL AND next_review_at <= $2 ORDER BY next_review_at, id LIMIT $3`, userID, now, limit), scanCard)
}

// ListCards returns every card of the user, newest first.
func (s *SQLiteStore) ListCards(ctx context.Context, userID string) ([]ReviewCard, error) {
	return collect(s.c().query(ctx, `SELECT `+cardColumns+` FROM review_cards WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC, id`, userID), scanCard)
}

// GradeCard applies a self-grade (1 again · 2 hard · 3 good) with FSRS at the
// user's target retention and logs it. Returns the updated card.
func (s *SQLiteStore) GradeCard(ctx context.Context, userID, cardID string, rating int, now time.Time) (ReviewCard, error) {
	if rating < learning.RatingAgain || rating > learning.RatingGood {
		return ReviewCard{}, ErrInvalidCard
	}
	var out ReviewCard
	err := s.tx(ctx, func(c conn) error {
		var m learning.Memory
		var lapses int
		err := c.row(ctx, `SELECT stability, difficulty, reps, last_review_at, lapses FROM review_cards WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
			cardID, userID).Scan(&m.Stability, &m.Difficulty, &m.Reps, nts(&m.LastReviewAt), &lapses)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		prefs, err := studyPrefs(ctx, c, userID)
		if err != nil {
			return err
		}
		next := learning.NextMemory(m, rating, now)
		if rating == learning.RatingAgain && m.Reps > 0 {
			lapses++
		}
		days := learning.FSRSInterval(next.Stability, prefs.TargetRetention)
		due := learning.KSTDayStart(now).AddDate(0, 0, days)
		if rating == learning.RatingAgain {
			due = now.Add(learning.MinCreditGap) // 틀린 카드는 오늘 안에(12시간 뒤) 다시
		}
		if _, err := c.exec(ctx, `UPDATE review_cards SET stability = $3, difficulty = $4, reps = $5, lapses = $6,
				last_review_at = $7, next_review_at = $8, updated_at = $7 WHERE id = $1 AND user_id = $2`,
			cardID, userID, next.Stability, next.Difficulty, next.Reps, lapses, now, due); err != nil {
			return err
		}
		if _, err := c.exec(ctx, `INSERT INTO review_card_log (card_id, user_id, rating, reviewed_at) VALUES ($1, $2, $3, $4)`,
			cardID, userID, rating, now); err != nil {
			return err
		}
		out, err = scanCard(c.row(ctx, `SELECT `+cardColumns+` FROM review_cards WHERE id = $1`, cardID))
		return err
	})
	return out, err
}

// CardCounts: cards due now (a deleted card is not due) and reviews done today (KST) — the review
// log counts whatever became of the card afterwards.
func (s *SQLiteStore) CardCounts(ctx context.Context, userID string, now time.Time) (due, doneToday int, err error) {
	err = s.c().row(ctx, `SELECT
			(SELECT COUNT(*) FROM review_cards WHERE user_id = $1 AND deleted_at IS NULL AND suspended_at IS NULL AND next_review_at <= $2),
			(SELECT COUNT(*) FROM review_card_log WHERE user_id = $1 AND reviewed_at >= $3)`,
		userID, now, learning.KSTDayStart(now)).Scan(&due, &doneToday)
	return due, doneToday, err
}

// SearchHit is one search result.
type SearchHit struct {
	Kind      string `json:"kind"` // lecture | note | question | inquiry
	RefID     string `json:"refId"`
	LectureID string `json:"lectureId"`
	Anchor    string `json:"anchor"`
	Title     string `json:"title"`
	// SectionTitle: the lecture section the hit is in (lecture hits; "" otherwise) — one lecture can match in several sections.
	SectionTitle string `json:"sectionTitle"`
	Snippet      string `json:"snippet"`
}

// SearchMinFTS: shorter queries cannot use the trigram index and fall back to LIKE.
const SearchMinFTS = 3

// Search finds study material visible to the user: published lectures
// (library or the user's own), the user's notes, questions/threads and
// inquiries. Queries of 3+ characters use FTS5 (ranked); shorter ones LIKE.
// Snippets mark the match with [[ ]].
func (s *SQLiteStore) Search(ctx context.Context, userID, q string, limit int) ([]SearchHit, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []SearchHit{}, nil
	}
	scan := func(row rowScanner) (SearchHit, error) {
		var h SearchHit
		err := row.Scan(&h.Kind, &h.RefID, &h.LectureID, &h.Anchor, &h.Title, &h.Snippet)
		return h, err
	}
	// 강의는 지금도 보이는 것만(다른 사람의 즉석 강의·공개 안 된 판 제외 — 색인 user_id 로 1차, 여기서 다시).
	visible := `(user_id = $1 OR user_id = '')`
	var hits []SearchHit
	var err error
	if utf8.RuneCountInString(q) >= SearchMinFTS {
		phrase := `"` + strings.ReplaceAll(q, `"`, `""`) + `"`
		hits, err = collect(s.c().query(ctx, `SELECT kind, ref_id, lecture_id, anchor, title, snippet(search_index, -1, '[[', ']]', '…', 12)
			FROM search_index WHERE search_index MATCH $2 AND `+visible+` ORDER BY rank LIMIT $3`, userID, phrase, limit), scan)
	} else {
		like := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
		hits, err = collect(s.c().query(ctx, `SELECT kind, ref_id, lecture_id, anchor, title, substr(body, max(1, instr(body, $4) - 30), 120)
			FROM search_index WHERE (title LIKE $2 ESCAPE '\' OR body LIKE $2 ESCAPE '\') AND `+visible+` LIMIT $3`, userID, like, limit, q), scan)
	}
	if err != nil {
		return nil, err
	}
	for i := range hits {
		hits[i].Snippet = NormalizeSnippet(hits[i].Snippet)
	}
	return hits, s.fillSectionTitles(ctx, hits)
}

// NormalizeSnippet makes a search excerpt one line: the body keeps the
// lecture's line breaks (and FTS snippet() keeps them), and a break right
// before the trailing '…' left the ellipsis alone on its own line. Whitespace
// runs become one space and the ellipsis is glued to its word with a no-break
// space so it never wraps alone.
func NormalizeSnippet(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, " …", "\u00a0…")
	return strings.ReplaceAll(s, "… ", "…\u00a0")
}

// fillSectionTitles sets SectionTitle of lecture hits from the lecture's sections (one query per lecture).
func (s *SQLiteStore) fillSectionTitles(ctx context.Context, hits []SearchHit) error {
	titles := map[string]map[string]string{}
	for i, h := range hits {
		if h.Kind != "lecture" || h.LectureID == "" {
			continue
		}
		byAnchor, ok := titles[h.LectureID]
		if !ok {
			type row struct{ anchor, title string }
			rows, err := collect(s.c().query(ctx, `SELECT COALESCE(json_extract(sv.value, '$.anchor'), ''), COALESCE(json_extract(sv.value, '$.title'), '')
				FROM lectures l, json_each(l.content, '$.sections') sv WHERE l.id = $1 AND json_valid(l.content)`, h.LectureID),
				func(r rowScanner) (row, error) {
					var x row
					err := r.Scan(&x.anchor, &x.title)
					return x, err
				})
			if err != nil {
				return fmt.Errorf("search section titles: %w", err)
			}
			byAnchor = make(map[string]string, len(rows))
			for _, x := range rows {
				byAnchor[x.anchor] = x.title
			}
			titles[h.LectureID] = byAnchor
		}
		hits[i].SectionTitle = byAnchor[h.Anchor]
	}
	return nil
}
