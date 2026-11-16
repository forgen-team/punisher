// Package backup exports the data folder to one zip and imports it back
// (design v3 §9 O6). The zip holds exactly two entries: punisher.db (a
// consistent copy made with VACUUM INTO, safe while the app runs in WAL mode)
// and manifest.json.
package backup

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite" // database/sql driver "sqlite"
)

const (
	dbName       = "punisher.db"
	manifestName = "manifest.json"
	// MaxDBBytes bounds an imported database (zip bombs).
	MaxDBBytes = 4 << 30
)

// Manifest describes an export.
type Manifest struct {
	AppVersion string    `json:"appVersion"`
	Schema     string    `json:"schema"` // last applied migration, e.g. 016_learning_v3.sql
	ExportedAt time.Time `json:"exportedAt"`
}

var (
	// ErrNewerSchema: the export comes from a newer Punisher than this one.
	ErrNewerSchema = errors.New("더 새 버전의 Punisher에서 내보낸 파일이에요. Punisher를 먼저 업데이트해 주세요")
	// ErrBadArchive: not a Punisher export (unexpected entries or paths).
	ErrBadArchive = errors.New("Punisher 내보내기 파일이 아니에요")
)

// Export writes a zip of the database at dbPath to w.
func Export(ctx context.Context, dbPath, appVersion string, w io.Writer) (Manifest, error) {
	tmp, err := os.MkdirTemp(filepath.Dir(dbPath), ".export-")
	if err != nil {
		return Manifest{}, err
	}
	defer os.RemoveAll(tmp)
	copyPath := filepath.Join(tmp, dbName)
	db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(10000)")
	if err != nil {
		return Manifest{}, err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `VACUUM INTO ?`, copyPath); err != nil {
		return Manifest{}, fmt.Errorf("copy database: %w", err)
	}
	m := Manifest{AppVersion: appVersion, ExportedAt: time.Now().UTC()}
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), '') FROM schema_migrations`).Scan(&m.Schema); err != nil {
		return Manifest{}, fmt.Errorf("read schema version: %w", err)
	}
	zw := zip.NewWriter(w)
	entry := func(name string) (io.Writer, error) {
		return zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: m.ExportedAt})
	}
	mw, err := entry(manifestName)
	if err != nil {
		return m, err
	}
	if err := json.NewEncoder(mw).Encode(m); err != nil {
		return m, err
	}
	dw, err := entry(dbName)
	if err != nil {
		return m, err
	}
	f, err := os.Open(copyPath)
	if err != nil {
		return m, err
	}
	defer f.Close()
	if _, err := io.Copy(dw, f); err != nil {
		return m, err
	}
	return m, zw.Close()
}

// ReadManifest validates the archive's entries and returns its manifest:
// exactly manifest.json and punisher.db, no directories or other paths.
func ReadManifest(zr *zip.Reader) (Manifest, *zip.File, error) {
	var m Manifest
	var dbFile, manFile *zip.File
	for _, f := range zr.File {
		switch f.Name {
		case dbName:
			dbFile = f
		case manifestName:
			manFile = f
		default:
			return m, nil, fmt.Errorf("%w: unexpected entry %q", ErrBadArchive, f.Name)
		}
	}
	if dbFile == nil || manFile == nil {
		return m, nil, fmt.Errorf("%w: missing %s or %s", ErrBadArchive, dbName, manifestName)
	}
	if dbFile.UncompressedSize64 > MaxDBBytes {
		return m, nil, fmt.Errorf("%w: database too large", ErrBadArchive)
	}
	rc, err := manFile.Open()
	if err != nil {
		return m, nil, err
	}
	defer rc.Close()
	if err := json.NewDecoder(io.LimitReader(rc, 1<<16)).Decode(&m); err != nil {
		return m, nil, fmt.Errorf("%w: manifest: %v", ErrBadArchive, err)
	}
	return m, dbFile, nil
}

// CheckSchema refuses an export whose last migration this build does not know.
func CheckSchema(m Manifest, known []string) error {
	if m.Schema == "" {
		return nil
	}
	sort.Strings(known)
	i := sort.SearchStrings(known, m.Schema)
	if i < len(known) && known[i] == m.Schema {
		return nil
	}
	return ErrNewerSchema
}

// Import replaces dataDir with the archive: it is unpacked into a sibling
// temporary folder, the current folder's other files (phone settings, …) are
// copied over, then the current folder becomes <dataDir>.bak-<time> and the new
// one takes its place. The caller holds the data folder lock (the app is not running).
func Import(zipPath, dataDir string, known []string, now time.Time) (backupDir string, m Manifest, err error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", m, fmt.Errorf("%w: %v", ErrBadArchive, err)
	}
	defer zr.Close()
	m, dbFile, err := ReadManifest(&zr.Reader)
	if err != nil {
		return "", m, err
	}
	if err := CheckSchema(m, known); err != nil {
		return "", m, err
	}
	parent := filepath.Dir(filepath.Clean(dataDir))
	staging, err := os.MkdirTemp(parent, "."+filepath.Base(dataDir)+".import-")
	if err != nil {
		return "", m, err
	}
	done := false
	defer func() {
		if !done {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := extract(dbFile, filepath.Join(staging, dbName)); err != nil {
		return "", m, err
	}
	if err := copyExtras(dataDir, staging); err != nil {
		return "", m, err
	}
	stamp := now.Format("20060102-150405")
	if _, err := os.Stat(dataDir); err == nil {
		backupDir = dataDir + ".bak-" + stamp
		if err := os.Rename(dataDir, backupDir); err != nil {
			return "", m, fmt.Errorf("keep current data: %w", err)
		}
	}
	if err := os.Rename(staging, dataDir); err != nil {
		if backupDir != "" {
			_ = os.Rename(backupDir, dataDir)
		}
		return "", m, fmt.Errorf("put imported data in place: %w", err)
	}
	done = true
	return backupDir, m, nil
}

func extract(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(rc, MaxDBBytes+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > MaxDBBytes {
		err = fmt.Errorf("%w: database too large", ErrBadArchive)
	}
	return err
}

// copyExtras copies the data folder's files other than the database, its WAL
// files, the lock and tmp/ (e.g. phone settings) into the new folder.
func copyExtras(from, to string) error {
	entries, err := os.ReadDir(from)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, dbName) || name == LockName {
			continue
		}
		b, err := os.ReadFile(filepath.Join(from, name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(to, name), b, 0o600); err != nil {
			return err
		}
	}
	return nil
}
