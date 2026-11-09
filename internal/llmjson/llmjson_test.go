package llmjson

import (
	"errors"
	"testing"
)

type sample struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

func TestTryParseJSON(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", `{"title":"a","tags":["x"]}`, "a"},
		{"json fence", "```json\n{\"title\":\"b\"}\n```", "b"},
		{"bare fence", "```\n{\"title\":\"c\"}\n```", "c"},
		{"prose then fence", "Here is the JSON:\n```json\n{\"title\":\"d\"}\n```\nThanks", "d"},
		{"prose around object", `Sure! {"title":"e","tags":[]} hope it helps`, "e"},
		{"bom and spaces", "\xef\xbb\xbf  {\"title\":\"f\"}  ", "f"},
		{"bare object with code block in value", "{\"title\":\"g\",\"tags\":[\"```kotlin\\nval x = 1\\n```\"]}", "g"},
		{"bare object with fenced code in value", "{\"title\":\"h\",\"tags\":[\"설명\\n```kotlin\\nfun f() = 1\\n```\\n끝\"]}", "h"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := TryParseJSON[sample](tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Title != tc.want {
				t.Fatalf("title = %q, want %q", got.Title, tc.want)
			}
		})
	}
}

func TestTryParseJSONArray(t *testing.T) {
	got, err := TryParseJSON[[]sample]("결과입니다:\n[{\"title\":\"a\"},{\"title\":\"b\"}]\n끝")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[1].Title != "b" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestTryParseJSONErrors(t *testing.T) {
	if _, err := TryParseJSON[sample]("   "); !errors.Is(err, ErrEmpty) {
		t.Fatalf("expected ErrEmpty, got %v", err)
	}
	if _, err := TryParseJSON[sample]("no json here"); err == nil {
		t.Fatal("expected error for non-JSON text")
	}
	if _, err := TryParseJSON[sample](`{"title": "broken"`); err == nil {
		t.Fatal("expected error for truncated JSON")
	}
}
