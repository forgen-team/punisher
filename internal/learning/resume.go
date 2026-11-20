package learning

// NewerVersion is a published lecture made by '다시 만들기' from an older one.
type NewerVersion struct {
	Item ResumeItem // LectureID/ConceptID/Title/EstimatedMinutes of the new version
	// Started: the user already has progress on the new version (it shows up,
	// or was finished, on its own), so the old row is just dropped.
	Started bool
	// Checkable: the check can take the new version too (its concept and objectives are still
	// active and visible); only then does 확인하기 move to it.
	Checkable bool
}

// ReplaceRegenerated points 이어보기 at the newest regenerated version: an
// in-progress lecture that was remade (e.g. after a failed review) is replaced
// by its published new version, starting from the top. newer maps an old
// lecture ID to its newest published version; chains are followed.
func ReplaceRegenerated(resume []ResumeItem, newer map[string]NewerVersion) []ResumeItem {
	if len(newer) == 0 {
		return resume
	}
	out := make([]ResumeItem, 0, len(resume))
	seen := map[string]bool{}
	for _, r := range resume {
		n, ok := newest(r.LectureID, newer)
		if !ok {
			if !seen[r.LectureID] {
				seen[r.LectureID] = true
				out = append(out, r)
			}
			continue
		}
		if n.Started || seen[n.Item.LectureID] {
			continue
		}
		seen[n.Item.LectureID] = true
		item := n.Item
		item.Anchor, item.Percent, item.ParentLectureID = "", 0, r.ParentLectureID
		out = append(out, item)
	}
	return out
}

// newest follows old → new → newer links to the last version (bounded, so a
// malformed cycle cannot loop).
func newest(lectureID string, newer map[string]NewerVersion) (NewerVersion, bool) {
	n, ok := newer[lectureID]
	if !ok {
		return NewerVersion{}, false
	}
	for i := 0; i < len(newer); i++ {
		next, more := newer[n.Item.LectureID]
		if !more {
			break
		}
		n = next
	}
	return n, true
}
