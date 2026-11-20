package learning

// MaxUncheckedCards caps the "○○ 확인하기" cards on the today screen.
const MaxUncheckedCards = 3

// UncheckedLecture is a finished lecture whose concept was never checked
// ('확인하기'), so no review is scheduled from it yet. Most recent first.
type UncheckedLecture struct {
	LectureID string
	ConceptID string
	Title     string
}

// UncheckedCard is a "○○ 확인하기" card: a check on a finished lecture, which
// is what puts its objectives on the review schedule.
type UncheckedCard struct {
	LectureID   string `json:"lectureId"`
	ConceptID   string `json:"conceptId"`
	ConceptName string `json:"conceptName"`
	Area        string `json:"area"`
	Title       string `json:"title"`
}

// LatestVersions points each lecture remade with '다시 만들기' at its newest
// published version of the same concept (the check should ask about the
// corrected lecture; a link to another concept is ignored).
func LatestVersions(in []UncheckedLecture, newer map[string]NewerVersion) []UncheckedLecture {
	out := make([]UncheckedLecture, 0, len(in))
	for _, u := range in {
		if n, ok := newest(u.LectureID, newer); ok && n.Checkable && n.Item.ConceptID == u.ConceptID {
			u.LectureID, u.Title = n.Item.LectureID, n.Item.Title
		}
		out = append(out, u)
	}
	return out
}

// uncheckedCards: one card per concept still on the map, most recent first,
// up to MaxUncheckedCards.
func uncheckedCards(in TodayInput) []UncheckedCard {
	out := []UncheckedCard{}
	seen := map[string]bool{}
	for _, u := range in.Unchecked {
		c, ok := in.Concepts[u.ConceptID]
		if !ok || seen[u.ConceptID] || len(out) >= MaxUncheckedCards {
			continue
		}
		seen[u.ConceptID] = true
		out = append(out, UncheckedCard{LectureID: u.LectureID, ConceptID: c.ID, ConceptName: c.Name, Area: c.Area, Title: u.Title})
	}
	return out
}
