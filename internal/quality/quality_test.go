package quality

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/forgen-team/punisher/internal/lecture"
)

func section(kind string, blocks ...lecture.Block) lecture.Section {
	for i := range blocks {
		blocks[i].Anchor = kind + "-" + string(rune('1'+i))
	}
	return lecture.Section{Kind: kind, Anchor: kind, Title: kind, Minutes: 5, Blocks: blocks}
}

func code(lang, text string) lecture.Block {
	return lecture.Block{Type: lecture.BlockCode, Lang: lang, Text: text}
}
func logb(text string) lecture.Block {
	return lecture.Block{Type: lecture.BlockLog, Lang: "text", Text: text}
}
func md(text string) lecture.Block { return lecture.Block{Type: lecture.BlockMarkdown, Text: text} }

func TestCheckSQL(t *testing.T) {
	schema := "CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT UNIQUE);\nINSERT INTO users (email) VALUES ('a@x.com'), ('b@x.com');"
	cases := []struct {
		name   string
		blocks []lecture.Block
		want   string
		reason string
	}{
		{"rows match", []lecture.Block{code("sql", schema+"\nSELECT id, email FROM users ORDER BY id;"), logb(" id | email\n  1 | a@x.com\n  2 | b@x.com")}, SQLMatch, ""},
		{"wrong value", []lecture.Block{code("sql", schema+"\nSELECT COUNT(*) FROM users;"), logb("count\n3")}, SQLMismatch, "값이 출력 예시에 없음"},
		{"after pause", []lecture.Block{code("sql", schema+"\nSELECT COUNT(*) FROM users;"), {Type: lecture.BlockPause, Text: "몇 행?"}, logb("2")}, SQLMatch, ""},
		{"error shown", []lecture.Block{code("sql", schema+"\nINSERT INTO users (email) VALUES ('a@x.com');"), logb("ERROR: duplicate key value violates unique constraint")}, SQLMatch, ""},
		{"error not shown", []lecture.Block{code("sql", schema+"\nINSERT INTO users (email) VALUES ('a@x.com');\nSELECT COUNT(*) FROM users;"), logb("3")}, SQLMismatch, "오류"},
		{"sqlite lenient", []lecture.Block{code("sql", "CREATE TABLE t (n INTEGER);\nINSERT INTO t VALUES ('abc');"), logb("ERROR: invalid input syntax for type integer")}, SQLSkipped, "SQLite 에서는 성공"},
		{"postgres dialect", []lecture.Block{code("sql", "SELECT 2147483647::int + 1;"), logb("ERROR: integer out of range")}, SQLSkipped, "방언"},
		{"attach refused", []lecture.Block{code("sql", "ATTACH DATABASE '/tmp/x.db' AS x;"), logb("ok")}, SQLSkipped, "실행하지 않는"},
		{"pragma refused", []lecture.Block{code("sql", "PRAGMA journal_mode = WAL;")}, SQLSkipped, "허용 목록 밖"},
		{"missing schema", []lecture.Block{code("sql", "SELECT * FROM orders;"), logb("1")}, SQLSkipped, "스키마"},
		{"no output", []lecture.Block{code("sql", schema)}, SQLRan, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckSQL(lecture.Content{Sections: []lecture.Section{section("walkthrough", tc.blocks...)}})
			if len(got) != 1 || got[0].Status != tc.want || !strings.Contains(got[0].Reason, tc.reason) {
				t.Fatalf("want %s (%q), got %+v", tc.want, tc.reason, got)
			}
		})
	}
}

func TestCheckSQLRunsBlocksInOrder(t *testing.T) {
	c := lecture.Content{Sections: []lecture.Section{
		section("principles", code("sql", "CREATE TABLE t (n INTEGER); INSERT INTO t VALUES (7);")),
		section("walkthrough", code("sql", "SELECT n * 6 FROM t;"), logb("42")),
	}}
	got := CheckSQL(c)
	if len(got) != 2 || got[0].Status != SQLRan || got[1].Status != SQLMatch {
		t.Fatalf("schema from an earlier block: %+v", got)
	}
}

func TestSplitSQL(t *testing.T) {
	got := SplitSQL("SELECT 'a;b'; -- c;d\nSELECT \"x;\" /* ; */ ;\n\n;")
	if len(got) != 2 || got[0] != "SELECT 'a;b'" || !strings.HasPrefix(got[1], `SELECT "x;"`) {
		t.Fatalf("split: %q", got)
	}
}

func TestCheckArithmetic(t *testing.T) {
	cases := []struct {
		text string
		bad  int
	}{
		{"3 + 4 = 7 이고 2 × 8 = 16 입니다.", 0},
		{"3 + 4 = 8 입니다.", 1},
		{"2^31 − 1 = 2,147,483,647 이 상한입니다.", 0},
		{"0.1 + 0.2 = 0.30000000000000004", 0},
		{"1000 / 3 = 333.33", 0},
		{"0111 + 0001 = 1000 (2진수)", 0},
		{"1111 + 0001 = 0000 (4비트에서 넘침)", 0},
		{"2147483647 + 1 = -2147483648 (32비트 넘침)", 0},
		{"0.1 + 0.2 === 0.3 은 false", 0},
		{"8~10줄, 2026-11-25 = 오늘", 0},
		{"(2 + 3) * 4 = 20", 0},
		{"6 x 7 = 41", 1},
		{"`1 + 1` = 3", 0}, // inline code is not prose
	}
	for _, tc := range cases {
		c := lecture.Content{Sections: []lecture.Section{section("principles", md(tc.text))}}
		if got, _ := CheckArithmetic(c); len(got) != tc.bad {
			t.Errorf("%q: want %d mismatches, got %+v", tc.text, tc.bad, got)
		}
	}
	trace := lecture.Block{Type: lecture.BlockTrace, Data: []byte(`{"caption":"c","columns":["a","b"],"rows":[["1","5 + 5 = 11"],["2","x"]]}`)}
	got, checked := CheckArithmetic(lecture.Content{Sections: []lecture.Section{section("principles", trace)}})
	if checked != 1 || len(got) != 1 || got[0].Computed != "10" {
		t.Fatalf("trace cell: %+v (%d checked)", got, checked)
	}
}

func TestRulesOnSmallLecture(t *testing.T) {
	ex := func(kind string) lecture.Block {
		return lecture.Block{Type: lecture.BlockExercise, Kind: kind, Text: "문제"}
	}
	ans := lecture.Block{Type: lecture.BlockAnswer, Text: "풀이"}
	recall := lecture.Block{Type: lecture.BlockRecall, Text: "질문?\n떠올릴 것: 답", Objective: "c#1"}
	c := lecture.Content{ConceptID: "c", ObjectiveIDs: []string{"c#1"}, Sections: []lecture.Section{
		section(lecture.SectionSituation, md("장면 409 와 1041 이 보입니다."), logb("1\n2\n3\n4\n5\n6\n7\n8")),
		section(lecture.SectionPrinciples, lecture.Block{Type: lecture.BlockKeypoint, Kind: lecture.KeypointDefinition, Text: "정의"},
			ex(lecture.ExerciseGuided), ans, md("[조인](term:db.joins) 그리고 [조인](term:db.joins)"), recall),
		section(lecture.SectionWalkthrough, code("ts", "x"), lecture.Block{Type: lecture.BlockPause, Text: "?"}, logb("y"),
			ex(lecture.ExerciseModify), ans, recall, lecture.Block{Type: lecture.BlockSources}),
		section(lecture.SectionJudgment, md("409 와 1041 을 다시 계산하면 당연히 맞습니다.")),
	}}
	rep := Check(c, Context{})
	want := map[string]string{"G09": Pass, "G11": Fail, "G12": Fail, "G14": Pass, "G16": Pass, "G17": Pass, "G20": Pass, "G23": Pass}
	for id, status := range want {
		if got := rep.Rule(id).Status; got != status {
			t.Errorf("%s: want %s, got %s (%s)", id, status, got, rep.Rule(id).Detail)
		}
	}
	for _, i := range rep.Issues() {
		if !i.Advisory || !strings.HasPrefix(i.Path, "quality.G") {
			t.Fatalf("rubric issues are advisory quality.* issues: %+v", i)
		}
	}
	c.Sections[2].Blocks[3].Kind = lecture.ExerciseWhy
	if got := Check(c, Context{}).Rule("G17").Status; got != Fail {
		t.Fatalf("walkthrough without a modify exercise: %s", got)
	}
}

func TestHostAllowed(t *testing.T) {
	if !HostAllowed("www.postgresql.org", nil) || !HostAllowed("docs.python.org", nil) || HostAllowed("medium.com", nil) {
		t.Fatal("official hosts")
	}
	if !HostAllowed("brendangregg.com", map[string]bool{"www.brendangregg.com": true}) {
		t.Fatal("curriculum hosts")
	}
}

func TestLinkCheckerOnlyAllowedHosts(t *testing.T) {
	var mu sync.Mutex
	var hits []string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits = append(hits, r.Method+" "+r.URL.Path)
		mu.Unlock()
		if r.URL.Path == "/gone" {
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	host := mustHost(t, srv.URL)
	c := lecture.Content{Sections: []lecture.Section{section("principles", lecture.Block{Type: lecture.BlockSources, Sources: []lecture.Source{
		{URL: srv.URL + "/ok"}, {URL: srv.URL + "/gone"}, {URL: "https://medium.com/post"}, {URL: "http://" + host + "/plain"},
	}})}}
	l := NewLinkChecker()
	l.Client = srv.Client()
	got := l.Check(context.Background(), c, map[string]bool{host: true})
	if len(got) != 2 || len(BrokenLinkIssues(got)) != 1 || len(hits) != 2 {
		t.Fatalf("results %+v, hits %v", got, hits)
	}
	l.Check(context.Background(), c, map[string]bool{host: true})
	if len(hits) != 2 {
		t.Fatalf("cached: %v", hits)
	}
	var none *LinkChecker
	if none.Check(context.Background(), c, nil) != nil {
		t.Fatal("nil checker checks nothing")
	}
}

func mustHost(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u.Hostname()
}
