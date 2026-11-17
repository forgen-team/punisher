package main

import (
	"sync"

	"github.com/forgen-team/punisher/internal/config"
)

// cliEnglish: terminal messages follow the OS language (the app setting lives
// in the database, which is not open yet for most of these messages).
var cliEnglish = sync.OnceValue(func() bool { return config.DefaultLocale() == "en" })

// tr picks the terminal message for the OS language.
func tr(ko, en string) string {
	if cliEnglish() {
		return en
	}
	return ko
}
