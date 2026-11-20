package learning

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Next-chapter trigger and suggestion ranking (pure; no I/O).

// ChapterRunwayDays: offer the next chapter when what is left to learn fits in
// fewer than this many days at the user's daily pace.
const ChapterRunwayDays = 14

// DefaultDailyMinutes is the daily pace when the user has not set one.
const DefaultDailyMinutes = 30

// SessionMinutesPerLecture: one lecture's study session (lecture ~20 + prediction
// + check + review ≈ 30 minutes, quality design v2.2 Q3) — the runway unit.
const SessionMinutesPerLecture = 30

// RemainingMinutes estimates what is left to learn: one study session
// (SessionMinutesPerLecture) per depth that still has a pending objective, over
// concepts that are learnable (not completed/known/dismissed, not merely proposed).
func RemainingMinutes(concepts map[string]TodayConcept) int {
	total := 0
	for _, c := range concepts {
		if c.Proposed || c.Dismissed || IsDone(NormalizeState(c.State)) {
			continue
		}
		total += c.PendingDepths * SessionMinutesPerLecture
	}
	return total
}

// NextChapterPrompt is shown on the today screen when the runway is short.
type NextChapterPrompt struct {
	RemainingMinutes int `json:"remainingMinutes"`
	ThresholdMinutes int `json:"thresholdMinutes"`
	DailyMinutes     int `json:"dailyMinutes"`
}

// CheckNextChapter returns the prompt when remaining < daily × ChapterRunwayDays.
func CheckNextChapter(remaining, dailyMinutes int) *NextChapterPrompt {
	if dailyMinutes <= 0 {
		dailyMinutes = DefaultDailyMinutes
	}
	threshold := dailyMinutes * ChapterRunwayDays
	if remaining >= threshold {
		return nil
	}
	return &NextChapterPrompt{RemainingMinutes: remaining, ThresholdMinutes: threshold, DailyMinutes: dailyMinutes}
}

// Suggestion kinds (sources).
const (
	SuggestProposed = "proposed" // (a) proposed concepts that keep coming up
	SuggestDeepen   = "deepen"   // (b) deeper objectives of finished concepts
	SuggestTrend    = "trend"    // (c) RSS rising topics
	SuggestReading  = "reading"  // (c) concepts in articles the user opened, not on the map
	SuggestInterest = "interest" // (c) RSS interests
	SuggestNewArea  = "new_area" // (d) areas outside the curriculum
)

// ProposedSignal is one of the user's proposed concepts and how often it came up.
type ProposedSignal struct {
	ID       string
	Name     string
	Area     string
	Mentions int
}

// DeepenSignal is a finished concept with deeper objectives left.
type DeepenSignal struct {
	ConceptID string
	Name      string
	Area      string
	Remaining int // pending objectives at depth 2-3
	Mentions  int // e.g. how often RSS explanations pointed to it
	// ReadMentions: articles the user opened recently that were classified to it.
	ReadMentions int
}

// ReadTopicSignal is a concept that keeps appearing in opened articles but is not on the map.
type ReadTopicSignal struct {
	Name  string
	Count int
}

// TrendSignal is an RSS rising topic.
type TrendSignal struct {
	Topic string
	Score float64 // z value
}

// NewArea is a topic outside the seed curriculum.
type NewArea struct {
	Key      string
	Title    string
	Pitch    string
	TitleEn  string
	PitchEn  string
	Keywords []string
}

// SuggestionInput is everything RankSuggestions needs.
type SuggestionInput struct {
	Proposed  []ProposedSignal
	Deepen    []DeepenSignal
	Interests []string
	Trending  []TrendSignal
	// ReadTopics stays empty without RSS (standalone build).
	ReadTopics []ReadTopicSignal
	NewAreas   []NewArea
	// Covered: normalized names of concepts already on the user's map (skip as topics).
	Covered map[string]bool
	// Locale: ko|en of titles and reasons ("" = ko).
	Locale string
}

// Suggestion is one card of "다음엔 어떤 걸 공부해 보고 싶어요?".
type Suggestion struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	Reason     string   `json:"reason"`
	Score      float64  `json:"score"`
	ConceptIDs []string `json:"conceptIds,omitempty"`
}

const maxPerKind = 2

// RankSuggestions scores every candidate, keeps at most maxPerKind per kind
// (diversity) and returns the best `limit`. Signals matching an RSS interest
// or rising topic get a boost.
func RankSuggestions(in SuggestionInput, limit int) []Suggestion {
	hot := append([]string{}, in.Interests...)
	for _, t := range in.Trending {
		hot = append(hot, t.Topic)
	}
	var all []Suggestion
	loc := in.Locale
	for _, p := range in.Proposed {
		score := 2 + math.Min(float64(p.Mentions), 5)*0.5
		reason := fmt.Sprintf(inLocale(loc, "강의·질문에서 %d번 나왔지만 아직 지도에 없는 개념이에요.",
			"Came up %d times in lectures and questions but is not on your map yet."), p.Mentions)
		if matchesAny(p.Name, hot) {
			score += 1.5
			reason = fmt.Sprintf(inLocale(loc, "강의·질문에서 %d번 나왔고 요즘 읽는 글의 관심사와도 겹쳐요.",
				"Came up %d times in lectures and questions and matches what you have been reading."), p.Mentions)
		}
		all = append(all, Suggestion{ID: SuggestProposed + ":" + p.ID, Kind: SuggestProposed, Title: p.Name,
			Reason: reason, Score: score, ConceptIDs: []string{p.ID}})
	}
	for _, d := range in.Deepen {
		score := 1.5 + math.Min(float64(d.Remaining), 6)*0.25 + math.Min(float64(d.Mentions), 4)*0.3 + math.Min(float64(d.ReadMentions), 4)*0.3
		reason := fmt.Sprintf(inLocale(loc, "기본은 마쳤고 심화 목표 %d개가 남았어요.",
			"You finished the basics; %d advanced objectives remain."), d.Remaining)
		if d.Mentions > 0 {
			reason = fmt.Sprintf(inLocale(loc, "기본은 마쳤고 심화 목표 %d개가 남았어요. 최근 글 해설에서도 %d번 이어졌어요.",
				"You finished the basics; %d advanced objectives remain. Recent article explanations led here %d times."), d.Remaining, d.Mentions)
		}
		if d.ReadMentions > 0 {
			reason += fmt.Sprintf(inLocale(loc, " 최근 연 글 %d건에도 나왔어요.", " It also appeared in %d articles you opened recently."), d.ReadMentions)
		}
		title := d.Name + inLocale(loc, " 심화", " (advanced)")
		all = append(all, Suggestion{ID: SuggestDeepen + ":" + d.ConceptID, Kind: SuggestDeepen, Title: title,
			Reason: reason, Score: score, ConceptIDs: []string{d.ConceptID}})
	}
	trendTopics := map[string]bool{}
	for _, t := range in.Trending {
		if covered(t.Topic, in.Covered) {
			continue
		}
		trendTopics[normalize(t.Topic)] = true
		all = append(all, Suggestion{ID: SuggestTrend + ":" + t.Topic, Kind: SuggestTrend, Title: t.Topic,
			Reason: inLocale(loc, "RSS 흐름에서 최근 크게 늘어난 주제예요.", "A topic rising sharply in your RSS feeds."), Score: 1 + math.Min(math.Max(t.Score, 0), 5)*0.4})
	}
	for _, t := range in.ReadTopics {
		if covered(t.Name, in.Covered) || trendTopics[normalize(t.Name)] {
			continue
		}
		trendTopics[normalize(t.Name)] = true // 관심 분야와 겹치면 이 카드 하나만
		reason := fmt.Sprintf(inLocale(loc, "최근 연 글 %d건에 나왔지만 아직 지도에 없는 주제예요.",
			"Appeared in %d articles you opened recently but is not on your map yet."), t.Count)
		if matchesExact(t.Name, in.Interests) {
			reason += inLocale(loc, " 등록한 관심 분야이기도 해요.", " It is also one of your interests.")
		}
		// ID 는 대소문자 변형이 있어도 같게(초안을 만들 때 제안을 다시 계산해 ID 로 찾는다).
		all = append(all, Suggestion{ID: SuggestReading + ":" + normalize(t.Name), Kind: SuggestReading, Title: t.Name,
			Reason: reason, Score: 1.3 + math.Min(float64(t.Count), 5)*0.3})
	}
	for _, name := range in.Interests {
		if covered(name, in.Covered) || trendTopics[normalize(name)] {
			continue
		}
		all = append(all, Suggestion{ID: SuggestInterest + ":" + name, Kind: SuggestInterest, Title: name,
			Reason: inLocale(loc, "RSS 관심 분야로 등록해 둔 주제예요.", "A topic you registered as an RSS interest."), Score: 1.2})
	}
	for _, a := range in.NewAreas {
		pitch, areaTitle := a.Pitch, a.Title
		if loc == LocaleEN && a.PitchEn != "" {
			pitch, areaTitle = a.PitchEn, a.TitleEn
		}
		score, reason := 0.8, pitch
		for _, k := range a.Keywords {
			if matchesAny(k, hot) {
				score += 1.5
				reason = pitch + inLocale(loc, " 요즘 읽는 글과도 이어져요.", " It also connects to what you have been reading.")
				break
			}
		}
		all = append(all, Suggestion{ID: SuggestNewArea + ":" + a.Key, Kind: SuggestNewArea, Title: areaTitle,
			Reason: reason, Score: score})
	}

	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Score != all[j].Score {
			return all[i].Score > all[j].Score
		}
		return all[i].ID < all[j].ID
	})
	// First pass keeps at most maxPerKind per kind; when that leaves slots
	// empty (few sources), a second pass fills them in score order.
	perKind := map[string]int{}
	picked := map[string]bool{}
	out := []Suggestion{}
	for pass := 0; pass < 2; pass++ {
		for _, s := range all {
			if len(out) == limit {
				break
			}
			if picked[s.ID] || (pass == 0 && perKind[s.Kind] == maxPerKind) {
				continue
			}
			perKind[s.Kind]++
			picked[s.ID] = true
			s.Score = math.Round(s.Score*100) / 100
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

const (
	maxReadTopicRunes = 60
	maxReadTopicWords = 6
)

// CleanReadTopicName validates a concept name extracted from article text: it
// ends up in the chapter-design prompt, so anything that is not a short
// technical term (markup, control characters, sentences) is rejected.
func CleanReadTopicName(raw string) (string, bool) {
	words := strings.Fields(raw)
	name := strings.Join(words, " ")
	if name == "" || len(words) > maxReadTopicWords || utf8.RuneCountInString(name) > maxReadTopicRunes || strings.Contains(name, "##") {
		return "", false
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r) && !strings.ContainsRune(" .+#-_/()·&", r) {
			return "", false
		}
	}
	return name, true
}

func matchesExact(s string, list []string) bool {
	for _, x := range list {
		if normalize(x) == normalize(s) {
			return true
		}
	}
	return false
}

func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

func covered(topic string, names map[string]bool) bool {
	return names[normalize(topic)]
}

// matchesAny: case-insensitive containment either way (e.g. "RAG" ~ "RAG 파이프라인").
func matchesAny(s string, list []string) bool {
	a := normalize(s)
	if a == "" {
		return false
	}
	for _, x := range list {
		b := normalize(x)
		if b != "" && (strings.Contains(a, b) || strings.Contains(b, a)) {
			return true
		}
	}
	return false
}

// DefaultNewAreas are topics outside the seed curriculum (source d).
var DefaultNewAreas = []NewArea{
	{Key: "llm-engineering", Title: "LLM 엔지니어링", Pitch: "프롬프트·RAG·평가처럼 LLM을 제품에 넣는 기술은 지금 지도 밖에 있어요.",
		TitleEn: "LLM engineering", PitchEn: "Building LLMs into products — prompting, RAG, evaluation — is outside your map today.",
		Keywords: []string{"llm", "rag", "ai", "프롬프트", "에이전트", "임베딩", "claude", "gpt"}},
	{Key: "web-performance", Title: "웹 성능", Pitch: "렌더링·번들·Core Web Vitals처럼 화면이 빨라지는 원리는 지도에 없는 영역이에요.",
		TitleEn: "Web performance", PitchEn: "What makes pages fast — rendering, bundles, Core Web Vitals — is an area not on your map.",
		Keywords: []string{"성능", "performance", "web vitals", "렌더링", "번들", "react"}},
	{Key: "data-pipelines", Title: "데이터 파이프라인", Pitch: "배치·스트리밍·CDC로 데이터를 옮기고 가공하는 일은 지도 밖이에요.",
		TitleEn: "Data pipelines", PitchEn: "Moving and shaping data with batch, streaming and CDC is outside your map.",
		Keywords: []string{"kafka", "스트리밍", "etl", "파이프라인", "cdc", "데이터"}},
	{Key: "testing-quality", Title: "테스트와 품질 엔지니어링", Pitch: "테스트 전략·플레이키 테스트·계약 테스트처럼 품질을 지키는 기술은 지도 밖이에요.",
		TitleEn: "Testing and quality engineering", PitchEn: "Keeping quality — test strategy, flaky tests, contract tests — is outside your map.",
		Keywords: []string{"테스트", "test", "qa", "품질", "e2e"}},
}
