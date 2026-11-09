// Package config loads the standalone app's settings. Everything has a
// working default; env vars (PUNISHER_*) exist only for troubleshooting.
package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// DefaultPort is tried first; when it is busy a random free port is used.
const DefaultPort = 4710

type Config struct {
	// Port to listen on (127.0.0.1 only). 0 = DefaultPort, falling back to a free port.
	Port int
	// DataDir holds punisher.db and temp dirs for CLI calls.
	DataDir string
	// CurriculumDir overrides the embedded seed with a directory on disk (dev only).
	CurriculumDir string
	// OpenBrowser opens the UI in the default browser after start.
	OpenBrowser bool

	// Generation via the local claude CLI.
	ClaudeBin         string // "" = auto-detect
	LectureModel      string
	FastModel         string
	Concurrency       int
	DailyLectureLimit int // 0 = off
	DailyTokenLimit   int // 0 = off
	DailyCallLimit    int // 0 = off
	LibraryAhead      int
	StudyPregenerate  bool
	LectureMaxTokens  int
	SchedulerEnabled  bool
	SchedulerInterval time.Duration
}

// Load reads PUNISHER_* env vars over the defaults.
func Load() Config {
	return Config{
		Port:          readInt("PUNISHER_PORT", 0),
		DataDir:       readEnv("PUNISHER_DATA_DIR", DefaultDataDir()),
		CurriculumDir: strings.TrimSpace(os.Getenv("PUNISHER_CURRICULUM_DIR")),
		OpenBrowser:   readEnv("PUNISHER_OPEN_BROWSER", "true") != "false",

		ClaudeBin:         strings.TrimSpace(os.Getenv("PUNISHER_CLAUDE_BIN")),
		LectureModel:      readEnv("PUNISHER_LECTURE_MODEL", "sonnet"),
		FastModel:         readEnv("PUNISHER_FAST_MODEL", "haiku"),
		Concurrency:       readInt("PUNISHER_CONCURRENCY", 2),
		DailyLectureLimit: readInt("PUNISHER_DAILY_LECTURE_LIMIT", 0),
		DailyTokenLimit:   readInt("PUNISHER_DAILY_TOKEN_LIMIT", 0),
		DailyCallLimit:    readInt("PUNISHER_DAILY_CALL_LIMIT", 0),
		LibraryAhead:      readInt("PUNISHER_LIBRARY_AHEAD", 1),
		StudyPregenerate:  readEnv("PUNISHER_STUDY_PREGENERATE", "true") != "false",
		LectureMaxTokens:  readInt("PUNISHER_LECTURE_MAX_TOKENS", 16000),
		SchedulerEnabled:  readEnv("PUNISHER_SCHEDULER_ENABLED", "true") != "false",
		SchedulerInterval: readDuration("PUNISHER_SCHEDULER_INTERVAL", 60*time.Minute),
	}
}

// DefaultDataDir is ~/Library/Application Support/Punisher on macOS and
// ~/.punisher elsewhere.
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".punisher"
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "Punisher")
	}
	return filepath.Join(home, ".punisher")
}

func readInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func readDuration(key string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(os.Getenv(key)))
	if err != nil || d < time.Minute {
		return fallback
	}
	return d
}

func readEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
