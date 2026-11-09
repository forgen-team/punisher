// Package store is the SQLite persistence layer (pure-Go modernc.org/sqlite,
// no CGO), split per domain (sqlite_curriculum.go, sqlite_concepts.go, ...).
//
// Conventions (ported from the PostgreSQL store):
//   - timestamps are TEXT in UTC "2006-01-02T15:04:05.000Z" so they sort and
//     compare lexicographically; NOW() in queries is rewritten to strftime;
//   - $N placeholders are rewritten to ?N (numbered, order-independent);
//   - TEXT[] columns are JSON arrays; `= ANY(x)` becomes json_each;
//   - UUID primary keys are generated in Go (NewID).
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

var (
	// ErrNotFound is returned when the requested row does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConceptRetired is returned when writing state for a retired concept.
	ErrConceptRetired = errors.New("concept is retired")
)

// SQLiteStore implements Store on one SQLite file.
type SQLiteStore struct {
	db *sql.DB
}

var _ Store = (*SQLiteStore)(nil)

// Open opens (creating if needed) the database file at path with WAL,
// foreign keys and a busy timeout. A single connection serialises access:
// the app has one user and short queries, and it rules out SQLITE_BUSY on
// read→write transaction upgrades.
func Open(path string) (*SQLiteStore, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}
	}
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxIdleTime(0)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// DB exposes the handle for tests and migrations.
func (s *SQLiteStore) DB() *sql.DB { return s.db }

// ---------------------------------------------------------------------------
// query helpers
// ---------------------------------------------------------------------------

// TimeLayout is the stored timestamp format (matches nowExpr).
const TimeLayout = "2006-01-02T15:04:05.000Z"

const nowExpr = `strftime('%Y-%m-%dT%H:%M:%fZ','now')`

var placeholderRE = regexp.MustCompile(`\$(\d+)`)

var rewriteCache sync.Map

// rewrite converts the PostgreSQL-flavoured query text used in this package
// to SQLite: NOW() → strftime, $N → ?N.
func rewrite(q string) string {
	if v, ok := rewriteCache.Load(q); ok {
		return v.(string)
	}
	out := strings.ReplaceAll(q, "NOW()", nowExpr)
	out = placeholderRE.ReplaceAllString(out, "?$1")
	rewriteCache.Store(q, out)
	return out
}

// FormatTime renders t in the stored format.
func FormatTime(t time.Time) string { return t.UTC().Format(TimeLayout) }

// arg converts Go values to what the schema stores.
func arg(v any) any {
	switch x := v.(type) {
	case time.Time:
		return FormatTime(x)
	case *time.Time:
		if x == nil {
			return nil
		}
		return FormatTime(*x)
	case []string:
		if x == nil {
			x = []string{}
		}
		b, _ := json.Marshal(x)
		return string(b)
	case json.RawMessage:
		if x == nil {
			return nil
		}
		return string(x)
	default:
		return v
	}
}

func args(in []any) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = arg(v)
	}
	return out
}

type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// conn applies rewrite/arg conversion on a *sql.DB or *sql.Tx.
type conn struct{ q querier }

func (c conn) exec(ctx context.Context, q string, a ...any) (sql.Result, error) {
	return c.q.ExecContext(ctx, rewrite(q), args(a)...)
}

// rowsResult carries a query result into collect.
type rowsResult struct {
	rows *sql.Rows
	err  error
}

func (c conn) query(ctx context.Context, q string, a ...any) rowsResult {
	rows, err := c.q.QueryContext(ctx, rewrite(q), args(a)...)
	return rowsResult{rows, err}
}

func (c conn) row(ctx context.Context, q string, a ...any) *sql.Row {
	return c.q.QueryRowContext(ctx, rewrite(q), args(a)...)
}

func (s *SQLiteStore) c() conn { return conn{s.db} }

// tx runs fn in one transaction (rolled back on error).
func (s *SQLiteStore) tx(ctx context.Context, fn func(conn) error) error {
	t, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	if err := fn(conn{t}); err != nil {
		_ = t.Rollback()
		return err
	}
	return t.Commit()
}

type rowScanner interface {
	Scan(dest ...any) error
}

// collect reads every row with scan and closes rows. It returns an empty
// (non-nil) slice when there are no rows so JSON encodes [].
func collect[T any](res rowsResult, scan func(rowScanner) (T, error)) ([]T, error) {
	if res.err != nil {
		return nil, res.err
	}
	rows := res.rows
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanString(r rowScanner) (string, error) {
	var s string
	err := r.Scan(&s)
	return s, err
}

func affected(res sql.Result) int64 {
	n, err := res.RowsAffected()
	if err != nil {
		return 0
	}
	return n
}

// NewID returns a random (v4) UUID string.
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// ---------------------------------------------------------------------------
// column scanners
// ---------------------------------------------------------------------------

func textOf(src any) (string, bool) {
	switch v := src.(type) {
	case string:
		return v, true
	case []byte:
		return string(v), true
	case time.Time:
		return FormatTime(v), true
	default:
		return "", false
	}
}

func parseTime(s string) (time.Time, error) {
	for _, layout := range []string{TimeLayout, time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable timestamp %q", s)
}

type timeScanner struct{ dst *time.Time }

func (t timeScanner) Scan(src any) error {
	if src == nil {
		*t.dst = time.Time{}
		return nil
	}
	s, ok := textOf(src)
	if !ok {
		return fmt.Errorf("timestamp: unexpected %T", src)
	}
	v, err := parseTime(s)
	if err != nil {
		return err
	}
	*t.dst = v
	return nil
}

type nullTimeScanner struct{ dst **time.Time }

func (t nullTimeScanner) Scan(src any) error {
	if src == nil {
		*t.dst = nil
		return nil
	}
	var v time.Time
	if err := (timeScanner{&v}).Scan(src); err != nil {
		return err
	}
	*t.dst = &v
	return nil
}

type stringsScanner struct{ dst *[]string }

func (s stringsScanner) Scan(src any) error {
	out := []string{}
	if src != nil {
		text, ok := textOf(src)
		if !ok {
			return fmt.Errorf("text array: unexpected %T", src)
		}
		if strings.TrimSpace(text) != "" {
			if err := json.Unmarshal([]byte(text), &out); err != nil {
				return fmt.Errorf("text array: %w", err)
			}
		}
	}
	if out == nil {
		out = []string{}
	}
	*s.dst = out
	return nil
}

type rawScanner struct{ dst *json.RawMessage }

func (r rawScanner) Scan(src any) error {
	if src == nil {
		*r.dst = nil
		return nil
	}
	text, ok := textOf(src)
	if !ok {
		return fmt.Errorf("json: unexpected %T", src)
	}
	*r.dst = json.RawMessage(text)
	return nil
}

func ts(t *time.Time) sql.Scanner            { return timeScanner{t} }
func nts(t **time.Time) sql.Scanner          { return nullTimeScanner{t} }
func strs(v *[]string) sql.Scanner           { return stringsScanner{v} }
func rawJSON(v *json.RawMessage) sql.Scanner { return rawScanner{v} }
