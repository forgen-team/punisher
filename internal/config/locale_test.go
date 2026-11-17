package config

import "testing"

func TestDetectLocale(t *testing.T) {
	cases := []struct {
		name  string
		env   map[string]string
		goos  string
		apple string
		want  string
	}{
		{"LANG ko", map[string]string{"LANG": "ko_KR.UTF-8"}, "linux", "", LocaleKO},
		{"LANG en", map[string]string{"LANG": "en_US.UTF-8"}, "darwin", "ko-KR", LocaleEN},
		{"LC_ALL wins", map[string]string{"LC_ALL": "ko_KR.UTF-8", "LANG": "en_US.UTF-8"}, "linux", "", LocaleKO},
		{"LC_MESSAGES before LANG", map[string]string{"LC_MESSAGES": "en_GB", "LANG": "ko_KR"}, "linux", "", LocaleEN},
		{"other language", map[string]string{"LANG": "ja_JP.UTF-8"}, "linux", "", LocaleEN},
		{"macOS AppleLanguages", map[string]string{}, "darwin", "ko-KR", LocaleKO},
		{"macOS C locale falls back", map[string]string{"LANG": "C"}, "darwin", "ko-KR", LocaleKO},
		{"macOS English", map[string]string{}, "darwin", "en-US", LocaleEN},
		{"linux without env", map[string]string{}, "linux", "ko-KR", LocaleEN},
		{"nothing", map[string]string{}, "darwin", "", LocaleEN},
	}
	for _, c := range cases {
		got := detectLocale(func(k string) string { return c.env[k] }, c.goos, func() string { return c.apple })
		if got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

func TestFirstAppleLanguage(t *testing.T) {
	if got := firstAppleLanguage("(\n    \"ko-KR\",\n    \"en-KR\"\n)\n"); got != "ko-KR" {
		t.Fatalf("plist array: %q", got)
	}
	if got := firstAppleLanguage("(\n    en\n)\n"); got != "en" {
		t.Fatalf("unquoted: %q", got)
	}
	if got := firstAppleLanguage(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
}

func TestDefaultLocaleOverride(t *testing.T) {
	t.Setenv("PUNISHER_LOCALE", "en")
	t.Setenv("LANG", "ko_KR.UTF-8")
	if got := DefaultLocale(); got != LocaleEN {
		t.Fatalf("PUNISHER_LOCALE=en: %s", got)
	}
	t.Setenv("PUNISHER_LOCALE", "ko")
	t.Setenv("LANG", "en_US.UTF-8")
	if got := DefaultLocale(); got != LocaleKO {
		t.Fatalf("PUNISHER_LOCALE=ko: %s", got)
	}
	t.Setenv("PUNISHER_LOCALE", "fr")
	t.Setenv("LC_ALL", "ko_KR.UTF-8")
	if got := DefaultLocale(); got != LocaleKO {
		t.Fatalf("unknown PUNISHER_LOCALE falls back to the OS: %s", got)
	}
}
