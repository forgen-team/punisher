package learning

import "github.com/forgen-team/punisher/internal/curriculum"

// AliasIndex maps a normalized alias/name (curriculum.NormalizeAlias) to a
// concept id. Concept ids map to themselves.
type AliasIndex map[string]string

// ResolveExact maps a free query to a concept id through the alias index.
// Every alias of a concept (rename, Korean/English) resolves to the same id,
// so dedup on (concept, objective, depth) is name-independent.
func ResolveExact(index AliasIndex, query string) (string, bool) {
	id, ok := index[curriculum.NormalizeAlias(query)]
	return id, ok
}
