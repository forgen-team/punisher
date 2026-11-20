package learning

import "github.com/forgen-team/punisher/internal/curriculum"

// noviceFoundationsStart: where a novice starts the foundations track
// (learner-level design ②: concepts 5·8·9·10·11 — terminal, HTTP, JSON, SQL, git).
var noviceFoundationsStart = []string{
	"found.terminal-and-shell",
	"found.http-first-look",
	"found.json-and-api",
	"found.tables-and-sql",
	"found.git-version-control",
}

// FoundationsHidden reports whether the foundations track is hidden by default
// for a reader level: practitioner, advanced and "" (not chosen) never get it
// recommended and the map collapses it until the user expands it. Changing the
// level only changes this default (no stored state depends on it).
func FoundationsHidden(level string) bool {
	return level != ReaderBeginner && level != ReaderNovice
}

// FoundationsOrder is the order today's picks walk the foundations track for a
// level: beginner = the whole track in order; novice = the five starting
// concepts, then the rest in track order; hidden levels = nil.
func FoundationsOrder(level string) []string {
	switch level {
	case ReaderBeginner:
		return append([]string{}, curriculum.FoundationsTrack...)
	case ReaderNovice:
		out := append([]string{}, noviceFoundationsStart...)
		first := make(map[string]bool, len(noviceFoundationsStart))
		for _, id := range noviceFoundationsStart {
			first[id] = true
		}
		for _, id := range curriculum.FoundationsTrack {
			if !first[id] {
				out = append(out, id)
			}
		}
		return out
	}
	return nil
}

// addFoundations recommends the not-yet-learned foundations concepts in the
// level's order. Inside the track the order is the path, so a concept whose
// in-track prerequisite is still open may be suggested (a novice skips the
// first concepts on purpose); other areas never depend on the track.
func (b *todayBuilder) addFoundations() {
	for _, id := range FoundationsOrder(b.in.ReaderLevel) {
		if b.full() {
			return
		}
		c, ok := b.learnable(id)
		if !ok || c.Proposed {
			continue
		}
		b.add(c, Recommendation{Kind: RecEssentialGap, Reason: inLocale(b.in.Locale,
			"0단계 기초 경로에서 아직 배우지 않은 다음 개념입니다. 이어지는 강의를 읽는 데 바탕이 됩니다.",
			"The next concept you have not learned on the Foundations path. It is the groundwork for the lectures that follow.")})
	}
}
