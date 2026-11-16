package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newDB(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, dbName)
	db, err := sql.Open("sqlite", "file:"+p+"?_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{
		`CREATE TABLE schema_migrations (version TEXT PRIMARY KEY)`,
		`INSERT INTO schema_migrations VALUES ('001_init.sql'), ('016_learning_v3.sql')`,
		`CREATE TABLE notes (body TEXT)`, `INSERT INTO notes VALUES ('hello')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

var known = []string{"001_init.sql", "016_learning_v3.sql"}

func TestExportImportRoundTrip(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	var buf bytes.Buffer
	m, err := Export(context.Background(), newDB(t, src), "v0.8.0", &buf)
	if err != nil || m.Schema != "016_learning_v3.sql" || m.AppVersion != "v0.8.0" {
		t.Fatalf("export = %+v %v", m, err)
	}
	zipPath := filepath.Join(root, "backup.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(root, "dst")
	newDB(t, dst) // existing data to be replaced
	if err := os.WriteFile(filepath.Join(dst, "phone.json"), []byte(`{"host":"h"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bak, _, err := Import(zipPath, dst, known, time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if bak != dst+".bak-20261116-120000" {
		t.Fatalf("backup dir = %s", bak)
	}
	if _, err := os.Stat(filepath.Join(bak, dbName)); err != nil {
		t.Fatalf("old data must be kept: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "phone.json")); string(b) != `{"host":"h"}` {
		t.Fatalf("other files must be carried over, got %q", b)
	}
	db, err := sql.Open("sqlite", filepath.Join(dst, dbName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var body string
	if err := db.QueryRow(`SELECT body FROM notes`).Scan(&body); err != nil || body != "hello" {
		t.Fatalf("imported data = %q %v", body, err)
	}
}

func zipOf(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "x.zip")
	if err := os.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestImportRefuses(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "data")
	cases := map[string]struct {
		files map[string]string
		want  error
	}{
		"zip slip":     {map[string]string{"../evil": "x", dbName: "x", manifestName: "{}"}, ErrBadArchive},
		"extra entry":  {map[string]string{"a/punisher.db": "x", dbName: "x", manifestName: "{}"}, ErrBadArchive},
		"no manifest":  {map[string]string{dbName: "x"}, ErrBadArchive},
		"newer schema": {map[string]string{dbName: "x", manifestName: `{"schema":"099_future.sql"}`}, ErrNewerSchema},
		"bad manifest": {map[string]string{dbName: "x", manifestName: `nope`}, ErrBadArchive},
	}
	for name, c := range cases {
		if _, _, err := Import(zipOf(t, c.files), dst, known, time.Now()); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", name, err, c.want)
		}
	}
	if _, err := os.Stat(dst); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a refused import must not touch the data folder")
	}
}

func TestLock(t *testing.T) {
	dir := t.TempDir()
	a, err := Lock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Lock(dir); !errors.Is(err, ErrLocked) {
		t.Fatalf("second lock = %v, want ErrLocked", err)
	}
	Unlock(a)
	b, err := Lock(dir)
	if err != nil {
		t.Fatalf("lock after unlock: %v", err)
	}
	Unlock(b)
}
