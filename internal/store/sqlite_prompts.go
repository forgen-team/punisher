package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// PromptHistoryLimit is how many history entries are kept per (user, block).
const PromptHistoryLimit = 20

// PromptOverride is a user's saved block body.
type PromptOverride struct {
	PromptID  string    `json:"promptId"`
	Body      string    `json:"body"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// PromptHistoryEntry is one saved/reset/restored version of a block.
type PromptHistoryEntry struct {
	ID        int64     `json:"id"`
	PromptID  string    `json:"promptId"`
	Body      string    `json:"body"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"createdAt"`
}

// PromptOverrides returns the user's overrides by block id.
func (s *SQLiteStore) PromptOverrides(ctx context.Context, userID string) (map[string]PromptOverride, error) {
	list, err := collect(s.c().query(ctx, `SELECT prompt_id, body, updated_at FROM prompt_overrides WHERE user_id = $1`, userID),
		func(r rowScanner) (PromptOverride, error) {
			var o PromptOverride
			err := r.Scan(&o.PromptID, &o.Body, ts(&o.UpdatedAt))
			return o, err
		})
	if err != nil {
		return nil, fmt.Errorf("scan prompt overrides: %w", err)
	}
	out := make(map[string]PromptOverride, len(list))
	for _, o := range list {
		out[o.PromptID] = o
	}
	return out, nil
}

// SavePromptOverride upserts the block body and records it in history.
func (s *SQLiteStore) SavePromptOverride(ctx context.Context, userID, promptID, body, action string) (PromptOverride, error) {
	o := PromptOverride{PromptID: promptID, Body: body}
	err := s.tx(ctx, func(tx conn) error {
		if err := tx.row(ctx, `
		INSERT INTO prompt_overrides (user_id, prompt_id, body, updated_at) VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, prompt_id) DO UPDATE SET body = excluded.body, updated_at = NOW()
		RETURNING updated_at`, userID, promptID, body).Scan(ts(&o.UpdatedAt)); err != nil {
			return fmt.Errorf("save prompt override: %w", err)
		}
		return addPromptHistory(ctx, tx, userID, promptID, body, action)
	})
	return o, err
}

// ResetPromptOverride deletes the override (back to the code default); the
// removed body is kept in history so it can be restored. No override = no-op.
func (s *SQLiteStore) ResetPromptOverride(ctx context.Context, userID, promptID string) error {
	return s.tx(ctx, func(tx conn) error {
		var body string
		err := tx.row(ctx, `DELETE FROM prompt_overrides WHERE user_id = $1 AND prompt_id = $2 RETURNING body`,
			userID, promptID).Scan(&body)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reset prompt override: %w", err)
		}
		return addPromptHistory(ctx, tx, userID, promptID, body, "reset")
	})
}

// PromptHistory returns the newest-first history of one block.
func (s *SQLiteStore) PromptHistory(ctx context.Context, userID, promptID string) ([]PromptHistoryEntry, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT id, prompt_id, body, action, created_at FROM prompt_override_history
		WHERE user_id = $1 AND prompt_id = $2 ORDER BY id DESC LIMIT $3`, userID, promptID, PromptHistoryLimit),
		func(r rowScanner) (PromptHistoryEntry, error) {
			var e PromptHistoryEntry
			err := r.Scan(&e.ID, &e.PromptID, &e.Body, &e.Action, ts(&e.CreatedAt))
			return e, err
		})
	if err != nil {
		return nil, fmt.Errorf("scan prompt history: %w", err)
	}
	if list == nil {
		list = []PromptHistoryEntry{}
	}
	return list, nil
}

// PromptHistoryEntryByID loads one of the user's history entries.
func (s *SQLiteStore) PromptHistoryEntryByID(ctx context.Context, userID string, id int64) (PromptHistoryEntry, error) {
	var e PromptHistoryEntry
	err := s.c().row(ctx, `
		SELECT id, prompt_id, body, action, created_at FROM prompt_override_history WHERE user_id = $1 AND id = $2`,
		userID, id).Scan(&e.ID, &e.PromptID, &e.Body, &e.Action, ts(&e.CreatedAt))
	if errors.Is(err, sql.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

func addPromptHistory(ctx context.Context, tx conn, userID, promptID, body, action string) error {
	if _, err := tx.exec(ctx, `
		INSERT INTO prompt_override_history (user_id, prompt_id, body, action) VALUES ($1, $2, $3, $4)`,
		userID, promptID, body, action); err != nil {
		return fmt.Errorf("prompt history insert: %w", err)
	}
	if _, err := tx.exec(ctx, `
		DELETE FROM prompt_override_history WHERE user_id = $1 AND prompt_id = $2 AND id NOT IN (
			SELECT id FROM prompt_override_history WHERE user_id = $1 AND prompt_id = $2 ORDER BY id DESC LIMIT $3)`,
		userID, promptID, PromptHistoryLimit); err != nil {
		return fmt.Errorf("prompt history trim: %w", err)
	}
	return nil
}

// UserGloss returns the user's gloss cached for a prompt hash ("" when none).
func (s *SQLiteStore) UserGloss(ctx context.Context, userID, conceptID, promptHash string) (string, error) {
	var g string
	err := s.c().row(ctx, `
		SELECT gloss FROM user_term_glosses WHERE user_id = $1 AND concept_id = $2 AND prompt_hash = $3`,
		userID, conceptID, promptHash).Scan(&g)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return g, err
}

// SaveUserGloss caches a gloss generated with the user's gloss prompt.
func (s *SQLiteStore) SaveUserGloss(ctx context.Context, userID, conceptID, promptHash, gloss, model string) error {
	_, err := s.c().exec(ctx, `
		INSERT INTO user_term_glosses (user_id, concept_id, prompt_hash, gloss, model) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, concept_id, prompt_hash) DO UPDATE SET gloss = excluded.gloss, model = excluded.model`,
		userID, conceptID, promptHash, gloss, model)
	return err
}
