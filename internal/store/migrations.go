package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
)

// Migrate applies the embedded *.sql files in name order. Each file runs in
// its own transaction together with its schema_migrations record, so a
// failed file leaves no partial state.
func (s *SQLiteStore) Migrate(ctx context.Context, files fs.FS) error {
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		if err := s.applyOne(ctx, files, e.Name()); err != nil {
			return err
		}
	}
	return nil
}

// fkOffMarker on the first line of a migration runs it with foreign keys
// disabled (SQLite's documented 12-step table rebuild: a CHECK constraint can
// only be changed by recreating the table, and dropping the old table with
// foreign keys on would cascade-delete referencing rows). After commit the
// whole database is checked with PRAGMA foreign_key_check.
const fkOffMarker = "-- punisher:foreign_keys=off"

func (s *SQLiteStore) applyOne(ctx context.Context, files fs.FS, version string) error {
	var applied string
	err := s.db.QueryRowContext(ctx, `SELECT version FROM schema_migrations WHERE version = ?`, version).Scan(&applied)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check migration %s: %w", version, err)
	}
	content, err := fs.ReadFile(files, version)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", version, err)
	}
	fkOff := strings.HasPrefix(string(content), fkOffMarker)

	// Pin one connection: PRAGMA foreign_keys is per connection and a no-op inside a transaction.
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migration %s: %w", version, err)
	}
	defer conn.Close()
	if fkOff {
		if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
			return fmt.Errorf("migration %s: foreign keys off: %w", version, err)
		}
		defer func() { _, _ = conn.ExecContext(context.Background(), `PRAGMA foreign_keys = ON`) }()
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", version, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("apply migration %s: %w", version, err)
	}
	if fkOff {
		rows, err := tx.QueryContext(ctx, `PRAGMA foreign_key_check`)
		if err != nil {
			return fmt.Errorf("migration %s: foreign key check: %w", version, err)
		}
		bad := rows.Next()
		_ = rows.Close()
		if bad {
			return fmt.Errorf("migration %s: foreign key check failed; rolled back", version)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", version, err)
	}
	log.Printf("applied migration: %s", version)
	return nil
}
