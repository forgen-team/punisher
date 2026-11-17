package curriculum_test

import (
	"testing"
	"testing/fstest"

	seed "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
)

// The shipped overlay translates every concept, objective and practical case.
func TestShippedEnglishOverlayIsComplete(t *testing.T) {
	s, err := curriculum.Load(seed.Files)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, c := range s.Concepts {
		if c.NameEn == "" || c.SummaryEn == "" {
			t.Errorf("%s: English name/summary missing", c.ID)
		}
		if len(c.PracticalCasesEn) != len(c.PracticalCases) {
			t.Errorf("%s: English practical cases %d/%d", c.ID, len(c.PracticalCasesEn), len(c.PracticalCases))
		}
		for i, o := range c.Objectives {
			if o.TextEn == "" {
				t.Errorf("%s#%d: English objective missing", c.ID, i+1)
			}
		}
	}
	for _, a := range curriculum.Areas {
		if curriculum.AreaTitleIn(a, "en") == curriculum.AreaTitle(a) {
			t.Errorf("area %s has no English title", a)
		}
	}
}

const koSeed = `area: os
title: 운영체제
concepts:
  - id: os.process
    name: 프로세스
    depth: 1
    summary: 요약
    objectives: [하나, 둘]
    practical_cases: [사례]
  - id: os.thread
    name: 스레드
    depth: 1
    summary: 요약
    objectives: [하나]
`

func TestOverlayAppliesAndFallsBack(t *testing.T) {
	fs := fstest.MapFS{
		"os.yaml": {Data: []byte(koSeed)},
		"i18n/en/os.yaml": {Data: []byte(`area: os
title: Operating Systems
concepts:
  os.process:
    name: Process
    summary: Summary
    objectives: [One, Two]
    practical_cases: [Case]
`)},
	}
	s, err := curriculum.Load(fs)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	by := map[string]curriculum.Concept{}
	for _, c := range s.Concepts {
		by[c.ID] = c
	}
	p, th := by["os.process"], by["os.thread"]
	if p.NameIn("en") != "Process" || p.Objectives[1].TextIn("en") != "Two" || p.NameIn("ko") != "프로세스" {
		t.Fatalf("overlay not applied: %+v", p)
	}
	if th.NameIn("en") != "스레드" || th.Objectives[0].TextIn("en") != "하나" {
		t.Fatalf("missing overlay should fall back to Korean: %+v", th)
	}
	found := false
	for _, a := range s.Aliases {
		if a.Alias == curriculum.NormalizeAlias("Process") && a.ConceptID == "os.process" {
			found = true
		}
	}
	if !found {
		t.Fatal("English name should be an alias")
	}
}

func TestOverlayMustFollowSeed(t *testing.T) {
	cases := map[string]string{
		"objective count": "concepts:\n  os.process:\n    name: P\n    summary: S\n    objectives: [One]\n",
		"unknown concept": "concepts:\n  os.nope:\n    name: P\n    summary: S\n    objectives: [One]\n",
		"alias clash":     "concepts:\n  os.thread:\n    name: 프로세스\n    summary: S\n    objectives: [One]\n",
	}
	for name, body := range cases {
		fs := fstest.MapFS{"os.yaml": {Data: []byte(koSeed)}, "i18n/en/os.yaml": {Data: []byte("area: os\n" + body)}}
		if _, err := curriculum.Load(fs); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}
