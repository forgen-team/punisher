package config

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Locales of the screen and generated text (i18n design L1).
const (
	LocaleKO = "ko"
	LocaleEN = "en"
)

// DefaultLocale is the language of a user who never saved settings (i18n
// design L2): PUNISHER_LOCALE (ko|en) when set, else the OS locale.
func DefaultLocale() string {
	if l, ok := parseLocale(os.Getenv("PUNISHER_LOCALE")); ok {
		return l
	}
	return detectLocale(os.Getenv, runtime.GOOS, appleLanguages)
}

// parseLocale reads an explicit ko|en (also ko_KR, en-US …).
func parseLocale(v string) (string, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	switch {
	case strings.HasPrefix(v, LocaleKO):
		return LocaleKO, true
	case strings.HasPrefix(v, LocaleEN):
		return LocaleEN, true
	}
	return "", false
}

// detectLocale: the first of LC_ALL, LC_MESSAGES, LANG that names a language
// decides (ko… → ko, else en); "C"/"POSIX" name none. Without one, macOS uses
// the first of AppleLanguages (apps opened from Finder or launchd get no LANG).
func detectLocale(getenv func(string) string, goos string, apple func() string) string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.TrimSpace(getenv(key))
		if v == "" || v == "C" || v == "POSIX" || strings.HasPrefix(v, "C.") {
			continue
		}
		return koOrEN(v)
	}
	if goos == "darwin" {
		if first := apple(); first != "" {
			return koOrEN(first)
		}
	}
	return LocaleEN
}

func koOrEN(v string) string {
	if strings.HasPrefix(strings.ToLower(v), LocaleKO) {
		return LocaleKO
	}
	return LocaleEN
}

// appleLanguages is the first entry of `defaults read -g AppleLanguages` ("" on failure).
func appleLanguages() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "defaults", "read", "-g", "AppleLanguages").Output()
	if err != nil {
		return ""
	}
	return firstAppleLanguage(string(out))
}

// firstAppleLanguage parses the plist array `( "ko-KR", "en-KR" )`.
func firstAppleLanguage(out string) string {
	for _, line := range strings.Split(out, "\n") {
		v := strings.Trim(strings.TrimSpace(line), `(),"`)
		if v != "" {
			return v
		}
	}
	return ""
}
