package curriculum

import (
	"strings"
	"testing"

	seed "github.com/forgen-team/punisher/curriculum"
)

func depthConcept(id, area string, depth int, prereqs ...string) Concept {
	c := concept(id, prereqs...)
	c.Area, c.Depth = area, depth
	c.Objectives = []Objective{{Text: "목표", Depth: depth}}
	return c
}

// depth 0 은 0단계 기초 영역에만 허용된다(개념·목표 모두).
func TestValidateDepthZeroOnlyForFoundations(t *testing.T) {
	if err := Validate([]Concept{depthConcept("found.a", AreaFoundations, 0)}); err != nil {
		t.Fatalf("foundations depth 0 must be valid: %v", err)
	}
	err := Validate([]Concept{depthConcept("os.a", "os", 0)})
	if err == nil || !strings.Contains(err.Error(), "depth must be 1..3, got 0") {
		t.Fatalf("depth 0 outside foundations must fail, got %v", err)
	}
	bad := depthConcept("os.b", "os", 1)
	bad.Objectives = []Objective{{Text: "목표", Depth: 0}}
	// normalizeConcept 가 0 을 개념 깊이로 채우므로 Validate 를 직접 불러 목표 규칙을 본다.
	err = Validate([]Concept{bad})
	if err == nil || !strings.Contains(err.Error(), "objective 1 depth must be 1..3, got 0") {
		t.Fatalf("objective depth 0 outside foundations must fail, got %v", err)
	}
	if err := Validate([]Concept{depthConcept("found.b", AreaFoundations, -1)}); err == nil {
		t.Fatal("negative depth must fail")
	}
}

// 다른 영역 개념이 0단계 개념을 선행으로 삼으면 실무자 진도를 막으므로 거부한다.
func TestValidateRejectsFoundationsPrerequisiteOutsideTrack(t *testing.T) {
	err := Validate([]Concept{
		depthConcept("found.a", AreaFoundations, 0),
		depthConcept("net.b", "network-web", 1, "found.a"),
	})
	if err == nil || !strings.Contains(err.Error(), "is a foundations concept") {
		t.Fatalf("expected foundations prerequisite error, got %v", err)
	}
	if err := Validate([]Concept{
		depthConcept("found.a", AreaFoundations, 0),
		depthConcept("found.b", AreaFoundations, 0, "found.a"),
	}); err != nil {
		t.Fatalf("prerequisites inside the track are fine: %v", err)
	}
}

// 배포 시드의 0단계 트랙: 13개, 모두 depth 0, FoundationsTrack 이 YAML 순서와 같다.
func TestShippedFoundationsTrack(t *testing.T) {
	file, err := decodeFile(seed.Files, "foundations.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Concepts) != 13 || len(FoundationsTrack) != 13 {
		t.Fatalf("want 13 foundations concepts, yaml %d track %d", len(file.Concepts), len(FoundationsTrack))
	}
	for i, c := range file.Concepts {
		if c.ID != FoundationsTrack[i] {
			t.Errorf("track[%d] = %s, yaml has %s", i, FoundationsTrack[i], c.ID)
		}
		if c.Depth != 0 || len(c.Objectives) != 3 {
			t.Errorf("%s: depth %d objectives %d (want 0 and 3)", c.ID, c.Depth, len(c.Objectives))
		}
	}
	s, err := Load(seed.Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range s.Concepts {
		if c.Area != AreaFoundations {
			continue
		}
		if c.NameEn == "" || c.SummaryEn == "" || len(c.PracticalCasesEn) != len(c.PracticalCases) {
			t.Errorf("%s: English overlay incomplete", c.ID)
		}
		for _, o := range c.Objectives {
			if o.Depth != 0 || o.TextEn == "" {
				t.Errorf("%s: objective depth %d / English %q", c.ID, o.Depth, o.TextEn)
			}
		}
	}
	for _, a := range ProposalAreas {
		if a == AreaFoundations {
			t.Fatal("foundations must not be a proposal area")
		}
	}
	if len(ProposalAreas) != len(Areas)-1 || IsProposalArea(AreaFoundations) || !IsProposalArea("os") {
		t.Fatalf("proposal areas %v", ProposalAreas)
	}
}
