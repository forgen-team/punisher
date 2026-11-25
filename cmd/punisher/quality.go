package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/quality"
)

const qualityUsage = `punisher quality score [파일|폴더 ...]   강의 JSON(샘플 형식)을 결정적 루브릭으로 채점 (기본: 내장 샘플)
punisher quality compare <전.json> <후.json>   두 채점 결과를 규칙별로 비교`

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
