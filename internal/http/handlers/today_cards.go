package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/store"
)

// appBasePath is where the Witems shell mounts punisher (contract §2 basePath = /apps/<name>).
// TodayRow.href is a full shell path, so every link starts with it (rss.md §8-3).
const appBasePath = "/apps/punisher"

// TodayCard 는 Witems 셸의 요약 카드 계약(rss.md §8-3)이다. 앱은 데이터만 넘기고 셸이 그린다.
type TodayCard struct {
	ID    string     `json:"id"`
	Kind  string     `json:"kind"` // todo | read | study
	Title string     `json:"title"`
	Count int        `json:"count,omitempty"`
	Rows  []TodayRow `json:"rows"`
	More  *TodayMore `json:"more,omitempty"`
	AsOf  string     `json:"asOf"`
}

// TodayRow is one 1-line title + 1-line meta; href carries basePath and never a hash.
type TodayRow struct {
	Title string `json:"title"`
	Meta  string `json:"meta,omitempty"`
	Href  string `json:"href"`
}

// TodayMore is the link at the right end of the card group head.
type TodayMore struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

// todayCardsInput is everything buildTodayCards needs; it performs no I/O.
type todayCardsInput struct {
	Locale string
	Now    time.Time
	Review learning.ReviewToday
	// Unchecked: finished lectures not checked yet ("확인 대기"). Cards: review cards due now.
	Unchecked, Cards int
	Synthesis        bool // 종합 문제가 오늘 열려 있다
	RestDay          bool
	// Resume: the most recently read in-progress lecture (nil = none).
	Resume *learning.Recommendation
	// Next: today's first recommendation that is not a resume (nil = none).
	// NextLectureID: its already-made lecture ("" = none yet → the start screen makes it).
	Next          *learning.Recommendation
	NextLectureID string
}

// buildTodayCards makes the cards of GET /v1/today-cards: 복습(todo) then 이어보기(study).
// A card with no rows is left out; with nothing at all the result is an empty slice, not nil.
func buildTodayCards(in todayCardsInput) []TodayCard {
	cards := []TodayCard{}
	asOf := in.Now.Format(time.RFC3339)
	more := &TodayMore{Label: inLocale(in.Locale, "모두 보기", "See all"), Href: appBasePath + "/"}
	if c, ok := reviewCard(in); ok {
		c.More, c.AsOf = more, asOf
		cards = append(cards, c)
	}
	if c, ok := studyCard(in); ok {
		c.More, c.AsOf = more, asOf
		cards = append(cards, c)
	}
	return cards
}

// reviewCard: 복습 + 확인 대기 + 카드를 행 하나로. 합이 0 이면 카드를 뺀다.
// 묶음 머리가 이미 "복습 n" 이므로 행 제목은 할 일("문항 12개 · 카드 8장 복습 · 강의 3개 확인"),
// meta 는 걸리는 시간만 둔다(같은 말을 머리·제목·meta 에 되풀이하지 않는다). 쉬는 날은 숫자 없이 알린다.
func reviewCard(in todayCardsInput) (TodayCard, bool) {
	total := in.Review.Due + in.Unchecked + in.Cards
	if total <= 0 {
		return TodayCard{}, false
	}
	card := TodayCard{ID: "review", Kind: "todo", Title: inLocale(in.Locale, "복습", "Review"), Rows: []TodayRow{{Href: appBasePath + "/"}}}
	if in.RestDay {
		card.Rows[0].Title = inLocale(in.Locale, "오늘은 쉬는 날이에요", "Today is a rest day")
		return card, true
	}
	card.Count = total
	card.Rows[0].Title = reviewTodo(in)
	// 카드는 한 장에 30초쯤(reminder 와 같은 기준). 확인 대기는 문항 수가 확인마다 달라 시간에 넣지 않는다.
	if minutes := in.Review.EstimatedMinutes + (in.Cards+1)/2; minutes > 0 {
		card.Rows[0].Meta = fmt.Sprintf(inLocale(in.Locale, "약 %d분", "about %d min"), minutes)
	}
	return card, true
}

// reviewTodo is the review row title as things to do, grouped by verb:
// "문항 12개 · 카드 8장 복습 · 강의 3개 확인 · 종합 문제 풀기" (0 인 것은 뺀다).
func reviewTodo(in todayCardsInput) string {
	var review []string
	if in.Review.Due > 0 {
		review = append(review, fmt.Sprintf(inLocale(in.Locale, "문항 %d개", "%d questions"), in.Review.Due))
	}
	if in.Cards > 0 {
		review = append(review, fmt.Sprintf(inLocale(in.Locale, "카드 %d장", "%d cards"), in.Cards))
	}
	var todo []string
	if len(review) > 0 {
		todo = append(todo, fmt.Sprintf(inLocale(in.Locale, "%s 복습", "Review %s"), strings.Join(review, " · ")))
	}
	if in.Unchecked > 0 {
		todo = append(todo, fmt.Sprintf(inLocale(in.Locale, "강의 %d개 확인", "Check %d lectures"), in.Unchecked))
	}
	if in.Synthesis {
		todo = append(todo, inLocale(in.Locale, "종합 문제 풀기", "Take the synthesis quiz"))
	}
	return strings.Join(todo, " · ")
}

// studyCard: 읽던 강의 이어보기 + 다음 강의. 둘 다 없으면 카드를 뺀다.
func studyCard(in todayCardsInput) (TodayCard, bool) {
	card := TodayCard{ID: "study", Kind: "study", Title: inLocale(in.Locale, "이어보기", "Continue")}
	if r := in.Resume; r != nil {
		card.Rows = append(card.Rows, resumeRow(in.Locale, *r))
	}
	if n := in.Next; n != nil {
		card.Rows = append(card.Rows, nextRow(in.Locale, *n, in.NextLectureID))
	}
	return card, len(card.Rows) > 0
}

// resumeRow: 개념 이름 · "강의 제목 · 40% 읽음" → /lectures/:id?at=<앵커>.
func resumeRow(locale string, r learning.Recommendation) TodayRow {
	row := TodayRow{Title: r.ConceptName, Href: appBasePath + "/lectures/" + url.PathEscape(r.LectureID)}
	var meta []string
	if row.Title == "" {
		row.Title = r.Title // 개념 이름이 비면 강의 제목을 제목으로 쓰고 meta 에는 겹쳐 넣지 않는다
	} else if r.Title != "" && r.Title != r.ConceptName {
		meta = append(meta, r.Title)
	}
	if r.Percent > 0 {
		meta = append(meta, fmt.Sprintf(inLocale(locale, "%d%% 읽음", "%d%% read"), r.Percent))
	}
	row.Meta = strings.Join(meta, " · ")
	if r.Anchor != "" {
		row.Href += "?at=" + url.QueryEscape(r.Anchor)
	}
	return row
}

// nextRow: 개념 이름 · "왜 오늘 · 약 12분" → 만든 강의가 있으면 /lectures/:id, 없으면 /start?concept=.
func nextRow(locale string, n learning.Recommendation, lectureID string) TodayRow {
	row := TodayRow{Title: n.ConceptName, Href: appBasePath + "/start?concept=" + url.QueryEscape(n.ConceptID)}
	if row.Title == "" {
		row.Title = n.Title
	}
	if lectureID != "" {
		row.Href = appBasePath + "/lectures/" + url.PathEscape(lectureID)
	}
	meta := []string{whyToday(locale, n.Kind)}
	if n.EstimatedMinutes > 0 {
		meta = append(meta, fmt.Sprintf(inLocale(locale, "약 %d분", "about %d min"), n.EstimatedMinutes))
	}
	row.Meta = strings.Join(meta, " · ")
	return row
}

// whyToday is the short reason of a recommendation (Recommendation.Reason is a full sentence — too long for one meta line).
func whyToday(locale, kind string) string {
	switch kind {
	case learning.RecGoalPath:
		return inLocale(locale, "목표 경로의 다음 강의", "Next on your goal path")
	case learning.RecStudyRequest:
		return inLocale(locale, "학습 요청에 답해요", "Answers your request")
	case learning.RecEssentialGap:
		return inLocale(locale, "꼭 알아야 할 기초", "Essential basics")
	case learning.RecNewGoal:
		return inLocale(locale, "새 목표 제안", "New goal suggestion")
	default:
		return inLocale(locale, "오늘의 추천", "Today's pick")
	}
}

func (h *LearnHandler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

// TodayCards handles GET /v1/today-cards — the summary cards the Witems 오늘 home draws (rss.md §8-3).
// It reuses what /v1/today computes (recommendations, review counts, unchecked lectures) plus
// due cards and the rest-day pref, and returns [] — never null — when there is nothing to show.
func (h *LearnHandler) TodayCards(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	ctx, now := r.Context(), h.now()
	res, err := h.Svc.Today(ctx, userID, 1) // 첫 추천 하나면 된다(이어보기는 한도 밖)
	if err != nil {
		writeServiceError(w, r, "today cards", err)
		return
	}
	in := todayCardsInput{Locale: h.localeOf(r, userID), Now: now, Review: res.Review, Unchecked: len(res.Unchecked)}
	if st, ok := h.Store.(store.StudyStore); ok { // 복습 카드·쉬는 날은 SQLite 에 있다(StudyStore 가 없는 fake 저장소는 건너뜀)
		if in.Cards, _, err = st.CardCounts(ctx, userID, now); err != nil {
			writeServiceError(w, r, "today cards", err)
			return
		}
		prefs, err := st.GetStudyPrefs(ctx, userID)
		if err != nil {
			writeServiceError(w, r, "today cards", err)
			return
		}
		in.RestDay = learning.IsRestDay(now, prefs.RestDays)
	}
	if due, err := h.Svc.SynthesisDue(ctx, userID, now); err != nil {
		log.Printf("today cards synthesis for %s: %v", userID, err) // 종합 표시가 없어도 카드는 그대로 보인다.
	} else {
		in.Synthesis = due
	}
	for i := range res.Items {
		item := res.Items[i]
		switch {
		case item.Kind == learning.RecResume && in.Resume == nil:
			in.Resume = &item
		case item.Kind != learning.RecResume && in.Next == nil:
			in.Next = &item
			in.NextLectureID = item.LectureID
		}
	}
	if in.Next != nil && in.NextLectureID == "" {
		if lec, ok := h.Svc.NextLecture(ctx, userID, in.Next.ConceptID); ok {
			in.NextLectureID = lec.ID
		}
	}
	writeJSON(w, http.StatusOK, buildTodayCards(in))
}
