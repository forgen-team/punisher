package chapter

import (
	"strings"
	"testing"
)

func ctx() Context {
	return Context{
		UserID:      "user-a",
		Visible:     map[string]string{"os.process": "프로세스", "net.tcp": "TCP"},
		OwnProposed: map[string]string{"infra.k8s-cni": "쿠버네티스 CNI"},
		Areas:       []string{"os", "network-web", "infra-ops"},
	}
}

func newConcept(key, name string, prereqs ...string) Concept {
	return Concept{Key: key, Name: name, Area: "infra-ops", Depth: 2, Summary: "s",
		Objectives: []Objective{{Text: name + "를 설명한다", Depth: 2}}, Prerequisites: prereqs}
}

func valid() Draft {
	return Draft{Title: "쿠버네티스 네트워킹", Concepts: []Concept{
		newConcept("service-routing", "서비스 라우팅", "pod-network"),
		newConcept("pod-network", "파드 네트워크", "net.tcp"),
		{ProposedID: "infra.k8s-cni", Area: "infra-ops", Depth: 2, Objectives: []Objective{{Text: "CNI 역할을 설명한다", Depth: 2}}, Prerequisites: []string{"pod-network"}},
		{ExistingID: "os.process"},
		newConcept("ingress", "인그레스", "service-routing"),
		newConcept("dup", "tcp"), // same name as a visible concept (case-insensitive) → reused
	}}
}

func TestValidateAssignsIDsActionsAndOrder(t *testing.T) {
	d, problems := Validate(valid(), ctx())
	if len(problems) > 0 {
		t.Fatalf("problems: %v", problems)
	}
	prefix := OwnerPrefix("user-a")
	pos := map[string]int{}
	actions := map[string]string{}
	for i, c := range d.Concepts {
		pos[c.ID] = i
		actions[c.ID] = c.Action
	}
	if actions["net.tcp"] != ActionReuse || len(d.Concepts) != 6 {
		t.Fatalf("name duplicate must be reused: %v", actions)
	}
	if actions[prefix+".pod-network"] != ActionCreate || actions["infra.k8s-cni"] != ActionPromote || actions["os.process"] != ActionReuse {
		t.Fatalf("actions = %v", actions)
	}
	if pos[prefix+".pod-network"] > pos[prefix+".service-routing"] || pos[prefix+".service-routing"] > pos[prefix+".ingress"] {
		t.Fatalf("not in prerequisite order: %v", pos)
	}
	for _, c := range d.Concepts {
		if c.ID == prefix+".service-routing" && (len(c.Prerequisites) != 1 || c.Prerequisites[0] != prefix+".pod-network") {
			t.Fatalf("prerequisites not resolved to ids: %v", c.Prerequisites)
		}
	}
	if !strings.HasPrefix(prefix, "u") || len(prefix) != 9 || OwnerPrefix("user-b") == prefix {
		t.Fatalf("prefix = %s", prefix)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*Draft){
		"too few":           func(d *Draft) { d.Concepts = d.Concepts[:3] },
		"bad key":           func(d *Draft) { d.Concepts[0].Key = "Bad Key" },
		"foreign existing":  func(d *Draft) { d.Concepts[3].ExistingID = "u12345678.someone-else" },
		"foreign proposed":  func(d *Draft) { d.Concepts[2].ProposedID = "infra.not-mine" },
		"unknown prereq":    func(d *Draft) { d.Concepts[0].Prerequisites = []string{"nowhere"} },
		"bad area":          func(d *Draft) { d.Concepts[0].Area = "llm" },
		"no objectives":     func(d *Draft) { d.Concepts[0].Objectives = nil },
		"cycle":             func(d *Draft) { d.Concepts[1].Prerequisites = []string{"ingress"} },
		"duplicate concept": func(d *Draft) { d.Concepts[4].Key = "pod-network" },
	}
	for name, mutate := range cases {
		d := valid()
		mutate(&d)
		if _, problems := Validate(d, ctx()); len(problems) == 0 {
			t.Errorf("%s: expected a problem", name)
		}
	}
}

func TestValidateIsIdempotent(t *testing.T) {
	d, problems := Validate(valid(), ctx())
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	again, problems := Validate(d, ctx())
	if len(problems) > 0 {
		t.Fatalf("re-validating a normalized draft must pass: %v", problems)
	}
	if len(again.Concepts) != len(d.Concepts) || again.Concepts[0].ID != d.Concepts[0].ID {
		t.Fatalf("re-validation changed the draft: %+v", again.Concepts)
	}
}

func TestValidateStripsTermLinks(t *testing.T) {
	d := valid()
	d.Summary = "[HTTP](term:net.http) 위에서"
	d.Concepts[0].Summary = "[TCP](term:net.tcp) 기반"
	got, problems := Validate(d, ctx())
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if got.Summary != "HTTP 위에서" {
		t.Fatalf("summary = %q", got.Summary)
	}
	for _, c := range got.Concepts {
		if strings.Contains(c.Summary, "term:") {
			t.Fatalf("concept summary keeps link markup: %q", c.Summary)
		}
	}
}
