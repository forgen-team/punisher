package handlers

import (
	"strings"
	"testing"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/services"
)

func TestValidateExplainBody(t *testing.T) {
	cases := []struct {
		name    string
		in      explainBody
		wantErr bool
		want    string
	}{
		{"empty", explainBody{Selection: "   \n "}, true, ""},
		{"one rune", explainBody{Selection: " 가 "}, true, ""},
		{"ok trims and collapses blank lines", explainBody{Anchor: " principles-2 ", Selection: "  첫 줄  \n\n\n\n둘째 줄\t\n"}, false, "첫 줄\n\n둘째 줄"},
		{"max length ok", explainBody{Selection: strings.Repeat("가", 1000)}, false, strings.Repeat("가", 1000)},
		{"too long", explainBody{Selection: strings.Repeat("가", 1001)}, true, ""},
		{"anchor too long", explainBody{Anchor: strings.Repeat("a", 201), Selection: "페이지 캐시"}, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateExplainBody(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && got.Selection != tc.want {
				t.Fatalf("selection = %q, want %q", got.Selection, tc.want)
			}
		})
	}
}

func TestBuildGraphRequestsAndAreaSummary(t *testing.T) {
	data := &models.GraphData{
		Concepts: []models.Concept{
			{ID: "os.a", Name: "A", Area: "os", Depth: 1, Status: "active", UserState: learning.Completed},
			{ID: "os.b", Name: "B", Area: "os", Depth: 1, Status: "active", UserState: learning.InProgress},
			{ID: "os.c", Name: "C", Area: "os", Depth: 2, Status: "active"},
			{ID: "os.new", Name: "New", Area: "os", Depth: 2, Status: "proposed"},
		},
		Edges:  []models.Edge{{From: "os.a", To: "os.new", Kind: "prerequisite"}},
		States: map[string]string{"os.a": learning.Completed, "os.b": learning.InProgress},
		Requests: []models.MapRequest{
			{ID: "r1", Query: "C 공부", ConceptIDs: []string{"os.c"}},
			{ID: "r2", Query: "A 다시", ConceptIDs: []string{"os.a"}}, // done concept: no badge
			{ID: "r3", Query: "새 개념", ConceptIDs: []string{"os.new"}},
			{ID: "r4", Query: "분류 중"},
		},
		Dismissed: map[string]bool{"os.c": true},
	}
	g := buildGraph(data, "", true)
	byID := map[string]models.GraphNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	if len(byID["os.c"].Requests) != 1 || len(byID["os.a"].Requests) != 0 || len(byID["os.new"].Requests) != 1 {
		t.Fatalf("requests not attached as expected: %+v", g.Nodes)
	}
	if len(g.PendingRequests) != 1 || g.PendingRequests[0].ID != "r4" {
		t.Fatalf("pending = %+v", g.PendingRequests)
	}
	if len(g.Edges) != 1 {
		t.Fatalf("proposed node edge missing: %+v", g.Edges)
	}
	if len(g.Coverage) != 1 {
		t.Fatalf("coverage = %+v", g.Coverage)
	}
	cov := g.Coverage[0]
	if cov.Total != 3 || cov.Completed != 1 || cov.InProgress != 1 || cov.Requested != 2 || cov.Dismissed != 1 {
		t.Fatalf("coverage = %+v", cov)
	}
}

func TestValidateParentLink(t *testing.T) {
	ok := services.InstantRequest{ParentLectureID: " d2ab944b-0cf8-4973-9104-edd533950aae ", ParentAnchor: " principles-2 "}
	if msg := validateParentLink(&ok); msg != "" || ok.ParentLectureID != "d2ab944b-0cf8-4973-9104-edd533950aae" || ok.ParentAnchor != "principles-2" {
		t.Fatalf("valid link rejected/not trimmed: %q %+v", msg, ok)
	}
	none := services.InstantRequest{ParentAnchor: "x"}
	if msg := validateParentLink(&none); msg != "" || none.ParentAnchor != "" {
		t.Fatalf("anchor without parent should be dropped: %q %+v", msg, none)
	}
	for _, bad := range []services.InstantRequest{
		{ParentLectureID: "not-a-uuid"},
		{ParentLectureID: "d2ab944b-0cf8-4973-9104-edd533950aae", ParentAnchor: strings.Repeat("a", 201)},
	} {
		if msg := validateParentLink(&bad); msg == "" {
			t.Fatalf("expected rejection for %+v", bad)
		}
	}
}

func TestPromptBodyRules(t *testing.T) {
	b, _ := prompts.LookupBlock(prompts.BlockQASystem)
	if !isDefaultBody(b, prompts.SystemQA+"\n\n  ") || isDefaultBody(b, prompts.SystemQA+" 추가") {
		t.Fatal("isDefaultBody must ignore trailing whitespace only")
	}
	if _, err := validatePromptBody("  \n"); err == nil {
		t.Fatal("empty body must be rejected")
	}
	if _, err := validatePromptBody(strings.Repeat("가", maxPromptRunes+1)); err == nil {
		t.Fatal("oversized body must be rejected")
	}
	if got, err := validatePromptBody("본문 \n\n"); err != nil || got != "본문" {
		t.Fatalf("got %q, %v", got, err)
	}
}
