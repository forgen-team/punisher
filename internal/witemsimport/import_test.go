package witemsimport

import (
	"strings"
	"testing"
)

func TestSelectExprTurnsPostgresTypesIntoSQLiteStorage(t *testing.T) {
	cases := map[string]string{
		"timestamp with time zone": `to_char("c" AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')`,
		"boolean":                  `CASE WHEN "c" THEN 1 WHEN NOT "c" THEN 0 END`,
		"ARRAY":                    `to_json("c")::text`,
		"uuid":                     `"c"::text`,
		"jsonb":                    `"c"::text`,
		"numeric":                  `"c"::float8`,
		"text":                     `"c"`,
		"smallint":                 `"c"`,
	}
	for pgType, want := range cases {
		if got := selectExpr(column{name: "c", pgType: pgType}); got != want {
			t.Errorf("%s: got %s, want %s", pgType, got, want)
		}
	}
}

func TestUserColumns(t *testing.T) {
	for _, name := range []string{"user_id", "owner_user_id"} {
		if !isUserColumn(name) {
			t.Errorf("%s should be mapped to the local user", name)
		}
	}
	for _, name := range []string{"users", "user_ids", "user_id_hint"} {
		if isUserColumn(name) {
			t.Errorf("%s is not a user id column", name)
		}
	}
}

func TestQuoteIdent(t *testing.T) {
	if got := quoteIdent(`a"b`); got != `"a""b"` {
		t.Fatalf("got %s", got)
	}
	if !strings.HasPrefix(selectExpr(column{name: `x"y`, pgType: "text"}), `"x""y"`) {
		t.Fatal("identifiers must be quoted")
	}
}
