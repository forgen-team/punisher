package store

import (
	"strings"
	"sync/atomic"

	"github.com/forgen-team/punisher/internal/learning"
)

// Language-aware reads of curriculum text (i18n §2): the English overlay
// column when the language is en and it is filled, else the Korean canonical.
// lang is a SQL expression — a bound parameter ("$3") or userLocaleSQL("$1").
// Change detection and snapshots (objective_text) keep reading the ko columns.

// locText is column col of alias in language lang.
func locText(alias, col, lang string) string {
	ko, en := alias+"."+col, alias+"."+col+"_en"
	return "(CASE WHEN " + lang + " = 'en' AND " + en + " <> '' THEN " + en + " ELSE " + ko + " END)"
}

// locName is the concept name of alias in language lang.
func locName(alias, lang string) string { return locText(alias, "name", lang) }

// locSummary is the concept summary of alias in language lang.
func locSummary(alias, lang string) string { return locText(alias, "summary", lang) }

// locObjective is the objective sentence of alias in language lang.
func locObjective(alias, lang string) string { return locText(alias, "text", lang) }

// locCases is the practical_cases JSON array (TEXT) of alias in language lang.
func locCases(alias, lang string) string {
	ko, en := alias+".practical_cases", alias+".practical_cases_en"
	return "(CASE WHEN " + lang + " = 'en' AND json_array_length(" + en + ") > 0 THEN " + en + " ELSE " + ko + " END)"
}

// userLocaleSQL is the locale of user param (i18n §1); the store default when
// the user has not saved settings (the standalone default follows the OS
// locale: config → SetDefaultLocale). The default is a marker that rewrite
// fills at query time, so package-level SQL built at init stays correct.
func userLocaleSQL(param string) string {
	return `COALESCE((SELECT uls.locale FROM user_learning_settings uls WHERE uls.user_id = ` + param + `), ` + defaultLocaleMarker + `)`
}

// defaultLocaleMarker is replaced by the default locale literal in rewrite.
const defaultLocaleMarker = "'@default_locale@'"

var defaultIsEN atomic.Bool

// SetDefaultLocale sets the locale of users without saved settings (ko|en;
// anything else = ko). Call before serving requests.
func SetDefaultLocale(locale string) {
	defaultIsEN.Store(learning.NormalizeLocale(locale) == learning.LocaleEN)
}

// defaultLocale is the locale of users without saved settings.
func defaultLocale() string {
	if defaultIsEN.Load() {
		return learning.LocaleEN
	}
	return learning.LocaleKO
}

// fillDefaultLocale puts the default locale literal in place of the marker.
func fillDefaultLocale(q string) string {
	if !strings.Contains(q, defaultLocaleMarker) {
		return q
	}
	return strings.ReplaceAll(q, defaultLocaleMarker, "'"+defaultLocale()+"'")
}
