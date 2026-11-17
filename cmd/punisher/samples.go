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
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/middleware"
	"github.com/forgen-team/punisher/internal/store"
)

// seedSamples adds the shipped sample lectures where they fit — only those in
// the user's language (i18n L9): curriculum/samples/*.json are Korean,
// curriculum/samples/<lang>/*.json the other languages. A failure only logs
// (the app works without them).
func seedSamples(ctx context.Context, st *store.SQLiteStore) {
	locale := st.LocaleFor(ctx, middleware.LocalUserID)
	dir := "samples"
	if locale != learning.LocaleKO {
		dir = "samples/" + locale
	}
	files, err := fs.Glob(seedfiles.Samples, dir+"/*.json")
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
		if sm.Language == "" {
			sm.Language = locale
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
// default) to curriculum/samples/<concept>.json in the current directory
// (an English lecture to curriculum/samples/en/<concept>.json).
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
		dir := filepath.Join("curriculum", "samples")
		if sm.Language != learning.LocaleKO {
			dir = filepath.Join(dir, sm.Language)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		out := filepath.Join(dir, sm.ConceptID+".json")
		if err := os.WriteFile(out, append(b, '\n'), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("%s → %s (%d check items)\n", sm.Title, out, len(sm.CheckItems))
	}
	return 0
}
