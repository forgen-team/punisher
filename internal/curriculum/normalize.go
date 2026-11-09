package curriculum

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// NormalizeAlias canonicalizes a name/alias/query for exact matching:
// Unicode NFC, trim, lowercase, and collapse internal whitespace.
func NormalizeAlias(s string) string {
	s = norm.NFC.String(s)
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Join(strings.Fields(s), " ")
}
