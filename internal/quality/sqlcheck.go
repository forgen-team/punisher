package quality

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/lecture"

	_ "modernc.org/sqlite" // in-process SQLite (pure Go): the examples run in a fresh :memory: database
)

// SQL example statuses.
const (
	SQLMatch    = "match"    // ran; the shown output agrees
	SQLMismatch = "mismatch" // ran; the shown output disagrees (reviewer evidence)
	SQLRan      = "ran"      // ran; no output block to compare
	SQLSkipped  = "skipped"  // not run or not comparable (Reason)
)

// SQLResult is one lang=sql code block run in memory.
type SQLResult struct {
	Anchor string `json:"anchor"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	// Output summarizes what the run gave (rows or the error).
	Output string `json:"output,omitempty"`
}

// sqlTimeout bounds the SQL run of one lecture; sqlMaxRows the rows read per statement.
const (
	sqlTimeout = 2 * time.Second
	sqlMaxRows = 200
	// sqlMaxPages bounds the memory database (4 KiB pages → 8 MiB).
	sqlMaxPages = 2000
)

var (
	// sqlAllowedRe: the statement kinds an example may use (no ATTACH, VACUUM, triggers, virtual tables, other PRAGMAs).
	sqlAllowedRe = regexp.MustCompile(`(?is)^(?:SELECT|WITH|VALUES|INSERT|REPLACE|UPDATE|DELETE|BEGIN|COMMIT|END|ROLLBACK|SAVEPOINT|RELEASE|` +
		`CREATE\s+(?:UNIQUE\s+)?INDEX|CREATE\s+(?:TEMP\s+|TEMPORARY\s+)?(?:TABLE|VIEW)|DROP\s+(?:TABLE|INDEX|VIEW)|ALTER\s+TABLE|` +
		`PRAGMA\s+foreign_keys\b|EXPLAIN\s+QUERY\s+PLAN)\b`)
	// sqlDeniedRe: functions/words never run, wherever they appear.
	sqlDeniedRe = regexp.MustCompile(`(?i)\b(?:load_extension|randomblob|zeroblob|readfile|writefile|edit|fts[345]|ATTACH|DETACH|VACUUM|TRIGGER|VIRTUAL)\b`)
	// sqlDialectRe: another database's syntax (PostgreSQL, MySQL, psql meta commands).
	sqlDialectRe    = regexp.MustCompile("(?im)::|\\b(?:SERIAL|BIGSERIAL|ILIKE|IDENTITY|AUTO_INCREMENT|JSONB|TIMESTAMPTZ|CONCURRENTLY|INTERVAL|nextval|information_schema|pg_[a-z_]+)\\b|\\bNOW\\(\\)|\\$\\d|^\\s*SHOW\\b|ENGINE\\s*=|`|^\\s*\\\\[a-z]|^\\s*\\.[a-z]")
	sqlErrorShownRe = regexp.MustCompile(`(?i)error|constraint|violat|duplicate|failed|에러|오류|실패|거부`)
	logNumberRe     = regexp.MustCompile(`-?\d+(?:\.\d+)?`)
)

// CheckSQL runs the lecture's lang=sql code blocks, in order, in one fresh
// in-memory SQLite database and compares each with the output block shown
// after it (directly, or after a @@pause). Blocks in another dialect, with
// statements outside the allow-list, or depending on schema the lecture never
// creates are skipped, never reported as wrong.
func CheckSQL(c lecture.Content) []SQLResult {
	var blocks []sqlBlock
	for _, s := range c.Sections {
		for i, b := range s.Blocks {
			if b.Type == lecture.BlockCode && isSQLLang(b.Lang) {
				out, ok := outputAfter(s.Blocks, i)
				blocks = append(blocks, sqlBlock{anchor: b.Anchor, code: b.Text, output: out, hasOut: ok})
			}
		}
	}
	if len(blocks) == 0 {
		return nil
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return []SQLResult{{Anchor: blocks[0].anchor, Status: SQLSkipped, Reason: "메모리 DB를 열 수 없음"}}
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), sqlTimeout)
	defer cancel()
	if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA max_page_count = %d", sqlMaxPages)); err != nil {
		return []SQLResult{{Anchor: blocks[0].anchor, Status: SQLSkipped, Reason: "메모리 DB 설정 실패"}}
	}
	out := make([]SQLResult, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, runSQLBlock(ctx, db, b))
	}
	return out
}

type sqlBlock struct {
	anchor, code string
	// output: the text of the log/code block shown as this code's output ("" = none).
	output string
	hasOut bool
}

func isSQLLang(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "sql", "sqlite", "sqlite3":
		return true
	}
	return false
}

// outputAfter is the output block of code block i: the next @@log (or a text
// code block), directly or after one @@pause.
func outputAfter(blocks []lecture.Block, i int) (string, bool) {
	j := i + 1
	if j < len(blocks) && blocks[j].Type == lecture.BlockPause {
		j++
	}
	if j >= len(blocks) {
		return "", false
	}
	switch b := blocks[j]; {
	case b.Type == lecture.BlockLog:
		return b.Text, true
	case b.Type == lecture.BlockCode && strings.EqualFold(b.Lang, "text"):
		return b.Text, true
	}
	return "", false
}

type stmtRun struct {
	err     error
	columns []string
	rows    [][]any
}

func runSQLBlock(ctx context.Context, db *sql.DB, b sqlBlock) SQLResult {
	res := SQLResult{Anchor: b.anchor}
	stmts := SplitSQL(b.code)
	switch {
	case len(stmts) == 0:
		res.Status, res.Reason = SQLSkipped, "문장 없음"
		return res
	case sqlDialectRe.MatchString(b.code):
		res.Status, res.Reason = SQLSkipped, "다른 DB 방언(PostgreSQL·MySQL·psql)"
		return res
	case sqlDeniedRe.MatchString(b.code):
		res.Status, res.Reason = SQLSkipped, "실행하지 않는 문장"
		return res
	}
	for _, st := range stmts {
		if !sqlAllowedRe.MatchString(st) {
			res.Status, res.Reason = SQLSkipped, "허용 목록 밖 문장: "+clip(st, 40)
			return res
		}
	}
	var runs []stmtRun
	for _, st := range stmts {
		r := runStatement(ctx, db, st)
		if r.err != nil {
			msg := r.err.Error()
			switch {
			case ctx.Err() != nil:
				res.Status, res.Reason = SQLSkipped, "시간 제한"
				return res
			case strings.Contains(msg, "syntax error") || strings.Contains(msg, "no such function") || strings.Contains(msg, "near \""):
				res.Status, res.Reason = SQLSkipped, "SQLite 에서 해석되지 않음: "+clip(msg, 80)
				return res
			case strings.Contains(msg, "no such table") || strings.Contains(msg, "no such column"):
				res.Status, res.Reason = SQLSkipped, "강의에 없는 스키마가 필요함: "+clip(msg, 80)
				return res
			}
		}
		runs = append(runs, r)
	}
	res.Output = summarizeRuns(runs)
	if !b.hasOut {
		res.Status = SQLRan
		return res
	}
	res.Status, res.Reason = compareSQLOutput(runs, b.output)
	return res
}

func runStatement(ctx context.Context, db *sql.DB, st string) stmtRun {
	rows, err := db.QueryContext(ctx, st)
	if err != nil {
		return stmtRun{err: err}
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		return stmtRun{err: err}
	}
	r := stmtRun{columns: cols}
	for rows.Next() && len(r.rows) < sqlMaxRows {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return stmtRun{err: err}
		}
		r.rows = append(r.rows, vals)
	}
	r.err = rows.Err()
	return r
}

// lastResult is the last statement that returned columns, and the first runtime error.
func lastResult(runs []stmtRun) (rows *stmtRun, firstErr error) {
	for i := range runs {
		if runs[i].err != nil && firstErr == nil {
			firstErr = runs[i].err
		}
		if runs[i].err == nil && len(runs[i].columns) > 0 {
			rows = &runs[i]
		}
	}
	return rows, firstErr
}

func summarizeRuns(runs []stmtRun) string {
	rows, firstErr := lastResult(runs)
	var parts []string
	if firstErr != nil {
		parts = append(parts, "오류: "+clip(firstErr.Error(), 120))
	}
	if rows != nil {
		var shown []string
		for i, row := range rows.rows {
			if i == 5 {
				shown = append(shown, "…")
				break
			}
			cells := make([]string, len(row))
			for j, v := range row {
				cells[j] = cellText(v)
			}
			shown = append(shown, "("+strings.Join(cells, ", ")+")")
		}
		parts = append(parts, fmt.Sprintf("%d행 %s", len(rows.rows), strings.Join(shown, " ")))
	}
	if len(parts) == 0 {
		return "실행됨(결과 행 없음)"
	}
	return strings.Join(parts, "; ")
}

// compareSQLOutput compares a run with the shown output. Only a disagreement
// the run proves is a mismatch: SQLite accepting what another database rejects
// (type affinity, foreign keys off by default) is skipped, not reported.
func compareSQLOutput(runs []stmtRun, shown string) (string, string) {
	rows, firstErr := lastResult(runs)
	shownErr := sqlErrorShownRe.MatchString(shown)
	switch {
	case firstErr != nil && shownErr:
		return SQLMatch, "실행 오류와 출력 예시의 오류가 같이 있음"
	case firstErr != nil:
		return SQLMismatch, "실행하면 오류(" + clip(firstErr.Error(), 80) + ")인데 출력 예시에는 오류가 없음"
	case shownErr:
		return SQLSkipped, "출력 예시는 오류지만 SQLite 에서는 성공(제품별 차이일 수 있음)"
	case rows == nil || len(rows.rows) == 0:
		return SQLSkipped, "결과 행이 없어 비교하지 않음"
	}
	numbers := map[float64]bool{}
	for _, m := range logNumberRe.FindAllString(strings.ReplaceAll(shown, ",", ""), -1) {
		if f, err := strconv.ParseFloat(m, 64); err == nil {
			numbers[f] = true
		}
	}
	var missing []string
	for i, row := range rows.rows {
		if i == 20 {
			break
		}
		for _, v := range row {
			if !valueShown(v, shown, numbers) {
				missing = append(missing, cellText(v))
			}
		}
	}
	if len(missing) == 0 {
		return SQLMatch, ""
	}
	if strings.Contains(shown, "…") || strings.Contains(shown, "...") {
		return SQLSkipped, "출력 예시가 줄여져 있어 비교하지 않음"
	}
	if len(missing) > 5 {
		missing = append(missing[:5], "…")
	}
	return SQLMismatch, "실행 결과의 값이 출력 예시에 없음: " + strings.Join(missing, ", ")
}

func valueShown(v any, shown string, numbers map[float64]bool) bool {
	switch x := v.(type) {
	case nil:
		return true
	case int64:
		return numbers[float64(x)]
	case float64:
		for n := range numbers {
			if math.Abs(n-x) <= 1e-9*math.Max(1, math.Abs(x)) {
				return true
			}
		}
		return false
	case []byte:
		return strings.Contains(shown, string(x))
	case string:
		return strings.Contains(shown, x)
	case bool:
		return true
	}
	return strings.Contains(shown, fmt.Sprint(v))
}

func cellText(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(x)
	}
	return fmt.Sprint(v)
}

// SplitSQL splits a script into statements at ';' outside quotes and comments
// (comments removed, empty statements dropped).
func SplitSQL(script string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	r := []rune(script)
	for i := 0; i < len(r); i++ {
		ch := r[i]
		switch {
		case ch == '-' && i+1 < len(r) && r[i+1] == '-':
			for i < len(r) && r[i] != '\n' {
				i++
			}
			cur.WriteRune('\n')
		case ch == '/' && i+1 < len(r) && r[i+1] == '*':
			i += 2
			for i+1 < len(r) && !(r[i] == '*' && r[i+1] == '/') {
				i++
			}
			i++
			cur.WriteRune(' ')
		case ch == '\'' || ch == '"':
			quote := ch
			cur.WriteRune(ch)
			for i++; i < len(r); i++ {
				cur.WriteRune(r[i])
				if r[i] == quote {
					if i+1 < len(r) && r[i+1] == quote {
						i++
						cur.WriteRune(r[i])
						continue
					}
					break
				}
			}
		case ch == ';':
			flush()
		default:
			cur.WriteRune(ch)
		}
	}
	flush()
	return out
}

func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// ruleSQL (G21): no SQL example whose shown output the in-memory run contradicts.
func ruleSQL(results []SQLResult) RuleResult {
	if len(results) == 0 {
		return na("G21", "SQL 코드 없음")
	}
	var bad []string
	counts := map[string]int{}
	for _, r := range results {
		counts[r.Status]++
		if r.Status == SQLMismatch {
			bad = append(bad, r.Anchor+": "+r.Reason)
		}
	}
	detail := fmt.Sprintf("SQL %d개(일치 %d, 불일치 %d, 실행만 %d, 건너뜀 %d)", len(results), counts[SQLMatch], counts[SQLMismatch], counts[SQLRan], counts[SQLSkipped])
	if len(bad) > 0 {
		detail += " — " + strings.Join(bad, "; ")
	}
	return result("G21", len(bad) == 0, float64(len(bad)), "불일치 0", detail)
}
