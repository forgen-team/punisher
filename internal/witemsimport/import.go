// Package witemsimport copies the Witems Punisher data (PostgreSQL) into a
// standalone SQLite database, once, when Punisher moves into the Witems shell
// as an app (PUNISHER_MODE=witems).
//
// The SQLite side is created the way the app starts (migrations + curriculum
// seed), then every table both sides have is copied column by column, with
// PostgreSQL types turned into this repo's storage (docs/SYNC.md SQL table):
// timestamps → "YYYY-MM-DDTHH:MM:SS.mmmZ" UTC text, booleans → 0/1,
// uuid/jsonb → text, text[] → JSON text. The Witems owner becomes the local
// user; rows of any other user are skipped and counted.
package witemsimport

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// LocalUserID is the app's only user (internal/middleware.LocalUserID).
const LocalUserID = "local"

// curriculum tables are already filled by the seed: keep the seed's rows and
// add only rows the seed does not have (user-made concepts and their links).
var curriculumTables = map[string]bool{"concepts": true, "concept_edges": true, "concept_aliases": true, "objectives": true}

// skipped tables: bookkeeping of each database itself.
var skipTables = map[string]bool{"curriculum_versions": true, "schema_migrations": true}

// TableReport is what happened to one table.
type TableReport struct {
	Table      string
	Source     int      // rows read from PostgreSQL
	Inserted   int      // rows written
	Kept       int      // curriculum rows the seed already had
	OtherUsers int      // rows of another Witems user (skipped)
	Missing    []string // PostgreSQL columns SQLite does not have (dropped)
	NoSource   bool     // the table does not exist in PostgreSQL
}

// Report lists every table in SQLite order.
type Report []TableReport

type column struct{ name, pgType string }

// Run copies pg into out. out must be a migrated, seeded, otherwise empty database.
func Run(ctx context.Context, pg, out *sql.DB, ownerUserID string) (Report, error) {
	if ownerUserID == "" {
		return nil, fmt.Errorf("owner user id is required")
	}
	tables, err := sqliteTables(ctx, out)
	if err != nil {
		return nil, err
	}
	conn, err := out.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	// PRAGMA foreign_keys is per connection and ignored inside a transaction: turn it off first,
	// copy in any order, then check every reference once at the end.
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return nil, err
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), `PRAGMA foreign_keys = ON`) }()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var report Report
	for _, table := range tables {
		if skipTables[table] {
			continue
		}
		r, err := copyTable(ctx, pg, tx, table, ownerUserID)
		if err != nil {
			return report, fmt.Errorf("%s: %w", table, err)
		}
		report = append(report, r)
	}
	if err := foreignKeyCheck(ctx, tx); err != nil {
		return report, err
	}
	if err := tx.Commit(); err != nil {
		return report, err
	}
	return report, nil
}

func sqliteTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func sqliteColumns(ctx context.Context, tx *sql.Tx, table string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT name FROM pragma_table_info(?) ORDER BY cid`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func pgColumns(ctx context.Context, pg *sql.DB, table string) ([]column, error) {
	rows, err := pg.QueryContext(ctx, `SELECT column_name, data_type FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1 ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []column
	for rows.Next() {
		var c column
		if err := rows.Scan(&c.name, &c.pgType); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// selectExpr reads one PostgreSQL column in this repo's SQLite representation.
func selectExpr(c column) string {
	q := quoteIdent(c.name)
	switch c.pgType {
	case "timestamp with time zone":
		return fmt.Sprintf(`to_char(%s AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')`, q)
	case "timestamp without time zone":
		return fmt.Sprintf(`to_char(%s, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')`, q)
	case "date":
		return fmt.Sprintf(`to_char(%s, 'YYYY-MM-DD')`, q)
	case "boolean":
		return fmt.Sprintf(`CASE WHEN %s THEN 1 WHEN NOT %s THEN 0 END`, q, q)
	case "ARRAY":
		return fmt.Sprintf(`to_json(%s)::text`, q)
	case "uuid", "jsonb", "json":
		return q + "::text"
	case "numeric":
		return q + "::float8"
	default:
		return q
	}
}

func isUserColumn(name string) bool {
	return name == "user_id" || strings.HasSuffix(name, "_user_id")
}

func copyTable(ctx context.Context, pg *sql.DB, tx *sql.Tx, table, owner string) (TableReport, error) {
	r := TableReport{Table: table}
	pgCols, err := pgColumns(ctx, pg, table)
	if err != nil {
		return r, err
	}
	if len(pgCols) == 0 {
		r.NoSource = true
		return r, nil
	}
	liteCols, err := sqliteColumns(ctx, tx, table)
	if err != nil {
		return r, err
	}
	have := map[string]bool{}
	for _, c := range liteCols {
		have[c] = true
	}
	var cols []column
	for _, c := range pgCols {
		if have[c.name] {
			cols = append(cols, c)
		} else {
			r.Missing = append(r.Missing, c.name)
		}
	}
	sort.Strings(r.Missing)

	exprs := make([]string, len(cols))
	names := make([]string, len(cols))
	marks := make([]string, len(cols))
	for i, c := range cols {
		exprs[i] = selectExpr(c)
		names[i] = quoteIdent(c.name)
		marks[i] = "?"
	}
	rows, err := pg.QueryContext(ctx, fmt.Sprintf(`SELECT %s FROM %s`, strings.Join(exprs, ", "), quoteIdent(table)))
	if err != nil {
		return r, err
	}
	defer rows.Close()

	verb := "INSERT"
	if curriculumTables[table] {
		verb = "INSERT OR IGNORE"
	}
	insert, err := tx.PrepareContext(ctx, fmt.Sprintf(`%s INTO %s (%s) VALUES (%s)`, verb, quoteIdent(table), strings.Join(names, ", "), strings.Join(marks, ", ")))
	if err != nil {
		return r, err
	}
	defer insert.Close()

	values := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range values {
		ptrs[i] = &values[i]
	}
rowLoop:
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return r, err
		}
		r.Source++
		for i, c := range cols {
			if b, ok := values[i].([]byte); ok {
				values[i] = string(b)
			}
			if !isUserColumn(c.name) || values[i] == nil {
				continue
			}
			if fmt.Sprint(values[i]) != owner {
				r.OtherUsers++
				continue rowLoop
			}
			values[i] = LocalUserID
		}
		res, err := insert.ExecContext(ctx, values...)
		if err != nil {
			return r, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			r.Inserted++
		} else {
			r.Kept++
		}
	}
	return r, rows.Err()
}

func foreignKeyCheck(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var problems []string
	for rows.Next() {
		var table, parent string
		var rowid sql.NullInt64
		var fkid int
		if err := rows.Scan(&table, &rowid, &parent, &fkid); err != nil {
			return err
		}
		problems = append(problems, fmt.Sprintf("%s(rowid %d) → %s", table, rowid.Int64, parent))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(problems) > 0 {
		if len(problems) > 20 {
			problems = append(problems[:20], fmt.Sprintf("… %d more", len(problems)-20))
		}
		return fmt.Errorf("foreign key check failed:\n  %s", strings.Join(problems, "\n  "))
	}
	return nil
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
