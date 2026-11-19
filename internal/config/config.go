// Package config loads the standalone app's settings. Everything has a
// working default; env vars (PUNISHER_*) exist only for troubleshooting.
package config

import (
	"errors"
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
	// Locale (ko|en) of a user who never saved settings: PUNISHER_LOCALE, else the OS locale (DefaultLocale).
	Locale string

	// Generation via the local claude CLI.
	ClaudeBin         string // "" = auto-detect
	LectureModel      string
	FastModel         string
	Concurrency       int
	DailyLectureLimit int // 0 = off
	DailyTokenLimit   int // 0 = off
	DailyCallLimit    int // 0 = off
	LibraryAhead      int
	NightAhead        int
	NightStartHour    int
	NightEndHour      int
	StudyPregenerate  bool
	LectureMaxTokens  int
	SchedulerEnabled  bool
	SchedulerInterval time.Duration
	// Mastery checks: grading model (default LectureModel), the dispute regrade
	// model (default: the one grading does not use) and their own daily token
	// limit (0 = unlimited; checks are excluded from DailyTokenLimit).
	CheckModel           string
	CheckRegradeModel    string
	DailyCheckTokenLimit int
	// BackfillReviewEnabled: the scheduler reviews stored lectures without a
	// pass review (lecture quality design v2.2 §2.2; also the lectures already
	// in a user's database). Runs only when SchedulerEnabled.
	BackfillReviewEnabled bool

	// Witems is the Witems app mode (PUNISHER_MODE=witems): the app runs as a
	// container inside the Witems shell instead of on this machine. See
	// docs/WITEMS-APP.md.
	Witems Witems
}

// Witems app mode settings. Only read when PUNISHER_MODE=witems.
type Witems struct {
	Enabled bool
	// Addr to listen on inside the container (PUNISHER_LISTEN, default 0.0.0.0:8080).
	Addr string
	// AuthServiceURL is Witems project-setup; the WITEMS_SESSION cookie is checked at {AuthServiceURL}/v1/auth/me.
	AuthServiceURL string
	// OwnerUserID is the Witems user allowed to use the app (PUNISHER_OWNER_USER_ID); others get 403.
	OwnerUserID string
	// Origin is the Witems origin (PUNISHER_WITEMS_ORIGIN, e.g. https://witems.kr). A request carrying another Origin is refused.
	Origin string
	// GatewayURL is the host claude-gateway (CLAUDE_GATEWAY_URL); generation goes there instead of a local CLI.
	GatewayURL string
	// InternalToken authenticates service-to-service calls (WITEMS_INTERNAL_TOKEN: claude-gateway, rss-server).
	InternalToken string
	// RSSServerURL enables reading signals for chapter suggestions (PUNISHER_RSS_SERVER_URL; empty = off).
	RSSServerURL string
}

// Load reads PUNISHER_* env vars over the defaults.
func Load() Config {
	lectureModel := readEnv("PUNISHER_LECTURE_MODEL", "sonnet")
	fastModel := readEnv("PUNISHER_FAST_MODEL", "haiku")
	checkModel := readEnv("PUNISHER_CHECK_MODEL", lectureModel)
	return Config{
		Port:          readInt("PUNISHER_PORT", 0),
		DataDir:       readEnv("PUNISHER_DATA_DIR", DefaultDataDir()),
		CurriculumDir: strings.TrimSpace(os.Getenv("PUNISHER_CURRICULUM_DIR")),
		OpenBrowser:   readEnv("PUNISHER_OPEN_BROWSER", "true") != "false",
		Locale:        DefaultLocale(),

		ClaudeBin:         strings.TrimSpace(os.Getenv("PUNISHER_CLAUDE_BIN")),
		LectureModel:      lectureModel,
		FastModel:         fastModel,
		Concurrency:       readInt("PUNISHER_CONCURRENCY", 3),
		DailyLectureLimit: readInt("PUNISHER_DAILY_LECTURE_LIMIT", 0),
		DailyTokenLimit:   readInt("PUNISHER_DAILY_TOKEN_LIMIT", 0),
		DailyCallLimit:    readInt("PUNISHER_DAILY_CALL_LIMIT", 0),
		LibraryAhead:      readInt("PUNISHER_LIBRARY_AHEAD", 1),
		NightAhead:        readInt("PUNISHER_NIGHT_AHEAD", 3),
		NightStartHour:    readHour("PUNISHER_NIGHT_START_HOUR", 2),
		NightEndHour:      readHour("PUNISHER_NIGHT_END_HOUR", 6),
		StudyPregenerate:  readEnv("PUNISHER_STUDY_PREGENERATE", "true") != "false",
		LectureMaxTokens:  readInt("PUNISHER_LECTURE_MAX_TOKENS", 24000),
		SchedulerEnabled:  readEnv("PUNISHER_SCHEDULER_ENABLED", "true") != "false",
		SchedulerInterval: readDuration("PUNISHER_SCHEDULER_INTERVAL", 60*time.Minute),

		CheckModel:           checkModel,
		CheckRegradeModel:    readEnv("PUNISHER_CHECK_REGRADE_MODEL", otherModel(checkModel, lectureModel, fastModel)),
		DailyCheckTokenLimit: readInt("PUNISHER_DAILY_CHECK_TOKEN_LIMIT", 0),

		BackfillReviewEnabled: readEnv("PUNISHER_BACKFILL_REVIEW_ENABLED", "true") != "false",

		Witems: loadWitems(),
	}
}

func loadWitems() Witems {
	return Witems{
		Enabled:        strings.EqualFold(strings.TrimSpace(os.Getenv("PUNISHER_MODE")), "witems"),
		Addr:           readEnv("PUNISHER_LISTEN", "0.0.0.0:8080"),
		AuthServiceURL: strings.TrimRight(strings.TrimSpace(os.Getenv("AUTH_SERVICE_URL")), "/"),
		OwnerUserID:    strings.TrimSpace(os.Getenv("PUNISHER_OWNER_USER_ID")),
		Origin:         strings.TrimRight(strings.TrimSpace(os.Getenv("PUNISHER_WITEMS_ORIGIN")), "/"),
		GatewayURL:     strings.TrimRight(strings.TrimSpace(os.Getenv("CLAUDE_GATEWAY_URL")), "/"),
		InternalToken:  strings.TrimSpace(os.Getenv("WITEMS_INTERNAL_TOKEN")),
		RSSServerURL:   strings.TrimRight(strings.TrimSpace(os.Getenv("PUNISHER_RSS_SERVER_URL")), "/"),
	}
}

// Validate reports the first missing setting the Witems mode cannot start without.
func (w Witems) Validate() error {
	switch {
	case w.AuthServiceURL == "":
		return errors.New("PUNISHER_MODE=witems needs AUTH_SERVICE_URL")
	case w.OwnerUserID == "":
		return errors.New("PUNISHER_MODE=witems needs PUNISHER_OWNER_USER_ID")
	case w.Origin == "":
		return errors.New("PUNISHER_MODE=witems needs PUNISHER_WITEMS_ORIGIN")
	case w.GatewayURL == "":
		return errors.New("PUNISHER_MODE=witems needs CLAUDE_GATEWAY_URL")
	}
	return nil
}

// otherModel is the regrade default: of the lecture and fast models, the one grading does not use.
func otherModel(grading, lecture, fast string) string {
	if grading == lecture {
		return fast
	}
	return lecture
}

// DefaultDataDir is ~/Library/Application Support/Punisher on macOS and
// $XDG_DATA_HOME/punisher (~/.local/share/punisher) elsewhere. A ~/.punisher
// folder from older versions is kept in use.
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".punisher"
	}
	return defaultDataDir(runtime.GOOS, home, os.Getenv("XDG_DATA_HOME"), dirExists)
}

func defaultDataDir(goos, home, xdgDataHome string, exists func(string) bool) string {
	if goos == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "Punisher")
	}
	if legacy := filepath.Join(home, ".punisher"); exists(legacy) {
		return legacy
	}
	if xdgDataHome != "" && filepath.IsAbs(xdgDataHome) {
		return filepath.Join(xdgDataHome, "punisher")
	}
	return filepath.Join(home, ".local", "share", "punisher")
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
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

// readHour reads a KST hour of day (0–23); anything else falls back.
func readHour(key string, fallback int) int {
	if h := readInt(key, fallback); h <= 23 {
		return h
	}
	return fallback
}
