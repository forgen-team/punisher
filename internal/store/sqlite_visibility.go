package store

// visibleTo is the SQL condition for "concept row `alias` is visible to the
// user bound at placeholder `param`": shared seed/dynamic concepts
// (owner_user_id IS NULL) plus the user's own chapter concepts; proposed
// concepts only to the user who proposed them. Every query that lists or
// resolves concepts for a user must apply it (user isolation of chapters).
// The standalone app has one user, but the queries keep the same shape as
// witems so syncs apply unchanged (docs/SYNC.md).
func visibleTo(alias, param string) string {
	return "((" + alias + ".owner_user_id IS NULL OR " + alias + ".owner_user_id = " + param + ")" +
		" AND (" + alias + ".status <> 'proposed' OR " + alias + ".proposed_by = " + param + "))"
}
