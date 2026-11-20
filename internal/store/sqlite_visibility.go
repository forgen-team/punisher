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

// openableLecture: lecture l is one the user can open (GetLecture's rule): a library lecture, or
// one made for this user. user is an SQL expression (a parameter or a column).
func openableLecture(l, user string) string {
	return "(" + l + ".kind = 'library' OR " + l + ".requested_by = " + user + ")"
}

// checkableLecture: lecture l can be opened and checked (loadCheckLecture's rules): openable, its
// concept not retired and visible to the user, and at least one of its objectives still active.
// Progress rows outlive what they point at (a data move retires concepts, an old identity's
// lectures stay behind): a list offering 확인하기 asks this, or the check answers 404.
func checkableLecture(l, user string) string {
	return openableLecture(l, user) +
		" AND EXISTS (SELECT 1 FROM concepts ck WHERE ck.id = " + l + ".concept_id AND ck.status <> 'retired' AND " + visibleTo("ck", user) + ")" +
		" AND EXISTS (SELECT 1 FROM objectives ok WHERE ok.concept_id = " + l + ".concept_id AND ok.status = 'active'" +
		" AND ok.id IN (SELECT value FROM json_each(" + l + ".objective_ids)))"
}
