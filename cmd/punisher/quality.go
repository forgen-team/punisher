package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/config"
	"github.com/forgen-team/punisher/internal/quality"
	"github.com/forgen-team/punisher/internal/store"
)

const qualityUsage = `punisher quality score [파일|폴더 ...]   강의 JSON(샘플 형식)을 결정적 루브릭으로 채점 (기본: 내장 샘플)
punisher quality compare <전.json> <후.json>   두 채점 결과를 규칙별로 비교
punisher quality db                            데이터 폴더의 감수 품질 기록을 프롬프트 버전별로 요약`

// runQuality is the offline quality harness (research #14): no LLM call.
func runQuality(args []string, stdout io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, qualityUsage)
		return 2
	}
	switch args[0] {
	case "score":
		return qualityScore(args[1:], stdout)
	case "compare":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, qualityUsage)
			return 2
		}
		return qualityCompare(args[1], args[2], stdout)
	case "db":
		return qualityDB(stdout)
	}
	fmt.Fprintln(os.Stderr, qualityUsage)
	return 2
}

func qualityScore(paths []string, stdout io.Writer) int {
	hosts, err := quality.CurriculumHosts(seedfiles.Files)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ctx := quality.Context{AllowedHosts: hosts}
	var scores []quality.Score
	if len(paths) == 0 {
		scores, err = quality.ScoreFS(seedfiles.Samples, "samples", ctx)
	}
	for _, p := range paths {
		var got []quality.Score
		got, err = scorePath(p, ctx)
		if err != nil {
			break
		}
		scores = append(scores, got...)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return writeJSON(stdout, scores)
}

func scorePath(p string, ctx quality.Context) ([]quality.Score, error) {
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return quality.ScoreFS(os.DirFS(p), ".", ctx)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	s, err := quality.ScoreLecture(filepath.Base(p), raw, ctx)
	return []quality.Score{s}, err
}

func qualityCompare(beforePath, afterPath string, stdout io.Writer) int {
	var sets [2][]quality.Score
	for i, p := range []string{beforePath, afterPath} {
		raw, err := os.ReadFile(p)
		if err == nil {
			err = json.Unmarshal(raw, &sets[i])
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, p+":", err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "%-4s %-20s %8s %8s %10s %10s\n", "규칙", "이름", "전 실패", "후 실패", "전 평균", "후 평균")
	for _, d := range quality.Compare(sets[0], sets[1]) {
		fmt.Fprintf(stdout, "%-4s %-20s %8d %8d %10.1f %10.1f\n", d.ID, d.Title, d.BeforeFail, d.AfterFail, d.BeforeMean, d.AfterMean)
	}
	fmt.Fprintf(stdout, "강의: 전 %d편, 후 %d편\n", len(sets[0]), len(sets[1]))
	return 0
}

func writeJSON(w io.Writer, v any) int {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// storedQuality is the part of lecture_reviews.quality the summary reads.
type storedQuality struct {
	Machine quality.Report `json:"machine"`
	LLM     []struct {
		ID    string `json:"id"`
		Score int    `json:"score"`
	} `json:"llm"`
}

// qualityDB summarizes the stored review quality of the data folder's
// lectures per prompt version: rule failures and mean reviewer scores.
func qualityDB(stdout io.Writer) int {
	dataDir := config.Load().DataDir
	st, err := store.Open(filepath.Join(dataDir, "punisher.db"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()
	records, err := st.QualityRecords(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	type group struct {
		n      int
		fails  map[string]int
		scores map[string][]int
	}
	groups := map[string]*group{}
	var order []string
	for _, r := range records {
		var q storedQuality
		if json.Unmarshal(r.Quality, &q) != nil {
			continue
		}
		g := groups[r.PromptVersion]
		if g == nil {
			g = &group{fails: map[string]int{}, scores: map[string][]int{}}
			groups[r.PromptVersion] = g
			order = append(order, r.PromptVersion)
		}
		g.n++
		for _, f := range q.Machine.Failed() {
			g.fails[f.ID]++
		}
		for _, sc := range q.LLM {
			g.scores[sc.ID] = append(g.scores[sc.ID], sc.Score)
		}
	}
	if len(order) == 0 {
		fmt.Fprintln(stdout, "품질 기록이 있는 강의가 없습니다.")
		return 0
	}
	for _, v := range order {
		g := groups[v]
		fmt.Fprintf(stdout, "\n%s — 강의 %d편\n  규칙 실패:", v, g.n)
		for _, id := range sortedKeys(g.fails) {
			fmt.Fprintf(stdout, " %s %d", id, g.fails[id])
		}
		fmt.Fprint(stdout, "\n  감수 점수 평균:")
		for _, id := range sortedKeys(g.scores) {
			sum := 0
			for _, x := range g.scores[id] {
				sum += x
			}
			fmt.Fprintf(stdout, " %s %.1f(%d)", id, float64(sum)/float64(len(g.scores[id])), len(g.scores[id]))
		}
		fmt.Fprintln(stdout)
	}
	return 0
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
