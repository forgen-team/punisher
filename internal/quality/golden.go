package quality

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/lecture"
)

// Offline harness (research #14): score stored lectures (shipped samples or
// lectures dumped with `punisher samples dump`) with the deterministic rubric
// and compare two score sets — e.g. before and after a prompt version — rule
// by rule. No LLM call; the same input always gives the same scores.

// Score is the rubric of one stored lecture.
type Score struct {
	File          string                `json:"file"`
	ConceptID     string                `json:"conceptId"`
	Language      string                `json:"language,omitempty"`
	PromptVersion string                `json:"promptVersion,omitempty"`
	Failed        []string              `json:"failed"`
	Rules         map[string]RuleResult `json:"rules"`
}

// storedLecture reads a sample file ({"content": …}) or bare lecture content.
type storedLecture struct {
	Content       json.RawMessage `json:"content"`
	Language      string          `json:"language"`
	PromptVersion string          `json:"promptVersion"`
}

// ScoreLecture scores one stored lecture file's bytes.
func ScoreLecture(name string, raw []byte, ctx Context) (Score, error) {
	var st storedLecture
	if err := json.Unmarshal(raw, &st); err != nil {
		return Score{}, fmt.Errorf("%s: %w", name, err)
	}
	body := st.Content
	if len(body) == 0 {
		body = raw
	}
	var c lecture.Content
	if err := json.Unmarshal(body, &c); err != nil || len(c.Sections) == 0 {
		return Score{}, fmt.Errorf("%s: not a lecture (content.sections missing)", name)
	}
	rep := Check(c, ctx)
	s := Score{File: name, ConceptID: c.ConceptID, Language: st.Language, PromptVersion: st.PromptVersion,
		Failed: []string{}, Rules: map[string]RuleResult{}}
	for _, r := range rep.Rules {
		s.Rules[r.ID] = r
		if r.Status == Fail {
			s.Failed = append(s.Failed, r.ID)
		}
	}
	return s, nil
}

// ScoreFS scores every *.json under root of fsys (README and other files skipped), sorted by path.
func ScoreFS(fsys fs.FS, root string, ctx Context) ([]Score, error) {
	var out []Score
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || path.Ext(p) != ".json" {
			return err
		}
		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		s, err := ScoreLecture(p, raw, ctx)
		if err != nil {
			return err
		}
		out = append(out, s)
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, err
}

// CurriculumHosts are the hosts of every doc_links URL of the curriculum (allowed source hosts).
func CurriculumHosts(fsys fs.FS) (map[string]bool, error) {
	seed, err := curriculum.Load(fsys)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, c := range seed.Concepts {
		for _, d := range c.DocLinks {
			if u, err := url.Parse(d.URL); err == nil && u.Hostname() != "" {
				out[strings.ToLower(u.Hostname())] = true
			}
		}
	}
	return out, nil
}

// RuleDelta compares one rule across two score sets.
type RuleDelta struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	BeforeFail int     `json:"beforeFail"`
	AfterFail  int     `json:"afterFail"`
	BeforeMean float64 `json:"beforeMean"`
	AfterMean  float64 `json:"afterMean"`
}

// Compare counts failing lectures and the mean measured value per rule in
// before and after (rules not applicable to a lecture are left out of its mean).
func Compare(before, after []Score) []RuleDelta {
	ids := map[string]bool{}
	for _, set := range [][]Score{before, after} {
		for _, s := range set {
			for id := range s.Rules {
				ids[id] = true
			}
		}
	}
	sorted := make([]string, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}
	sort.Strings(sorted)
	stats := func(set []Score, id string) (fails int, mean float64) {
		n := 0
		for _, s := range set {
			r, ok := s.Rules[id]
			if !ok || r.Status == NA {
				continue
			}
			if r.Status == Fail {
				fails++
			}
			mean += r.Value
			n++
		}
		if n > 0 {
			mean /= float64(n)
		}
		return fails, mean
	}
	out := make([]RuleDelta, 0, len(sorted))
	for _, id := range sorted {
		d := RuleDelta{ID: id, Title: RuleTitle(id)}
		d.BeforeFail, d.BeforeMean = stats(before, id)
		d.AfterFail, d.AfterMean = stats(after, id)
		out = append(out, d)
	}
	return out
}
