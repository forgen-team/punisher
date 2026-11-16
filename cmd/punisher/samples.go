package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/config"
	"github.com/forgen-team/punisher/internal/store"
)

// seedSamples adds the shipped sample lectures (curriculum/samples/*.json)
// where they fit; a failure only logs (the app works without them).
func seedSamples(ctx context.Context, st *store.SQLiteStore) {
	files, err := fs.Glob(seedfiles.Samples, "samples/*.json")
	if err != nil {
		log.Printf("samples: %v", err)
		return
	}
	for _, f := range files {
		b, err := seedfiles.Samples.ReadFile(f)
		if err != nil {
			log.Printf("samples %s: %v", f, err)
			continue
		}
		var sm store.Sample
		if err := json.Unmarshal(b, &sm); err != nil {
			log.Printf("samples %s: %v", f, err)
			continue
		}
		added, err := st.SeedSample(ctx, sm)
		if err != nil {
			log.Printf("samples %s: %v", f, err)
		} else if added {
			log.Printf("sample lecture added: %s", sm.Title)
		}
	}
}

// runSamples handles the development command `punisher samples dump <강의 id>...`:
// it writes each published lecture of the data folder (PUNISHER_DATA_DIR or the
// default) to curriculum/samples/<concept>.json in the current directory.
func runSamples(args []string) int {
	if len(args) < 2 || args[0] != "dump" {
		fmt.Fprintln(os.Stderr, "usage: punisher samples dump <lecture id>...")
		return 2
	}
	dataDir := config.Load().DataDir
	if dir := os.Getenv("PUNISHER_DATA_DIR"); dir != "" {
		dataDir = dir
	}
	st, err := store.Open(filepath.Join(dataDir, "punisher.db"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()
	if err := os.MkdirAll(filepath.Join("curriculum", "samples"), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for _, id := range args[1:] {
		sm, err := st.DumpSample(context.Background(), id)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		b, err := json.MarshalIndent(sm, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		out := filepath.Join("curriculum", "samples", sm.ConceptID+".json")
		if err := os.WriteFile(out, append(b, '\n'), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("%s → %s (%d check items)\n", sm.Title, out, len(sm.CheckItems))
	}
	return 0
}
