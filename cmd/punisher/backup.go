package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/forgen-team/punisher/internal/backup"
	"github.com/forgen-team/punisher/internal/config"
	"github.com/forgen-team/punisher/migrations"
)

const backupUsage = `데이터 내보내기·가져오기:
  punisher export [파일.zip]   지금 데이터를 파일 하나로 (실행 중이어도 돼요)
  punisher import <파일.zip>   그 파일로 데이터를 바꾸기 (앱을 끄고, 지금 데이터는 옆 폴더에 남겨요)`

// runBackup handles `punisher export|import` and returns the exit code.
func runBackup(cmd string, args []string) int {
	cfg := config.Load()
	if dir := os.Getenv("PUNISHER_DATA_DIR"); dir != "" {
		cfg.DataDir = dir
	}
	switch {
	case cmd == "export" && len(args) <= 1:
		out := fmt.Sprintf("punisher-backup-%s.zip", time.Now().Format("20060102"))
		if len(args) == 1 {
			out = args[0]
		}
		return exportTo(cfg.DataDir, out)
	case cmd == "import" && len(args) == 1:
		return importFrom(cfg.DataDir, args[0])
	}
	fmt.Fprintln(os.Stderr, backupUsage)
	return 2
}

func exportTo(dataDir, out string) int {
	f, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "파일을 만들 수 없어요: %v\n", err)
		return 1
	}
	m, err := backup.Export(context.Background(), filepath.Join(dataDir, "punisher.db"), version, f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(out)
		fmt.Fprintf(os.Stderr, "내보내지 못했어요: %v\n", err)
		return 1
	}
	fmt.Printf("내보냈어요: %s (스키마 %s)\n", out, m.Schema)
	return 0
}

func importFrom(dataDir, in string) int {
	lock, err := backup.Lock(dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	known, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		backup.Unlock(lock)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	bak, m, err := backup.Import(in, dataDir, known, time.Now())
	backup.Unlock(lock)
	if err != nil {
		fmt.Fprintf(os.Stderr, "가져오지 못했어요: %v\n", err)
		return 1
	}
	fmt.Printf("가져왔어요 (%s에 내보낸 데이터, 스키마 %s).\n", m.ExportedAt.Local().Format("2006-01-02 15:04"), m.Schema)
	if bak != "" {
		fmt.Printf("이전 데이터는 여기에 남겨 뒀어요: %s\n", bak)
	}
	fmt.Println("punisher 를 실행하면 필요한 DB 변경이 자동으로 적용돼요.")
	return 0
}
