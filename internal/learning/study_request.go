package learning

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Study request sources (design §3.7).
const (
	SourceRSS      = "rss"
	SourceManual   = "manual"
	SourceGuardian = "guardian"
)

// IsValidSource reports whether s is an accepted study-request source.
func IsValidSource(s string) bool {
	return s == SourceRSS || s == SourceManual || s == SourceGuardian
}

// DeriveIdempotencyKey builds a deterministic key when the client did not send
// one, so the same (source, sourceRef, query) is stored once per user.
// normalizedQuery must already be normalized (curriculum.NormalizeAlias).
func DeriveIdempotencyKey(source, sourceRef, normalizedQuery string) string {
	sum := sha256.Sum256([]byte(source + "\x00" + strings.TrimSpace(sourceRef) + "\x00" + normalizedQuery))
	return "derived:" + hex.EncodeToString(sum[:16])
}

// ObjectiveKey is the canonical form of an objective set: sorted, de-duplicated
// ids joined by ','. Lectures are unique on (concept, objective_key, depth, version),
// so a re-titled regeneration of the same objectives is not a new lecture.
func ObjectiveKey(objectiveIDs []string) string {
	seen := make(map[string]struct{}, len(objectiveIDs))
	ids := make([]string, 0, len(objectiveIDs))
	for _, id := range objectiveIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}
