package quality

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	seedfiles "github.com/forgen-team/punisher/curriculum"
)

var update = flag.Bool("update", false, "rewrite testdata/golden.json from the shipped samples")

// TestGoldenSamples scores the shipped sample lectures (no LLM call) and
// compares with testdata/golden.json, so a rule or threshold change shows up
// as a reviewed diff. `go test ./internal/quality -run Golden -update` rewrites it.
func TestGoldenSamples(t *testing.T) {
	hosts, err := CurriculumHosts(seedfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	scores, err := ScoreFS(seedfiles.Samples, "samples", Context{AllowedHosts: hosts})
	if err != nil {
		t.Fatal(err)
	}
	if len(scores) < 6 {
		t.Fatalf("want the 6 shipped samples, got %d", len(scores))
	}
	got, err := json.MarshalIndent(scores, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join("testdata", "golden.json")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, append(got, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if string(want) != string(got)+"\n" {
		t.Fatalf("sample scores changed; review and rerun with -update:\n%s", got)
	}
}

func TestCompare(t *testing.T) {
	before := []Score{{Rules: map[string]RuleResult{"G07": {ID: "G07", Status: Fail, Value: 29}, "G16": {ID: "G16", Status: NA}}}}
	after := []Score{{Rules: map[string]RuleResult{"G07": {ID: "G07", Status: Pass, Value: 21}, "G16": {ID: "G16", Status: Pass, Value: 0}}}}
	d := Compare(before, after)
	if len(d) != 2 || d[0].ID != "G07" || d[0].BeforeFail != 1 || d[0].AfterFail != 0 || d[0].BeforeMean != 29 || d[0].AfterMean != 21 {
		t.Fatalf("delta: %+v", d)
	}
	if d[1].BeforeMean != 0 || d[1].AfterFail != 0 {
		t.Fatalf("na rules are left out: %+v", d[1])
	}
}
