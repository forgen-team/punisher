package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/forgen-team/punisher/internal/learning"
	appMiddleware "github.com/forgen-team/punisher/internal/middleware"
	"github.com/forgen-team/punisher/internal/services"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/forgen-team/punisher/migrations"
)

var cardsNow = time.Date(2026, 11, 27, 7, 10, 0, 0, time.FixedZone("KST", 9*3600))

func resumeRec() *learning.Recommendation {
	return &learning.Recommendation{Kind: learning.RecResume, ConceptID: "c-12", ConceptName: "HTTP 캐시", LectureID: "12",
		Title: "Cache-Control 로 응답 재사용하기", Percent: 40, Anchor: "p7"}
}

func nextRec() *learning.Recommendation {
	return &learning.Recommendation{Kind: learning.RecGoalPath, ConceptID: "c-41", ConceptName: "ETag 와 조건부 요청", EstimatedMinutes: 12}
}

// cardByID finds a card by its id (the order is review → study).
func cardByID(cards []TodayCard, id string) *TodayCard {
	for i := range cards {
		if cards[i].ID == id {
			return &cards[i]
		}
	}
	return nil
}

func TestTodayCardsReview(t *testing.T) {
	base := todayCardsInput{Locale: "ko", Now: cardsNow, Review: learning.ReviewToday{Due: 12, EstimatedMinutes: 12}, Unchecked: 3, Cards: 8}

	t.Run("복습·확인 대기·카드 합이 count, meta 는 0 인 항목 빼고", func(t *testing.T) {
		c := cardByID(buildTodayCards(base), "review")
		if c == nil {
			t.Fatal("no review card")
		}
		if c.Kind != "todo" || c.Title != "복습" || c.Count != 23 || len(c.Rows) != 1 {
			t.Fatalf("card %+v", c)
		}
		row := c.Rows[0]
		if row.Title != "복습" || row.Meta != "복습 12 · 확인 대기 3 · 카드 8 · 약 16분" || row.Href != "/apps/punisher/" {
			t.Fatalf("row %+v", row)
		}
		if c.More == nil || c.More.Label != "모두 보기" || c.More.Href != "/apps/punisher/" || c.AsOf != "2026-11-27T07:10:00+09:00" {
			t.Fatalf("more/asOf %+v %q", c.More, c.AsOf)
		}
	})

	t.Run("0 인 항목 빼고 종합 문제는 뒤에", func(t *testing.T) {
		in := base
		in.Review, in.Unchecked, in.Cards, in.Synthesis = learning.ReviewToday{Due: 4}, 0, 0, true
		c := cardByID(buildTodayCards(in), "review")
		if c == nil || c.Rows[0].Meta != "복습 4 · 종합 1" || c.Count != 4 {
			t.Fatalf("card %+v", c)
		}
	})

	t.Run("합이 0 이면 카드를 뺀다(종합만 있어도)", func(t *testing.T) {
		in := base
		in.Review, in.Unchecked, in.Cards, in.Synthesis = learning.ReviewToday{}, 0, 0, true
		if got := buildTodayCards(in); len(got) != 0 {
			t.Fatalf("cards %+v", got)
		}
	})

	t.Run("쉬는 날은 숫자 없이 카드는 남긴다", func(t *testing.T) {
		in := base
		in.RestDay = true
		c := cardByID(buildTodayCards(in), "review")
		if c == nil || c.Rows[0].Meta != "오늘은 쉬는 날이에요" || c.Count != 0 {
			t.Fatalf("card %+v", c)
		}
		raw, _ := json.Marshal(c)
		if strings.Contains(string(raw), `"count"`) || strings.ContainsAny(c.Rows[0].Meta, "0123456789") {
			t.Fatalf("rest day must not carry numbers: %s", raw)
		}
	})

	t.Run("영어 locale", func(t *testing.T) {
		in := base
		in.Locale = "en"
		c := cardByID(buildTodayCards(in), "review")
		if c == nil || c.Title != "Review" || c.Rows[0].Meta != "Review 12 · To check 3 · Cards 8 · about 16 min" {
			t.Fatalf("card %+v", c)
		}
	})
}

func TestTodayCardsStudy(t *testing.T) {
	t.Run("이어보기 + 다음 강의(강의 없음 → start)", func(t *testing.T) {
		cards := buildTodayCards(todayCardsInput{Locale: "ko", Now: cardsNow, Resume: resumeRec(), Next: nextRec()})
		c := cardByID(cards, "study")
		if len(cards) != 1 || c == nil || c.Kind != "study" || c.Title != "이어보기" || len(c.Rows) != 2 {
			t.Fatalf("cards %+v", cards)
		}
		want := []TodayRow{
			{Title: "HTTP 캐시", Meta: "Cache-Control 로 응답 재사용하기 · 40% 읽음", Href: "/apps/punisher/lectures/12?at=p7"},
			{Title: "ETag 와 조건부 요청", Meta: "목표 경로의 다음 강의 · 약 12분", Href: "/apps/punisher/start?concept=c-41"},
		}
		for i := range want {
			if c.Rows[i] != want[i] {
				t.Errorf("row %d = %+v, want %+v", i, c.Rows[i], want[i])
			}
		}
	})

	t.Run("앵커가 없으면 쿼리 없이", func(t *testing.T) {
		r := resumeRec()
		r.Anchor = ""
		c := cardByID(buildTodayCards(todayCardsInput{Now: cardsNow, Resume: r}), "study")
		if c == nil || len(c.Rows) != 1 || c.Rows[0].Href != "/apps/punisher/lectures/12" || strings.Contains(c.Rows[0].Href, "?") {
			t.Fatalf("card %+v", c)
		}
	})

	t.Run("앵커는 쿼리 인코딩, 해시는 쓰지 않는다", func(t *testing.T) {
		r := resumeRec()
		r.Anchor = "pa-한 글&x"
		c := cardByID(buildTodayCards(todayCardsInput{Now: cardsNow, Resume: r}), "study")
		if c == nil || c.Rows[0].Href != "/apps/punisher/lectures/12?at=pa-%ED%95%9C+%EA%B8%80%26x" || strings.Contains(c.Rows[0].Href, "#") {
			t.Fatalf("card %+v", c)
		}
	})

	t.Run("개념 이름이 없으면 강의 제목을 제목으로", func(t *testing.T) {
		r := resumeRec()
		r.ConceptName = ""
		c := cardByID(buildTodayCards(todayCardsInput{Now: cardsNow, Resume: r}), "study")
		if c == nil || c.Rows[0].Title != "Cache-Control 로 응답 재사용하기" || c.Rows[0].Meta != "40% 읽음" {
			t.Fatalf("card %+v", c)
		}
	})

	t.Run("만든 강의가 있는 추천은 그 강의 주소", func(t *testing.T) {
		c := cardByID(buildTodayCards(todayCardsInput{Now: cardsNow, Next: nextRec(), NextLectureID: "31"}), "study")
		if c == nil || len(c.Rows) != 1 || c.Rows[0].Href != "/apps/punisher/lectures/31" {
			t.Fatalf("card %+v", c)
		}
	})

	t.Run("start 주소의 개념 id 는 url 인코딩", func(t *testing.T) {
		n := nextRec()
		n.ConceptID = "web/http cache?x"
		c := cardByID(buildTodayCards(todayCardsInput{Now: cardsNow, Next: n}), "study")
		if c == nil || c.Rows[0].Href != "/apps/punisher/start?concept=web%2Fhttp+cache%3Fx" {
			t.Fatalf("card %+v", c)
		}
	})

	t.Run("읽던 강의 없이 다음 강의만", func(t *testing.T) {
		c := cardByID(buildTodayCards(todayCardsInput{Now: cardsNow, Next: nextRec()}), "study")
		if c == nil || len(c.Rows) != 1 || c.Rows[0].Title != "ETag 와 조건부 요청" {
			t.Fatalf("card %+v", c)
		}
	})
}

// 계약(rss.md §8-3): 카드 제목 12자 이내, 행 최대 3, 행 없는 카드는 없음, href 는 basePath 전체 경로·해시 없음.
func TestTodayCardsContract(t *testing.T) {
	in := todayCardsInput{Locale: "ko", Now: cardsNow, Review: learning.ReviewToday{Due: 1}, Resume: resumeRec(), Next: nextRec()}
	for _, locale := range []string{"ko", "en"} {
		in.Locale = locale
		cards := buildTodayCards(in)
		if len(cards) != 2 || cards[0].ID != "review" || cards[1].ID != "study" {
			t.Fatalf("%s cards %+v", locale, cards)
		}
		rows := 0
		for _, c := range cards {
			if utf8.RuneCountInString(c.Title) > 12 || len(c.Rows) == 0 || len(c.Rows) > 3 || c.AsOf == "" || c.More == nil {
				t.Errorf("%s card %+v breaks the contract", locale, c)
			}
			rows += len(c.Rows)
			for _, row := range c.Rows {
				if !strings.HasPrefix(row.Href, "/apps/punisher/") || strings.Contains(row.Href, "#") || row.Title == "" {
					t.Errorf("%s row %+v", locale, row)
				}
			}
			if _, err := time.Parse(time.RFC3339, c.AsOf); err != nil {
				t.Errorf("asOf %q: %v", c.AsOf, err)
			}
		}
		if rows > 3 {
			t.Errorf("%s: %d rows in total, punisher.md §6 allows 3 lines", locale, rows)
		}
	}
}

func TestTodayCardsEmptyIsArray(t *testing.T) {
	got := buildTodayCards(todayCardsInput{Locale: "ko", Now: cardsNow})
	raw, err := json.Marshal(got)
	if err != nil || string(raw) != "[]" {
		t.Fatalf("empty cards = %s (%v), want []", raw, err)
	}
}

func openTodayCardsHandler(t *testing.T, now time.Time) (*LearnHandler, store.StudyStore, string) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "punisher.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx, migrations.Files); err != nil {
		t.Fatal(err)
	}
	userID, err := appMiddleware.GetUserID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	h := &LearnHandler{Store: st, Svc: &services.Service{Store: st, Keys: services.GatewayKeys{}, Hub: services.NewHub()}, Now: func() time.Time { return now }}
	return h, st, userID
}

func getTodayCards(t *testing.T, h *LearnHandler) (string, []TodayCard) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.TodayCards(rec, httptest.NewRequest(http.MethodGet, "/v1/today-cards", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var cards []TodayCard
	if err := json.Unmarshal(rec.Body.Bytes(), &cards); err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(rec.Body.String()), cards
}

// 빈 DB: 추천도 복습도 없다 → [] (null 아님).
func TestTodayCardsHandlerEmpty(t *testing.T) {
	h, _, _ := openTodayCardsHandler(t, cardsNow)
	body, cards := getTodayCards(t, h)
	if body != "[]" || len(cards) != 0 {
		t.Fatalf("body %q, want []", body)
	}
}

// 복습 카드가 만기면 복습 카드가 서고, 쉬는 날이면 숫자 없이 알린다.
func TestTodayCardsHandlerCardsAndRestDay(t *testing.T) {
	ctx := context.Background()
	h, st, userID := openTodayCardsHandler(t, cardsNow)
	// 외부 카드의 첫 복습은 며칠 뒤 → 저장 시각을 과거로 두면 지금은 만기다.
	if _, err := st.SaveExternalCard(ctx, userID, store.ExternalCard{ExternalID: "atlas:1", Question: "질문", Answer: "답"}, cardsNow.AddDate(0, 0, -30)); err != nil {
		t.Fatal(err)
	}
	_, cards := getTodayCards(t, h)
	c := cardByID(cards, "review")
	if len(cards) != 1 || c == nil || c.Count != 1 || c.Rows[0].Meta != "카드 1 · 약 1분" || c.Rows[0].Href != "/apps/punisher/" {
		t.Fatalf("cards %+v", cards)
	}

	rest := []int{int(cardsNow.Weekday())}
	if _, err := st.UpdateStudyPrefs(ctx, userID, store.StudyPrefsPatch{RestDays: rest}); err != nil {
		t.Fatal(err)
	}
	_, cards = getTodayCards(t, h)
	if c = cardByID(cards, "review"); len(cards) != 1 || c == nil || c.Rows[0].Meta != "오늘은 쉬는 날이에요" || c.Count != 0 {
		t.Fatalf("rest day cards %+v", cards)
	}
}
