// Command punisher-witems-import moves the Witems Punisher data (PostgreSQL)
// into the app's SQLite database once, before the first start in
// PUNISHER_MODE=witems. It refuses to touch an existing database.
//
//	punisher-witems-import -owner <witems user id> [-data-dir /data]   (PostgreSQL URL in WITEMS_PUNISHER_PG_URL)
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/forgen-team/punisher/internal/witemsimport"
	"github.com/forgen-team/punisher/migrations"
)

func main() {
	owner := flag.String("owner", os.Getenv("PUNISHER_OWNER_USER_ID"), "Witems user id of the owner (becomes the local user)")
	dataDir := flag.String("data-dir", envOr("PUNISHER_DATA_DIR", "/data"), "folder for punisher.db (must not contain one)")
	locale := flag.String("locale", envOr("PUNISHER_LOCALE", "ko"), "default language for settings that were never saved")
	flag.Parse()
	pgURL := os.Getenv("WITEMS_PUNISHER_PG_URL")
	if pgURL == "" || *owner == "" {
		fmt.Fprintln(os.Stderr, "set WITEMS_PUNISHER_PG_URL and -owner (or PUNISHER_OWNER_USER_ID)")
		os.Exit(2)
	}
	if err := run(pgURL, *owner, *dataDir, *locale); err != nil {
		fmt.Fprintln(os.Stderr, "import failed:", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func run(pgURL, owner, dataDir, locale string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	path := filepath.Join(dataDir, "punisher.db")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists — the import only fills a new database", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	pg, err := sql.Open("pgx", pgURL)
	if err != nil {
		return err
	}
	defer pg.Close()
	if err := pg.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}

	// The same start-up steps as the app: migrations, then the curriculum seed.
	st, err := store.Open(path)
	if err != nil {
		return err
	}
	defer st.Close()
	store.SetDefaultLocale(locale)
	if err := st.Migrate(ctx, migrations.Files); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		return err
	}
	if _, err := st.SyncCurriculum(ctx, seed); err != nil {
		return fmt.Errorf("curriculum seed: %w", err)
	}

	report, err := witemsimport.Run(ctx, pg, st.DB(), owner)
	for _, r := range report {
		switch {
		case r.NoSource:
			fmt.Printf("%-32s (no table in Witems)\n", r.Table)
		default:
			fmt.Printf("%-32s read %5d  wrote %5d  kept %4d  other users %d", r.Table, r.Source, r.Inserted, r.Kept, r.OtherUsers)
			if len(r.Missing) > 0 {
				fmt.Printf("  dropped columns %v", r.Missing)
			}
			fmt.Println()
		}
	}
	if err != nil {
		// leave no half-filled database behind (the transaction was rolled back)
		_ = st.Close()
		for _, suffix := range []string{"", "-wal", "-shm"} {
			_ = os.Remove(path + suffix)
		}
		return err
	}
	fmt.Println("done:", path)
	return nil
}
