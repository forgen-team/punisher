package store

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/forgen-team/punisher/internal/learning"
)

// External cards: another app (atlas, design D10) sends a card it made from
// confirmed knowledge. They are review cards like the others (FSRS, self-graded)
// with source_kind "external", source_id = external_id (e.g. "atlas:<cardId>",
// unique per user, so sending the same id again updates the card), a link back
// to the source and a stale flag the sender sets when its evidence changed.

// CardSourceExternal is review_cards.source_kind of external cards.
const CardSourceExternal = "external"

// External card limits.
const (
	CardExternalIDMax = 200
	CardSourceURLMax  = 2000
)

// ExternalCard is the input of SaveExternalCard.
type ExternalCard struct {
	ExternalID string
	SourceURL  string
	Question   string
	Answer     string
	Stale      bool
}

// CardPatch changes a card: nil fields are kept. Stale is only for external cards.
type CardPatch struct {
	Question *string
	Answer   *string
	Stale    *bool
}

// ValidExternalID: "app:id" form, printable, within the limit.
func ValidExternalID(id string) bool {
	if id == "" || utf8.RuneCountInString(id) > CardExternalIDMax || strings.TrimSpace(id) != id {
		return false
	}
	app, rest, ok := strings.Cut(id, ":")
	return ok && app != "" && rest != "" && !strings.ContainsAny(id, "/?#\x00\n\r\t ")
}

// ValidSourceURL: empty, or an absolute http(s) URL within the limit (it is shown as a link).
func ValidSourceURL(raw string) bool {
	if raw == "" {
		return true
	}
	if utf8.RuneCountInString(raw) > CardSourceURLMax {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// SaveExternalCard creates the external card of in.ExternalID or, when the
// user already has it, replaces its text, link and stale flag (its schedule is
// kept; a suspended or deleted card comes back). A new card is first due CardFirstReviewDays later.
func (s *SQLiteStore) SaveExternalCard(ctx context.Context, userID string, in ExternalCard, now time.Time) (ReviewCard, error) {
	if !ValidCardText(in.Question, in.Answer) || !ValidExternalID(in.ExternalID) || !ValidSourceURL(in.SourceURL) {
		return ReviewCard{}, ErrInvalidCard
	}
	due := learning.KSTDayStart(now).AddDate(0, 0, CardFirstReviewDays)
	return scanCard(s.c().row(ctx, `
		INSERT INTO review_cards (id, user_id, source_kind, source_id, external_id, source_url, stale, question, answer, next_review_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4, NULLIF($5, ''), $6, $7, $8, $9, $10, $10)
		ON CONFLICT (user_id, source_kind, source_id) DO UPDATE SET question = excluded.question, answer = excluded.answer,
			source_url = excluded.source_url, stale = excluded.stale, suspended_at = NULL, deleted_at = NULL, updated_at = excluded.updated_at
		RETURNING `+cardColumns,
		NewID(), userID, CardSourceExternal, in.ExternalID, in.SourceURL, in.Stale,
		strings.TrimSpace(in.Question), strings.TrimSpace(in.Answer), due, now))
}

// PatchCard changes the user's card by id. ErrNotFound when not the user's;
// ErrInvalidCard for bad text or stale on a card that is not external.
func (s *SQLiteStore) PatchCard(ctx context.Context, userID, cardID string, p CardPatch, now time.Time) (ReviewCard, error) {
	return s.patchCard(ctx, userID, `id = $2`, cardID, p, now)
}

// PatchExternalCard changes the user's external card by its external id.
func (s *SQLiteStore) PatchExternalCard(ctx context.Context, userID, externalID string, p CardPatch, now time.Time) (ReviewCard, error) {
	return s.patchCard(ctx, userID, `external_id = $2`, externalID, p, now)
}

func (s *SQLiteStore) patchCard(ctx context.Context, userID, where, key string, p CardPatch, now time.Time) (ReviewCard, error) {
	var out ReviewCard
	err := s.tx(ctx, func(c conn) error {
		cur, err := scanCard(c.row(ctx, `SELECT `+cardColumns+` FROM review_cards WHERE user_id = $1 AND deleted_at IS NULL AND `+where, userID, key))
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		question, answer, stale := cur.Question, cur.Answer, cur.Stale
		if p.Question != nil {
			question = *p.Question
		}
		if p.Answer != nil {
			answer = *p.Answer
		}
		if p.Stale != nil {
			if cur.SourceKind != CardSourceExternal {
				return ErrInvalidCard
			}
			stale = *p.Stale
		}
		if !ValidCardText(question, answer) {
			return ErrInvalidCard
		}
		out, err = scanCard(c.row(ctx, `UPDATE review_cards SET question = $2, answer = $3, stale = $4, updated_at = $5
			WHERE id = $1 RETURNING `+cardColumns, cur.ID, strings.TrimSpace(question), strings.TrimSpace(answer), stale, now))
		return err
	})
	return out, err
}
