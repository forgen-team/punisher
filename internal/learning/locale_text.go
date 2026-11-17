package learning

// inLocale is the server-made screen text (recommendation and suggestion
// reasons) in the user's locale (i18n L4): en when the locale is en, else ko.
func inLocale(locale, ko, en string) string {
	if locale == LocaleEN {
		return en
	}
	return ko
}
