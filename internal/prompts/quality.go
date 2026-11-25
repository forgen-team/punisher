package prompts

import (
	"fmt"
	"strings"
)

// Fixed (not user-editable) sections of the lecture quality work: like
// FactualRules and ReaderProfile they are appended by code after the
// editable blocks, so an edited block keeps working and still gets them.

// LectureShape is the reader-level order and exercise fading of a lecture
// (research #6 fading, #8 concrete → abstract). review adds how the reviewer
// judges the order.
func LectureShape(level string, review bool) string {
	var b strings.Builder
	b.WriteString("\n## 설명 순서와 연습(독자 수준별)\n")
	switch strings.TrimSpace(level) {
	case ReaderBeginner, ReaderNovice:
		b.WriteString("- 순서: 구체 → 추상. principles 는 장면의 숫자로 먼저 작게 손계산(@@trace)하고, 표·도식으로 정리한 뒤 이름을 붙여 정식 정의(@@keypoint kind=definition)합니다. 정의가 첫 손계산 바로 뒤에 와도 됩니다.\n")
		b.WriteString("- 연습: 처음 두 연습은 guided, 그 뒤 modify·error. free 는 마지막 하나만 씁니다.\n")
		if review {
			b.WriteString("- 감수: 이 수준에서는 정의가 첫 손계산 바로 뒤에 오는 것을 '핵심 정의 지연'으로 지적하지 않습니다(그 손계산이 정의를 바로 이끌어 낼 때).\n")
		}
	case ReaderAdvanced:
		b.WriteString("- 순서: 정의는 한 줄로 압축하고 바로 손계산·경계 사례로 갑니다.\n")
		b.WriteString("- 연습: guided 는 첫 하나만 짧게, 나머지는 free·modify·error 입니다.\n")
	default:
		b.WriteString("- 순서: 핵심 정의 → 작은 값 손계산 → 실제 규모로 확장. 핵심 정의는 principles 첫머리에 둡니다.\n")
		b.WriteString("- 연습: 첫 연습은 guided, 그 뒤 modify·error, 마지막은 error 또는 free 입니다.\n")
	}
	return b.String()
}

// RubricItems are the LLM self-assessment items of the review (research §4.2).
var RubricItems = []struct{ ID, Text string }{
	{"E01", "장면이 현실적이고 situation_question 이 원인을 밝히지 않음"},
	{"E02", "오늘의 질문이 장면의 숫자·현상에서 나오고 judgment_next 에서 같은 값으로 닫힘"},
	{"E03", "이 독자 수준으로 막힘 없이 읽힘(reader_gaps 가 적음)"},
	{"E04", "정의가 정의 + 예 + 비예이고 비유로 대신하지 않음"},
	{"E05", "손계산이 작은 값에서 일반 규칙으로 이어짐"},
	{"E06", "오개념이 실제로 할 법하고 반례가 계산 가능함"},
	{"E07", "연습이 목표와 맞고 풀이가 단계적이며 숫자만 바꾼 수준을 넘음"},
	{"E08", "처방 후 같은 장애가 사라지는지 단계별로 추적됨"},
	{"E09", "판단 기준에 조건·한계가 있고 과장이 없음"},
	{"E10", "학습 목표마다 피상적이지 않게 다룸"},
	{"E11", "난이도·밀도가 독자 수준에 맞음"},
	{"E12", "한 문단의 새 개념이 적고 정의 직후 바로 적용함"},
	{"E13", "코드 출력·명령 옵션·기본값이 정확함"},
}

// RubricRules asks the review for 1~4 scores with quoted evidence. The scores
// are stored for trends and never gate publishing.
func RubricRules() string {
	var b strings.Builder
	b.WriteString("\n## 품질 점수(추세 기록용 — 발행 판정에 쓰지 않음)\n")
	b.WriteString("같은 JSON 에 \"rubric\" 배열을 더합니다. 항목마다 score 1~4(1 미흡 · 2 부분 · 3 충분 · 4 모범)와 그 판단의 근거가 되는 본문 문장을 quote 에 그대로 인용합니다. 인용이 없거나 본문에 없는 문장이면 그 점수는 버립니다.\n")
	b.WriteString("\"rubric\": [{\"id\": \"E01\", \"score\": 3, \"quote\": \"본문 문장 그대로\", \"why\": \"한 줄\"}]\n")
	for _, it := range RubricItems {
		fmt.Fprintf(&b, "- %s %s\n", it.ID, it.Text)
	}
	return b.String()
}

// ReviewEvidence is what the app computed for the review: in-memory SQL runs,
// recomputed arithmetic, independent solutions, broken links (one line each).
type ReviewEvidence struct {
	SQL         []string
	Arithmetic  []string
	Independent []string
	Links       []string
}

func (e ReviewEvidence) empty() bool {
	return len(e.SQL)+len(e.Arithmetic)+len(e.Independent)+len(e.Links) == 0
}

// writeEvidence appends the evidence section (nothing when there is none).
func writeEvidence(b *strings.Builder, e ReviewEvidence) {
	if e.empty() {
		return
	}
	b.WriteString("\n## 앱이 계산한 근거\n")
	section := func(title string, lines []string) {
		if len(lines) == 0 {
			return
		}
		b.WriteString("### " + title + "\n")
		for _, l := range lines {
			b.WriteString("- " + l + "\n")
		}
	}
	section("SQL 예시를 SQLite 메모리 DB 에서 실행한 결과", e.SQL)
	section("본문의 식을 다시 계산한 결과", e.Arithmetic)
	section("초안의 풀이·출력을 보지 않고 따로 푼 답", e.Independent)
	section("열리지 않는 출처 링크", e.Links)
	b.WriteString("판정: 본문과 위 근거가 다르면 어느 쪽이 맞는지 직접 단계별로 확인합니다. 본문이 틀렸으면 factual_error 로 지적합니다. SQLite 와 다른 DB 의 차이나 환경 차이로 설명되고 본문에 그 조건이 있으면 advisory 입니다. 따로 푼 답이 틀렸으면 지적하지 않습니다.\n")
}

// SystemRecompute is the system prompt of the independent solving pass
// (research #11, factored verification): the solver sees the problems, never
// the draft's solutions or outputs.
const SystemRecompute = `당신은 강의의 연습 문제와 출력 예측 질문을 처음 보는 풀이자입니다. 주어진 문제와 코드만 보고 스스로 단계별로 풀어 최종 답을 짧게 씁니다. 풀이 과정은 쓰지 않습니다. 실행 환경·버전에 따라 달라 정할 수 없으면 answer 에 "정할 수 없음"과 이유 한 구절을 씁니다.
출력은 JSON 하나뿐입니다: {"answers":[{"id":"문제 id","answer":"최종 답 한두 줄"}]}`

// RecomputeItem is one problem for the independent pass: an exercise (its
// text, with the preceding code or trace as context) or a pause question with the code it asks about.
type RecomputeItem struct {
	ID, Kind, Question, Context string
}

// Recompute builds the user prompt of the independent pass.
func Recompute(conceptName string, items []RecomputeItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## 강의 개념\n%s\n\n## 문제\n", conceptName)
	for _, it := range items {
		fmt.Fprintf(&b, "\n### %s (%s)\n%s\n", it.ID, it.Kind, strings.TrimSpace(it.Question))
		if strings.TrimSpace(it.Context) != "" {
			fmt.Fprintf(&b, "<<<CONTEXT\n%s\nCONTEXT>>>\n", strings.TrimSpace(it.Context))
		}
	}
	b.WriteString("\n문제마다 id 를 그대로 써서 답하세요.\n")
	writeFactualRules(&b)
	return b.String()
}

// OutlineLiteTask is the short plan of an instant lecture (research #13): the
// thesis check, key definitions, misconceptions and scene numbers only.
const OutlineLiteTask = `
## 작업
본문을 쓰기 전에 짧은 계획만 세웁니다(본문 문장 금지, 1,500 토큰 이내). JSON만 출력하세요:
{
  "thesis": "강의 전체의 논지 한 문장",
  "thesis_check": [{"subtopic": "하위 주제", "holds": true, "why": "이 하위 주제에서도 논지가 참인 이유"}],
  "key_definitions": [{"term": "목표의 핵심 명사", "definition": "정식 정의 한 문장", "example": "예", "non_example": "비예"}],
  "misconceptions": [{"statement": "학습자가 할 법한 문장", "counterexample": "계산 가능한 반례", "after_definition": "바로 뒤에 둘 정의 term"}],
  "scene_numbers": ["첫 장면의 숫자·현상(judgment_next 에서 다시 계산해 닫을 값)"]
}
규칙:
- thesis_check 에 holds=false 가 있으면 논지를 고쳐 모든 하위 주제에서 참이 되게 한 뒤 출력합니다.
- key_definitions 는 목표의 핵심 명사 전부, misconceptions 는 최대 2개입니다.`

// OutlineLite asks for the short plan of an instant lecture.
func OutlineLite(c LectureContext) string {
	var b strings.Builder
	writeContext(&b, c)
	b.WriteString(OutlineLiteTask)
	writeReaderProfile(&b, c.Reader)
	writeDepthScope(&b, c.Depth)
	writeFactualRules(&b)
	return b.String()
}
