// Package prompts holds the Korean LLM prompts for Punisher. Prompts are
// versioned (Version) and the version is stored on every generated lecture.
package prompts

import (
	"fmt"
	"strings"
)

// Version is recorded in lectures.prompt_version.
const Version = "punisher-prompts-2026-11-09.3"

// ConceptRef is a concept the prompt may link to.
type ConceptRef struct {
	ID   string
	Name string
}

// CatalogRef is a knowledge-map concept listed for inline linking.
type CatalogRef struct {
	ID      string
	Name    string
	Area    string
	Learned bool
}

// ObjectiveRef is a learning objective to cover.
type ObjectiveRef struct {
	ID   string
	Text string
}

// DocLink is an official documentation link from the curriculum.
type DocLink struct {
	Title string
	URL   string
}

// LectureContext is everything a lecture prompt needs about the target.
type LectureContext struct {
	ConceptID      string
	ConceptName    string
	Area           string
	Depth          int
	Summary        string
	PracticalCases []string
	DocLinks       []DocLink
	Objectives     []ObjectiveRef
	// Learned: prerequisites/related concepts the learner already completed — link, don't re-teach.
	Learned []ConceptRef
	// Unlearned: prerequisites the learner has NOT completed — must be bridged/defined where used.
	Unlearned []ConceptRef
	// Catalog: every other concept of the knowledge map (link targets for names
	// the lecture mentions in passing, instead of "다른 강의에서 다룹니다").
	Catalog []CatalogRef
	// Linkable: existing concept ids that term blocks may reference.
	Linkable []ConceptRef
	// StyleHint: adjustment from feedback (too_easy / too_hard / insufficient ...).
	StyleHint string
	// Query: the learner's original question for instant lectures (may be empty).
	Query string
	// ExampleLanguage: display name of the learner's preferred example-code language ("" = no preference).
	ExampleLanguage string
}

// SystemTeacher is the shared system prompt for outline/lecture generation.
const SystemTeacher = `당신은 "Punisher"의 강사입니다. 수강생은 실무 경력이 있는 개발자지만 CS·운영 기반 지식(OS, 네트워크, DB, 분산 시스템, 인프라, 보안)에 공백이 있습니다. 매일 약 30분, 과제·채점 없이 "좋은 강의 한 편"을 읽습니다.

원칙:
1. 실무 상황에서 출발합니다. 첫 문단은 수강생이 실제로 겪었을 법한 장면(장애, 느린 쿼리, 이상한 로그)이고, 그 장면이 오늘의 질문으로 이어집니다.
2. 메커니즘을 단계별로 설명합니다. "그냥 그렇다"가 아니라 무엇이 어떤 순서로 일어나는지, 왜 그렇게 설계됐는지 인과를 보여줍니다.
3. 코드·명령어·로그·도식으로 직접 보여줍니다. 출력 예시는 실제 도구가 내는 형태를 따르되, 구체적인 숫자·버전·플래그가 확실하지 않으면 지어내지 말고 "환경에 따라 다르다"고 쓰거나 생략합니다.
4. 흔한 오해를 짚습니다. 경력자가 흔히 갖는 잘못된 직관을 명시하고 왜 틀렸는지 설명합니다.
5. 판단 기준과 한계를 줍니다. 언제 이 지식이 결정을 바꾸는지, 어디까지가 일반론이고 어디부터 구현·버전에 따라 다른지 구분합니다.
6. 공식 문서를 근거로 인용합니다(언어/OS man page, RFC, 공식 레퍼런스, 공식 엔지니어링 문서). 블로그·Q&A 사이트는 근거로 쓰지 않습니다. 확신할 수 없는 URL은 만들지 말고, 제공된 공식 문서 링크를 우선 사용합니다.
7. 이미 배운 개념은 다시 가르치지 않습니다. 첫 등장에서 용어 링크([용어](term:개념id))를 걸고 한 줄로 상기시킨 뒤 넘어갑니다.
8. 존댓말로, 군더더기 없이, 정확하게 씁니다. 과장("완벽하게", "항상", "절대")을 피하고 조건을 명시합니다.
9. 전제하지 않습니다. 수강생이 안다고 가정해도 되는 것은 웹 FE/BE 실무 범위(HTTP 요청·응답과 상태 코드, JSON, JS/TS 코드, SQL 기본 CRUD, git, cd·ls·cat·grep 정도의 셸)뿐입니다. OS·네트워크 스택·DB 내부·분산 시스템·인프라·보안의 용어, 약어, 도구 이름(예: systemd, 시그널, 서브넷, NAT, SYN, 격리 수준, 락, 시퀀스, SLO)은 모른다고 가정합니다.
10. 첫 등장에서 정의합니다. 위 범위 밖의 용어·약어·도구·명령 옵션이 처음 나오는 문장(또는 바로 다음 문장)에서 한 줄로 뜻을 풀어 씁니다(괄호 정의 가능). 지식 지도에 있는 개념이면 그 첫 등장에 링크도 겁니다. 정의를 뒤 섹션이나 용어 블록에 미루지 않습니다. 표·목록·로그 해설·판단 섹션 안의 약어도 같습니다.
11. 선행 메커니즘에는 다리를 놓습니다. 문장을 이해하려면 아직 배우지 않은 동작 원리(예: "감독자가 죽은 자식을 다시 띄운다", "커밋 전에는 다른 세션이 못 본다")가 필요하면, 그 자리에서 2~3문장으로 무엇이 어떤 순서로 일어나는지 설명하고 자세한 내용은 링크로 넘깁니다.
12. "당연히", "알다시피", "물론", "익히 아는", "잘 알려진", "기본이므로" 같은 전제 표현을 쓰지 않습니다. 이름만 던지는 "다른 강의에서 다룹니다"도 쓰지 않습니다 — 쓰려면 한 줄 정의나 링크를 함께 둡니다.`

// OutlineTask is the default fixed instruction of the outline step (block outline.task).
const OutlineTask = `
## 작업
위 개념으로 30분 강의의 개요를 설계하세요. 섹션은 반드시 다음 4개, 이 순서입니다.
- situation_question (약 3분): 실무 상황과 오늘의 질문
- principles (약 12분): 배경·원리를 단계별로
- walkthrough (약 10분): 코드·로그·도식 해설과 대안 비교
- judgment_next (약 5분): 판단 기준·한계·다음에 이어질 개념(새 개념은 이름만 던지지 말고 한 줄 정의 또는 링크)

JSON만 출력하세요(설명 문장 금지):
{
  "title": "강의 제목(질문형 또는 상황형, 40자 이내)",
  "new_scope": ["이번 강의에서 새로 다루는 범위(이미 배운 개념과 구분)"],
  "objective_ids": ["다룰 목표 id — 제공된 목표 전부"],
  "prerequisites_referenced": ["링크로만 참조할 기존 개념 id"],
  "sections": [
    {"kind": "situation_question", "title": "...", "minutes": 3, "key_points": ["..."], "planned_blocks": ["markdown", "log"]},
    ...
  ],
  "misconceptions": ["다룰 흔한 오해"],
  "sources": [{"title": "...", "url": "https://..."}],
  "uncertain_points": ["확신이 없어 본문에서 단정하지 않을 내용"],
  "terms_to_define": [{"term": "본문에 나올 용어·약어·도구", "where": "처음 나올 섹션", "how": "한 줄 정의 | 2~3문장 다리 | 링크(개념 id)"}]
}
분 합계는 25~35분이어야 합니다.`

// Outline asks for a JSON outline of the lecture.
func Outline(c LectureContext, set Set) string {
	var b strings.Builder
	writeContext(&b, c)
	b.WriteString(set.Text(BlockOutlineTask, c.vars()))
	return b.String()
}

// Lecture asks for the full lecture in markup. outlineJSON may be empty
// (instant lectures). revisionNotes, when set, lists reviewer/validator
// issues the regeneration must fix.
func Lecture(c LectureContext, outlineJSON, revisionNotes string, set Set) string {
	var b strings.Builder
	writeContext(&b, c)
	if outlineJSON != "" {
		b.WriteString("\n## 확정된 개요\n```json\n" + outlineJSON + "\n```\n")
	}
	if revisionNotes != "" {
		b.WriteString("\n## 이전 초안에 대한 검수 지적(반드시 모두 고칠 것)\n" + revisionNotes + "\n")
	}
	b.WriteString(set.Text(BlockLectureFormat, c.vars()))
	return b.String()
}

// MarkupSpec is the output-format contract for lecture bodies.
const MarkupSpec = `
## 출력 형식(엄격)
아래 마크업만 출력합니다. JSON·코드펜스로 감싸지 말고, 지시 줄은 반드시 줄 맨 앞 "@@"로 시작합니다.

@@lecture
title: 강의 제목
objectives: 목표id1, 목표id2        ← 제공된 목표 id를 모두 포함
new_scope: 이번에 새로 다루는 범위 한 줄 (여러 줄 가능)
prerequisites: 링크로만 참조한 기존 개념 id들(쉼표 구분, 없으면 비움)
@@section situation_question minutes=3 title=섹션 제목
@@markdown
본문(마크다운). 코드는 여기 넣지 말고 @@code 블록을 씁니다.
처음 나오는 개념은 문장 안에서 [프로세스](term:os.process)처럼 링크하고, 같은 문장에서 한 줄로 뜻을 풀어 씁니다.
@@section principles minutes=12 title=섹션 제목
@@markdown
...
@@mermaid
flowchart LR
  A --> B
@@misconception
흔한 오해와 바로잡기
@@term concept=새개념-slug proposed term=그래프에 없는 용어
한두 문장 배경 설명(그 용어가 처음 나온 문단 바로 다음에)
@@sources
- [문서 제목](https://공식-문서-URL)
@@section walkthrough minutes=10 title=섹션 제목
@@code lang=bash
명령어나 코드
@@log lang=text
실제 도구 출력 형태의 로그
@@sources
- [문서 제목](https://...)
@@section judgment_next minutes=5 title=섹션 제목
@@markdown
판단 기준, 한계, 다음에 배울 개념([개념 이름](term:개념id) — 왜 이어지는지 한 줄씩)
@@end

규칙:
- 섹션은 정확히 4개, 위 순서. minutes 합계 25~35.
- principles 와 walkthrough 에는 각각 @@sources 블록이 최소 1개 있어야 하며 https 공식 문서만 씁니다.
- @@code / @@log 는 lang 을 반드시 지정합니다(bash, go, sql, yaml, text 등).
- 도식(sequenceDiagram, flowchart, graph TD 등)은 반드시 따로 @@mermaid 블록에 씁니다. @@markdown 안에 mermaid 코드펜스나 도식 문법 줄을 넣으면 화면에 그림이 아니라 코드로 보입니다. @@mermaid 블록 안에는 코드펜스 없이 도식 문법만 씁니다.
- 개념 링크는 마크다운 문장 안의 [표시할 용어](term:개념id) 하나뿐입니다. 개념id 는 "연결 가능한 기존 개념"·"아직 배우지 않은 선행 개념"·"지식 지도의 다른 개념" 목록의 id만 쓰고, 용어가 처음 나오는 곳에 한 번만 겁니다. 문장 중간에 "@@term"을 쓰지 않습니다(@@ 는 줄 맨 앞 지시어 전용).
- @@term 블록은 목록에 없는 새 개념을 제안할 때만 씁니다(proposed 와 새 slug, 영역.이름 형식, 소문자-하이픈). 섹션 끝에 몰지 말고 그 용어가 처음 나온 문단 바로 다음에 둡니다.
- 명령어 예시에서 처음 쓰는 도구와 옵션은 주석이나 바로 다음 문장에서 무엇을 하는지 한 줄로 밝힙니다.
- 마지막 줄은 반드시 @@end 입니다.`

// ReviewTask is the default fixed instruction of the review step (block review.task).
const ReviewTask = `
## 작업
당신은 강사가 아니라 엄격한 기술 검수자입니다. 초안을 처음부터 끝까지 두 번 읽습니다.

1차 — 초보 독자 시뮬레이션: 당신은 웹 FE/BE 실무 개발자입니다(HTTP·JSON·JS/TS·SQL 기본 CRUD·git·기초 셸은 압니다). OS·네트워크 스택·DB 내부·인프라·보안 용어는 모릅니다. 이 독자로서 읽다가 "이 말을 몰라서" 문장의 요점을 놓치는 곳을 모두 reader_gaps 에 적습니다. 정의가 그 문장이나 바로 다음 문장에 없고, 위쪽에서도 풀린 적이 없는 용어·약어·도구·명령 옵션·선행 메커니즘, 그리고 "당연히/알다시피" 류 전제 문장이 대상입니다. 같은 것은 첫 등장 한 번만 적습니다.

2차 — 기술 검수: 다음을 지적합니다.
- factual_error: 사실 오류(동작 설명, 명령어 옵션, 출력 형식, 기본값, 버전 의존 동작을 일반론처럼 쓴 경우 포함)
- omission: 제공된 학습 목표 중 설명되지 않았거나 피상적으로만 다룬 것, 이해에 꼭 필요한데 빠진 단계
- assumed_knowledge: 전제 지식 누락 — reader_gaps 중 그 문장의 요점(판단 기준·핵심 원리)이 그 지식에 달린 것. 판단 섹션에서 처음 나오는 새 개념, 정의 없이 쓰인 약어, 다리 없이 건너뛴 선행 메커니즘이 전형입니다.
- overclaim: 과장·단정("항상", "절대", 근거 없는 수치), 확실하지 않은 구체값을 지어낸 것
- source: 출처가 공식 문서가 아님(블로그·Q&A·개인 사이트), 본문 주장과 무관한 링크, 존재가 의심되는 URL
- terminology: 같은 용어를 다르게 정의하거나 기존 개념과 모순되는 정의
- pedagogy: 이미 배운 개념을 다시 길게 가르침, 실무 상황과 본문이 연결되지 않음
- typo: 오타, 단어 중복(예: "INSERT INSERT INTO"), 닫히지 않은 코드블록·도식, 그대로 실행되지 않는 코드·명령. 실행이 깨지면 major, 표현만 어색하면 minor

severity 는 critical(틀린 지식을 가르침) / major(목표 미달·근거 부족, 또는 핵심 문장이 정의 없는 전제에 기대 이해되지 않음) / minor(표현 개선, 곁가지 용어의 정의 누락)입니다.
critical 또는 major 가 하나라도 있으면 verdict 는 "fail" 입니다. 확신이 없는 지적은 minor 로 두고 이유를 적습니다.

JSON만 출력하세요:
{
  "verdict": "pass" | "fail",
  "summary": "한두 문장 총평",
  "reader_gaps": [
    {"section": "situation_question|principles|walkthrough|judgment_next", "quote": "막힌 문장 일부(그대로 인용)",
     "unknown": "모를 수 있는 용어·메커니즘", "why": "웹 개발자가 이걸 모를 수 있는 이유",
     "fix": "한 줄 정의 / 2~3문장 다리 / 개념 링크 중 무엇을 어떻게"}
  ],
  "issues": [
    {"severity": "critical|major|minor", "kind": "factual_error|omission|assumed_knowledge|overclaim|source|terminology|pedagogy|typo",
     "section": "situation_question|principles|walkthrough|judgment_next|header", "quote": "문제 문장 일부",
     "message": "무엇이 왜 문제인지", "suggestion": "어떻게 고칠지"}
  ]
}`

// Review asks a separate reviewer pass to critique a lecture draft.
func Review(c LectureContext, markup string, set Set) string {
	var b strings.Builder
	writeContext(&b, c)
	b.WriteString("\n## 검수할 강의 초안\n<<<LECTURE\n" + markup + "\nLECTURE>>>\n")
	b.WriteString(set.Text(BlockReviewTask, c.vars()))
	return b.String()
}

// SystemReviewer is the system prompt for the review pass.
const SystemReviewer = `당신은 CS·운영 분야의 시니어 엔지니어이자 기술 감수자입니다. 칭찬하지 말고 문제만 정확히 찾습니다. 확인할 수 없는 사실은 "확인 필요"로 표시하고 단정하지 않습니다. 출력은 요청된 JSON 하나뿐입니다.`

func writeContext(b *strings.Builder, c LectureContext) {
	fmt.Fprintf(b, "## 대상 개념\n- id: %s\n- 이름: %s\n- 영역: %s\n- 깊이: %d (1 기본 ~ 3 심화)\n- 요약: %s\n",
		c.ConceptID, c.ConceptName, c.Area, c.Depth, c.Summary)
	if c.Query != "" {
		fmt.Fprintf(b, "- 수강생이 입력한 질문: %q (이 질문이 첫 상황이 되도록 하세요)\n", c.Query)
	}
	b.WriteString("\n## 이번 강의의 학습 목표(모두 다룰 것)\n")
	for _, o := range c.Objectives {
		fmt.Fprintf(b, "- %s: %s\n", o.ID, o.Text)
	}
	if len(c.PracticalCases) > 0 {
		b.WriteString("\n## 실무 사례 후보\n")
		for _, p := range c.PracticalCases {
			b.WriteString("- " + p + "\n")
		}
	}
	if len(c.DocLinks) > 0 {
		b.WriteString("\n## 커리큘럼에 등록된 공식 문서(우선 인용)\n")
		for _, d := range c.DocLinks {
			fmt.Fprintf(b, "- [%s](%s)\n", d.Title, d.URL)
		}
	}
	b.WriteString("\n## 수강생이 이미 배운 개념(다시 가르치지 말고 첫 등장에 [용어](term:id) 링크 + 한 줄 상기)\n")
	writeRefs(b, c.Learned)
	b.WriteString("\n## 아직 배우지 않은 선행 개념(본문에서 쓰면 그 자리에서 2~3문장으로 다리를 놓고 [용어](term:id) 링크)\n")
	writeRefs(b, c.Unlearned)
	b.WriteString("\n## 연결 가능한 기존 개념([용어](term:id) 링크에 쓸 수 있는 id)\n")
	writeRefs(b, c.Linkable)
	writeCatalog(b, c.Catalog)
	if c.StyleHint != "" {
		b.WriteString("\n## 설명 방식 조정\n" + c.StyleHint + "\n")
	}
	b.WriteString(ExampleLanguageRule(c.ExampleLanguage))
}

// ExampleLanguageRule tells the model which language to use for application
// code examples. Shell commands, SQL, config files and tool logs keep their
// own language — only "code a developer would write" follows the preference.
func ExampleLanguageRule(language string) string {
	if strings.TrimSpace(language) == "" {
		return ""
	}
	return fmt.Sprintf(`
## 예시 코드 언어
수강생은 %[1]s 개발자입니다. 애플리케이션 코드 예시(함수·클래스·클라이언트 호출 등)는 반드시 %[1]s 로 쓰고 lang 도 그에 맞게 지정하세요.
- 셸 명령어(bash), SQL, 설정 파일(yaml 등), 도구 출력 로그는 원래 형식을 그대로 씁니다. DB 주제처럼 SQL 자체가 핵심이면 SQL 예시만으로 충분합니다.
- 다른 언어의 동작 차이가 핵심일 때만 그 언어를 짧게 보여 주고, 왜 다른 언어를 썼는지 한 줄로 밝힙니다.
`, language)
}

// writeCatalog lists the rest of the knowledge map compactly, one line per area.
func writeCatalog(b *strings.Builder, refs []CatalogRef) {
	if len(refs) == 0 {
		return
	}
	b.WriteString("\n## 지식 지도의 다른 개념(본문에서 이름이 나오면 첫 등장에 [용어](term:id) 링크. \"다른 강의에서 다룹니다\"만 쓰지 말 것. ✓=이미 배움)\n")
	area := ""
	for i, r := range refs {
		if r.Area != area {
			if i > 0 {
				b.WriteString("\n")
			}
			area = r.Area
			fmt.Fprintf(b, "- %s:", area)
		} else {
			b.WriteString(",")
		}
		mark := ""
		if r.Learned {
			mark = "✓"
		}
		fmt.Fprintf(b, " %s %s%s", r.ID, r.Name, mark)
	}
	b.WriteString("\n")
}

func writeRefs(b *strings.Builder, refs []ConceptRef) {
	if len(refs) == 0 {
		b.WriteString("- (없음)\n")
		return
	}
	for _, r := range refs {
		fmt.Fprintf(b, "- %s: %s\n", r.ID, r.Name)
	}
}

// StyleHintFor maps recent feedback to a generation adjustment.
func StyleHintFor(feedback string) string {
	switch feedback {
	case "too_easy":
		return "직전 강의가 너무 쉬웠다는 피드백이 있습니다. 기초 정의는 짧게 줄이고 내부 동작·경계 사례·트레이드오프 비중을 늘리세요."
	case "too_hard":
		return "직전 강의가 어려웠다는 피드백이 있습니다. 단계를 더 잘게 나누고, 각 단계마다 구체적인 예와 그림(mermaid)을 붙이세요. 전문 용어는 처음 나올 때 한 줄로 풀어 쓰세요."
	case "insufficient":
		return "설명이 부족했다는 피드백이 있습니다. 같은 내용을 다른 비유·다른 예제로 한 번 더 보여주세요."
	case "want_more":
		return "더 알고 싶다는 피드백이 있습니다. judgment_next 에서 다음 심화 주제와 연결을 구체적으로 제시하세요."
	}
	return ""
}
