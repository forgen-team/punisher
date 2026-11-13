package prompts

import (
	"fmt"
	"strings"
)

// InquiryVersion tags explanations generated with InquiryTask (stored per answer).
const InquiryVersion = "punisher-inquiry-2026-11-13.3"

// InquiryContext is the input of one "이 질문에 답하기" explanation.
type InquiryContext struct {
	Question string
	// Catalog: concepts on the user's map (for term links and related concepts).
	Catalog []CatalogRef
	// Learned: concepts the user already completed/knows — link and remind, don't re-teach.
	Learned []ConceptRef
	// Revisit: completed concepts the user was weak on in checks (never also in Learned).
	Revisit []ConceptRef
	// Misconceptions: the user's latest open acknowledged misconceptions.
	Misconceptions []Misconception
	Areas          []string
	// Revisions: problems of the previous output to fix (rewrite).
	Revisions []string
}

// InquiryTask is the fixed instruction (the system prompt is the teacher block,
// so the lecture quality rules — define on first use, no assumed knowledge — apply).
const InquiryTask = `
## 작업
수강생이 실무에서 겪은 상황을 질문했습니다. 개념 하나를 가르치는 강의가 아니라, 이 질문의 상황이 왜 일어나는지를
강의처럼 자세히 풀어 주는 해설을 씁니다. 증상과 원인을 잇는 인과를 단계별로 보여 주고, 무엇을 확인하면 진짜 원인을
알 수 있는지, 어떻게 대처하는지까지 씁니다.

## 섹션(반드시 이 5개, 이 순서)
- summary: 한두 문단으로 먼저 답합니다 — 가장 가능성 높은 원인과, 증상이 왜 그렇게 보이는지.
- causes: 원인 사슬을 단계별로(1 → 2 → 3 …) 씁니다. 각 단계에서 무엇이 어떤 순서로 일어나는지, 왜 그 단계가 다음 증상을 만드는지.
  여기에는 일반적으로 성립하는 원리만 씁니다. 공식 문서를 @@sources 로 최소 1개 인용합니다.
- assumptions: 질문만으로는 알 수 없어 수강생 환경을 추정한 부분만 모읍니다(예: 앞단에 프록시·인그레스가 있다, 헬스체크가 DB를 확인하지 않는다).
  항목마다 "추정:"으로 시작하고, 그 추정이 틀리면 결론이 어떻게 달라지는지 한 줄을 붙입니다. 추정이 없으면 "추정 없음"과 이유 한 줄.
- verify: 진짜 원인을 확인하는 방법 — 무엇을(로그·지표·명령·응답 헤더) 보면 어떤 결과일 때 어느 원인인지. 명령·로그는 @@code / @@log 로.
- fix: 당장의 대처와 근본 대책, 각각의 트레이드오프.

근거와 해석을 섞지 않습니다. causes·verify·fix 에서 환경을 가정해야 하면 그 자리에 "(추정 — 아래 '내 환경 추정' 참고)"라고만 표시하고 내용은 assumptions 에 둡니다.

## 출력 형식(엄격)
아래 마크업만 출력합니다. JSON·코드펜스로 감싸지 말고, 지시 줄은 반드시 줄 맨 앞 "@@"로 시작합니다.

@@lecture
title: 질문을 한 줄로 다시 쓴 제목(40자 이내)
concepts: 이 해설과 가장 관련 깊은 지도 개념 id 2~6개(쉼표, 아래 목록의 id만)
new_concept: 영역접두어.slug | 이름 | 영역 id | 가장 가까운 선행 개념 id | 한 줄 요약
@@section summary title=섹션 제목
@@markdown
본문(마크다운)
@@section causes title=섹션 제목
@@markdown
...
@@mermaid
sequenceDiagram
  A->>B: 요청
@@sources
- [문서 제목](https://공식-문서-URL)
@@section assumptions title=섹션 제목
@@markdown
- 추정: ...
@@section verify title=섹션 제목
@@markdown
...
@@code lang=bash
명령
@@section fix title=섹션 제목
@@markdown
...
@@end

규칙:
- new_concept 줄은 지도에 없지만 이 해설의 핵심인 개념만, 최대 3줄입니다(없으면 쓰지 않음). slug 는 소문자-하이픈, 영역 id 는 아래 목록 중 하나.
- 개념 링크는 마크다운 문장 안의 [표시할 용어](term:개념id) 하나뿐입니다. 아래 목록의 id 또는 new_concept 로 제안한 id만 쓰고, 처음 나오는 곳에 한 번만 겁니다.
- 블록은 @@markdown, @@code lang=..., @@log lang=..., @@mermaid, @@misconception, @@sources 를 씁니다. @@code / @@log 는 lang 을 반드시 지정합니다.
- @@ 표기는 줄의 맨 앞에서만 씁니다. 문장 안에 @@term 같은 표기를 쓰면 화면에서 그 자리가 빈칸이 되고 검증에 실패합니다.
- 도식은 반드시 따로 @@mermaid 블록에 씁니다. @@markdown 안에 mermaid 코드펜스나 도식 문법 줄을 넣지 않고, @@mermaid 블록 안에는 코드펜스 없이 도식 문법만 씁니다.
- @@sources 는 https 공식 문서만 씁니다. 확신할 수 없는 URL 은 만들지 않습니다.
- 구체적인 숫자·기본값·버전 동작이 확실하지 않으면 지어내지 말고 "환경·버전에 따라 다르다"고 씁니다.
- 이 글은 강의가 아니라 질문에 대한 "해설"입니다. "이 강의에서는" 대신 "이 해설에서는"이라고 씁니다.
- 다른 섹션을 가리킬 때는 kind 이름(summary, verify 등)이 아니라 그 섹션의 제목이나 "아래 확인 방법"처럼 씁니다.
- 마지막 줄은 반드시 @@end 입니다.`

// Inquiry builds the user prompt of one explanation.
func Inquiry(c InquiryContext) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## 수강생 질문\n%q\n", c.Question)
	b.WriteString("\n## 지식 지도의 개념(링크·관련 개념에 쓸 수 있는 id)\n")
	if len(c.Catalog) == 0 {
		b.WriteString("- (없음)\n")
	}
	for _, r := range c.Catalog {
		fmt.Fprintf(&b, "- %s: %s (%s)\n", r.ID, r.Name, r.Area)
	}
	b.WriteString("\n## 수강생이 이미 배운 개념(다시 가르치지 말고 첫 등장에 링크 + 한 줄 상기)\n")
	writeRefs(&b, c.Learned)
	writeRevisit(&b, c.Revisit)
	writeMisconceptions(&b, c.Misconceptions)
	b.WriteString("\n## 영역 id\n" + strings.Join(c.Areas, ", ") + "\n")
	b.WriteString(InquiryTask)
	writeFactualRules(&b)
	if len(c.Revisions) > 0 {
		b.WriteString("\n\n## 직전 출력의 문제(모두 고쳐서 처음부터 다시 출력)\n")
		for _, p := range c.Revisions {
			b.WriteString("- " + p + "\n")
		}
	}
	return b.String()
}

// FollowupQA is one earlier question and answer of the thread.
type FollowupQA struct {
	Question string
	Answer   string
}

// FollowupContext is the input of one follow-up answer.
type FollowupContext struct {
	Question string
	// Quote: the passage of the explanation the question is about ('여기 더 자세히').
	Quote string
	// Original: the thread's first question; Sections: its explanation as text.
	Original string
	Sections []string
	Previous []FollowupQA
	Catalog  []CatalogRef
	Learned  []ConceptRef
	Revisit  []ConceptRef
	// Misconceptions: the user's latest open acknowledged misconceptions.
	Misconceptions []Misconception
	Areas          []string
	// Revisions: problems of the previous output to fix (rewrite).
	Revisions []string
}

// Context budgets (runes): a long thread keeps recent turns whole and shrinks the rest.
const (
	followupSectionRunes = 2500
	followupRecentTurns  = 3
	followupRecentRunes  = 2000
	followupOlderRunes   = 150
)

// FollowupTask is the fixed instruction of a follow-up answer.
const FollowupTask = `
## 작업
위 해설을 읽은 수강생이 이어서 질문했습니다. 이 질문에만 답하는 짧은 해설을 씁니다. 해설에 이미 쓴 내용은 반복하지 말고 필요한 만큼만 짚은 뒤,
무엇이 어떤 순서로 일어나는지 인과로 설명합니다. 수강생 환경을 추정해야 하면 그 문장 앞에 "추정:"을 붙여 일반 원리와 구분합니다.

## 출력 형식(엄격)
아래 마크업만 출력합니다. JSON·코드펜스로 감싸지 말고, 지시 줄은 반드시 줄 맨 앞 "@@"로 시작합니다.

@@lecture
title: 이 답의 제목(40자 이내)
concepts: 이 답에서 새로 중요해진 지도 개념 id 0~3개(쉼표, 아래 목록의 id만, 없으면 비움)
new_concept: 영역접두어.slug | 이름 | 영역 id | 가장 가까운 선행 개념 id | 한 줄 요약
@@section answer title=이 답의 제목
@@markdown
본문(마크다운)
@@end

규칙:
- new_concept 줄은 지도에 없지만 이 답의 핵심인 개념만, 최대 2줄입니다(없으면 쓰지 않음).
- 개념 링크는 [표시할 용어](term:개념id) 하나뿐이고, 아래 목록의 id 또는 new_concept 로 제안한 id만 씁니다.
- 필요하면 @@code lang=..., @@log lang=..., @@mermaid, @@sources 블록을 씁니다(도식은 @@mermaid 블록에만, 코드펜스 없이). @@sources 는 https 공식 문서만.
- @@ 표기는 줄의 맨 앞에서만 씁니다. 문장 안에 @@term 같은 표기를 쓰면 화면에서 그 자리가 빈칸이 되고 검증에 실패합니다.
- 처음 나오는 전문 용어는 그 문장에서 한 줄로 풉니다. 확실하지 않은 숫자·기본값은 지어내지 않습니다.
- 이 글은 강의가 아니라 해설입니다. 다른 섹션은 kind 이름이 아니라 제목으로 가리킵니다.
- 마지막 줄은 반드시 @@end 입니다.`

func clipRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "…(줄임)"
}

// InquiryFollowup builds the user prompt of a follow-up answer. Earlier turns
// are shrunk: the last few stay whole, older ones keep the question and the
// start of the answer.
func InquiryFollowup(c FollowupContext) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## 처음 질문\n%q\n\n## 처음 해설\n", c.Original)
	for _, s := range c.Sections {
		b.WriteString(clipRunes(s, followupSectionRunes) + "\n\n")
	}
	if len(c.Previous) > 0 {
		b.WriteString("## 이전 문답(순서대로)\n")
		for i, qa := range c.Previous {
			limit := followupOlderRunes
			if i >= len(c.Previous)-followupRecentTurns {
				limit = followupRecentRunes
			}
			fmt.Fprintf(&b, "Q: %s\nA: %s\n\n", qa.Question, clipRunes(qa.Answer, limit))
		}
	}
	b.WriteString("## 이번 질문\n")
	if c.Quote != "" {
		fmt.Fprintf(&b, "해설에서 고른 구절: %q\n", c.Quote)
	}
	fmt.Fprintf(&b, "%q\n", c.Question)
	b.WriteString("\n## 지식 지도의 개념(링크·관련 개념에 쓸 수 있는 id)\n")
	for _, r := range c.Catalog {
		fmt.Fprintf(&b, "- %s: %s (%s)\n", r.ID, r.Name, r.Area)
	}
	b.WriteString("\n## 수강생이 이미 배운 개념\n")
	writeRefs(&b, c.Learned)
	writeRevisit(&b, c.Revisit)
	writeMisconceptions(&b, c.Misconceptions)
	b.WriteString("\n## 영역 id\n" + strings.Join(c.Areas, ", ") + "\n")
	b.WriteString(FollowupTask)
	writeFactualRules(&b)
	if len(c.Revisions) > 0 {
		b.WriteString("\n\n## 직전 출력의 문제(모두 고쳐서 처음부터 다시 출력)\n")
		for _, p := range c.Revisions {
			b.WriteString("- " + p + "\n")
		}
	}
	return b.String()
}
