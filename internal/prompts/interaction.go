package prompts

import (
	"fmt"
	"strings"
)

// SystemGloss: short term popovers (haiku).
const SystemGloss = `당신은 개발자용 용어 팝오버를 쓰는 편집자입니다. 강의를 읽던 사람이 본문 흐름을 놓치지 않도록, 용어의 핵심만 2~3문장(200자 이내)으로 설명합니다. 정의 → 왜 중요한지(실무에서 마주치는 장면) 순서로 씁니다. 확실하지 않은 수치·버전은 쓰지 않습니다. 마크다운 제목·목록 없이 평문만 출력합니다.`

// Gloss builds the user prompt for a term popover.
func Gloss(name, area, summary string) string {
	return fmt.Sprintf("용어: %s\n영역: %s\n커리큘럼 요약: %s\n\n위 용어의 팝오버 설명을 쓰세요.", name, area, summary)
}

// SystemResolve: map a free query to concept ids (haiku).
const SystemResolve = `당신은 개발자의 질문을 개념 그래프의 노드에 연결하는 분류기입니다. 반드시 제공된 후보 목록의 id만 사용합니다. 후보 중 맞는 것이 없을 때만 새 개념을 제안하고, 새 개념의 선행 개념도 후보 목록의 id만 씁니다. 출력은 JSON 하나뿐입니다.`

// ResolveCandidate is a concept shown to the resolver.
type ResolveCandidate struct {
	ID      string
	Name    string
	Area    string
	Summary string
}

// Resolve builds the concept-resolution prompt.
func Resolve(query string, candidates []ResolveCandidate, areas []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## 질문\n%q\n\n## 후보 개념(id | 이름 | 영역 | 요약)\n", query)
	for _, c := range candidates {
		fmt.Fprintf(&b, "- %s | %s | %s | %s\n", c.ID, c.Name, c.Area, truncate(c.Summary, 80))
	}
	fmt.Fprintf(&b, "\n## 영역 목록\n%s\n", strings.Join(areas, ", "))
	b.WriteString(`
## 작업
1. 질문을 이해하는 데 핵심인 개념을 후보에서 최대 3개 고르세요(관련도 높은 순).
2. 후보 중 질문의 핵심을 다루는 개념이 전혀 없을 때만 "proposed" 를 채우세요.
   - id 는 "{영역 접두}.{영문-소문자-하이픈}" 형식(예: infra.pod-eviction). 영역 접두는 기존 후보 id 의 접두를 따릅니다.
   - prerequisites 는 반드시 후보 id 중에서 1~3개.
   - objectives 는 "~를 설명한다/~를 읽는다" 같은 관찰 가능한 문장 2~4개.

JSON만 출력:
{
  "concept_ids": ["후보 id"],
  "confidence": 0.0~1.0,
  "proposed": null 또는 {"id": "...", "name": "한국어 이름", "area": "영역", "depth": 1~3,
               "summary": "한두 문장", "aliases": ["질문에 나온 표현"], "prerequisites": ["후보 id"],
               "objectives": ["..."]}
}`)
	return b.String()
}

// SystemQA: contextual Q&A during a lecture.
const SystemQA = `당신은 강의 중 질문에 답하는 조교입니다. 수강생은 경력 개발자이며 지금 특정 강의의 특정 섹션을 읽고 있습니다.
- 질문에 직접 답하고, 강의 본문과 연결해 "본문의 어느 설명과 이어지는지"를 짚습니다.
- 필요하면 짧은 코드·명령어 예시를 마크다운 코드블록으로 붙입니다.
- 확실하지 않은 내용은 추측하지 말고 확인 방법(공식 문서, 명령어)을 안내합니다.
- 답변에 강의 범위를 벗어난 개념이 등장하면 후보 목록의 id로 연결하고, 목록에 없으면 새 개념으로 제안합니다.
출력은 JSON 하나뿐입니다.`

// QAContext is the lecture context of a question.
type QAContext struct {
	ConceptID     string
	ConceptName   string
	LectureTitle  string
	SectionTitle  string
	SectionText   string
	History       []QATurn
	Linkable      []ConceptRef
	LearnedBefore []ConceptRef
	// Misconceptions: the learner's open acknowledged misconceptions on this concept and its prerequisites.
	Misconceptions []Misconception
	// ExampleLanguage: display name of the preferred example-code language.
	ExampleLanguage string
}

// QATurn is a previous question/answer in the thread.
type QATurn struct {
	Question string
	Answer   string
}

// Question builds the Q&A prompt.
func Question(c QAContext, question string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## 강의\n- 개념: %s (%s)\n- 제목: %s\n- 현재 섹션: %s\n\n### 섹션 본문\n%s\n",
		c.ConceptName, c.ConceptID, c.LectureTitle, c.SectionTitle, truncate(c.SectionText, 6000))
	if len(c.LearnedBefore) > 0 {
		b.WriteString("\n## 수강생이 이미 배운 개념\n")
		writeRefs(&b, c.LearnedBefore)
	}
	if len(c.History) > 0 {
		b.WriteString("\n## 이 강의의 이전 질문\n")
		for _, t := range c.History {
			fmt.Fprintf(&b, "Q: %s\nA: %s\n", truncate(t.Question, 400), truncate(t.Answer, 800))
		}
	}
	b.WriteString("\n## 연결 가능한 개념 id\n")
	writeRefs(&b, c.Linkable)
	writeMisconceptions(&b, c.Misconceptions)
	b.WriteString(ExampleLanguageRule(c.ExampleLanguage))
	fmt.Fprintf(&b, "\n## 질문\n%s\n", question)
	b.WriteString(`
JSON만 출력:
{"answer": "마크다운 답변", "linked_concept_ids": ["후보 id"],
 "proposed_concepts": [{"name": "그래프에 없는 개념", "reason": "왜 연결할 가치가 있는지"}]}`)
	return b.String()
}

// AltExplanation builds the "explain differently" prompt (feedback=insufficient).
func AltExplanation(c QAContext, comment string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## 강의\n- 개념: %s\n- 섹션: %s\n\n### 섹션 본문\n%s\n",
		c.ConceptName, c.SectionTitle, truncate(c.SectionText, 6000))
	if comment != "" {
		fmt.Fprintf(&b, "\n## 수강생 코멘트\n%s\n", comment)
	}
	b.WriteString(ExampleLanguageRule(c.ExampleLanguage))
	b.WriteString(`
## 작업
수강생이 이 섹션의 설명이 부족하다고 했습니다. 같은 내용을 **다른 방식**으로 다시 설명하세요.
- 본문과 다른 비유 또는 다른 실무 예제를 쓰고, 단계를 더 잘게 나눕니다.
- 필요하면 mermaid 도식이나 짧은 코드/로그를 마크다운 코드블록으로 넣습니다.
- 본문에 없던 새 사실을 단정하지 말고, 핵심 오해 한 가지를 짚어 줍니다.

JSON만 출력:
{"answer": "마크다운 재설명", "linked_concept_ids": [], "proposed_concepts": []}`)
	return b.String()
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
