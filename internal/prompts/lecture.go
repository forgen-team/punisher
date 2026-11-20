// Package prompts holds the Korean LLM prompts for Punisher. Prompts are
// versioned (Version) and the version is stored on every generated lecture.
package prompts

import (
	"fmt"
	"strings"
)

// Version is recorded in lectures.prompt_version.
const Version = "punisher-prompts-2026-11-20.2"

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

// Misconception is a misconception the learner acknowledged on a check result.
type Misconception struct {
	ConceptName string
	Text        string
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
	// Revisit: completed concepts the learner was weak on in checks — remind and bridge
	// where used (user-owned generations only, design §5 P4 M6; never also in Learned/Catalog ✓).
	Revisit []ConceptRef
	// Unlearned: prerequisites the learner has NOT completed — must be bridged/defined where used.
	Unlearned []ConceptRef
	// Catalog: every other concept of the knowledge map (link targets for names
	// the lecture mentions in passing, instead of "다른 강의에서 다룹니다").
	Catalog []CatalogRef
	// Linkable: existing concept ids that term blocks may reference.
	Linkable []ConceptRef
	// StyleHint: adjustment from feedback (too_easy / too_hard / insufficient ...).
	StyleHint string
	// Misconceptions: open misconceptions the learner acknowledged on this concept and its
	// prerequisites (user-owned generations only, design §5 P4 M6).
	Misconceptions []Misconception
	// Query: the learner's original question for instant lectures (may be empty).
	Query string
	// ExampleLanguage: display name of the learner's preferred example-code language ("" = no preference).
	ExampleLanguage string
	// LabOS: the reader's lab environment for user-owned generations (macos | linux | none);
	// "" = library lecture (labs for both systems, design v3 §4.1).
	LabOS string
	// LearnerSignals: where the learner got stuck in the version being remade
	// ("섹션 제목 — 이유"), user-owned regenerations only (v3 §6).
	LearnerSignals []string
	// Glossary: basic terms of the shipped dictionary (term:g.…), linked instead of re-defined.
	Glossary []GlossaryRef
	// Reader: the reader level section appended after the blocks (ReaderProfile).
	Reader Reader
}

// GlossaryRef is a dictionary term the lecture may link.
type GlossaryRef struct {
	ID   string
	Name string
}

// SystemTeacher is the shared system prompt for outline/lecture generation.
const SystemTeacher = `당신은 "Punisher"의 강사입니다. 수강생이 누구인지(배경·전제 지식·설명 밀도)는 요청 끝의 '독자 수준' 절을 따릅니다. 매일 약 30분 학습 세션에서 약 20분 분량의 "좋은 강의 한 편"을 읽고, 손으로 따라 계산하고, 예측하고, 직접 풀어 본 뒤 짧은 확인을 합니다.

원칙:
1. 실무 상황에서 출발합니다. 첫 문단은 수강생이 실제로 겪었을 법한 장면(장애, 느린 쿼리, 이상한 로그)이고, 그 장면이 오늘의 질문으로 이어집니다.
2. 메커니즘을 단계별로 설명합니다. "그냥 그렇다"가 아니라 무엇이 어떤 순서로 일어나는지, 왜 그렇게 설계됐는지 인과를 보여줍니다.
3. 코드·명령어·로그·도식으로 직접 보여줍니다. 출력 예시는 실제 도구가 내는 형태를 따르되, 구체적인 숫자·버전·플래그가 확실하지 않으면 지어내지 말고 "환경에 따라 다르다"고 쓰거나 생략합니다.
4. 흔한 오해를 짚습니다. 수강생이 흔히 갖는 잘못된 직관을 명시하고 왜 틀렸는지 설명합니다.
5. 판단 기준과 한계를 줍니다. 언제 이 지식이 결정을 바꾸는지, 어디까지가 일반론이고 어디부터 구현·버전에 따라 다른지 구분합니다.
6. 공식 문서를 근거로 인용합니다(언어/OS man page, RFC, 공식 레퍼런스, 공식 엔지니어링 문서). 블로그·Q&A 사이트는 근거로 쓰지 않습니다. 확신할 수 없는 URL은 만들지 말고, 제공된 공식 문서 링크를 우선 사용합니다.
7. 이미 배운 개념은 다시 가르치지 않습니다. 첫 등장에서 용어 링크([용어](term:개념id))를 걸고 한 줄로 상기시킨 뒤 넘어갑니다.
8. 존댓말로, 군더더기 없이, 정확하게 씁니다. 과장("완벽하게", "항상", "절대")을 피하고 조건을 명시합니다.
9. 전제하지 않습니다. 수강생이 안다고 가정해도 되는 범위는 '독자 수준' 절의 전제 지식뿐입니다. 그 밖의 용어, 약어, 도구 이름은 모른다고 가정합니다.
10. 첫 등장에서 정의합니다. 그 범위 밖의 용어·약어·도구·명령 옵션이 처음 나오는 문장(또는 바로 다음 문장)에서 한 줄로 뜻을 풀어 씁니다(괄호 정의 가능). 지식 지도에 있는 개념이면 그 첫 등장에 링크도 겁니다. 정의를 뒤 섹션이나 용어 블록에 미루지 않습니다. 표·목록·로그 해설·판단 섹션 안의 약어도 같습니다.
11. 선행 메커니즘에는 다리를 놓습니다. 문장을 이해하려면 아직 배우지 않은 동작 원리(예: "감독자가 죽은 자식을 다시 띄운다", "커밋 전에는 다른 세션이 못 본다")가 필요하면, 그 자리에서 2~3문장으로 무엇이 어떤 순서로 일어나는지 설명하고 자세한 내용은 링크로 넘깁니다.
12. "당연히", "알다시피", "물론", "익히 아는", "잘 알려진", "기본이므로" 같은 전제 표현을 쓰지 않습니다. 이름만 던지는 "다른 강의에서 다룹니다"도 쓰지 않습니다 — 쓰려면 한 줄 정의나 링크를 함께 둡니다.`

// Outline asks for a JSON outline of the lecture.
func Outline(c LectureContext, set Set) string {
	var b strings.Builder
	writeContext(&b, c)
	b.WriteString(set.Text(BlockOutlineTask, c.vars()))
	writeReaderProfile(&b, c.Reader)
	writeDepthScope(&b, c.Depth)
	writeFactualRules(&b)
	return b.String()
}

// DepthOneScope keeps product/version-dependent details out of depth-1
// lectures (they were the unverifiable claims that failed review). It is a
// fixed safety rule like FactualRules, so it is not an editable block.
const DepthOneScope = `
## 깊이 1 범위
- 이 강의는 깊이 1(기본)입니다. 운영 마이그레이션 절차, 드라이버·라이브러리별 반환 타입, 잠금 수준처럼 제품·버전에 따라 달라지는 세부는 본문 주장으로 쓰지 않습니다.
- 꼭 필요하면 judgment_next 의 판단 기준에 "확인할 것"(무엇을 어느 공식 문서에서 확인할지)으로만 적습니다.
`

// writeDepthScope appends DepthOneScope for a depth-1 (or unset) lecture.
func writeDepthScope(b *strings.Builder, depth int) {
	if depth <= 1 {
		b.WriteString(DepthOneScope)
	}
}

// RevisionRules is the fixed instruction of a partial rewrite: fix only what
// the findings point at and keep everything else verbatim.
const RevisionRules = `
## 수정 방식(엄격)
- 위 지적이 가리키는 문장·블록만 고칩니다. 지적과 관계없는 문장·블록·섹션은 글자 하나 바꾸지 말고 그대로 옮겨 씁니다(블록 순서·지시 줄 포함).
- 확인할 수 없다거나 사실과 다르다고 지적된 주장은 다음 순서로 고칩니다: ① 그 주장을 뺍니다 ② 뺄 수 없으면 조건(제품·버전·"환경에 따라 다름")을 붙이거나 판단 기준의 "확인할 것"으로 바꿉니다. 새 세부 사실을 더해 메우지 않습니다.
- 출력이 실제와 다르다는 지적은, 그 출력이 언어 표준만으로 결정되지 않으면 출력 예시를 빼거나 무엇을 확인할지로 바꿉니다.
- "개선 권장" 항목은 고쳐도 새 사실 오류가 생기지 않는 범위에서만 반영합니다. 반영하려고 새 주장을 늘리지 않습니다.
- 출력은 고친 강의 전체입니다(아래 출력 형식 그대로, 마지막 줄 @@end).
`

// LectureRevision asks for a partial rewrite of previous: the findings in
// notes are fixed and the rest is kept verbatim (code compares the blocks).
func LectureRevision(c LectureContext, outlineJSON, previous, notes string, set Set) string {
	var b strings.Builder
	writeContext(&b, c)
	if outlineJSON != "" {
		b.WriteString("\n## 확정된 개요\n```json\n" + outlineJSON + "\n```\n")
	}
	b.WriteString("\n## 이전 초안(부분 수정 대상)\n<<<DRAFT\n" + previous + "\nDRAFT>>>\n")
	b.WriteString("\n## 이전 초안에 대한 지적\n" + notes + "\n")
	b.WriteString(RevisionRules)
	b.WriteString(set.Text(BlockLectureFormat, c.vars()))
	writeReaderProfile(&b, c.Reader)
	writeDepthScope(&b, c.Depth)
	writeFactualRules(&b)
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
		b.WriteString("\n## 이전 초안에 대한 검수 지적('반드시 고칠 것'은 모두 고칠 것)\n" + revisionNotes + "\n")
	}
	b.WriteString(set.Text(BlockLectureFormat, c.vars()))
	writeReaderProfile(&b, c.Reader)
	writeDepthScope(&b, c.Depth)
	writeFactualRules(&b)
	return b.String()
}

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
	writeRevisit(b, c.Revisit)
	b.WriteString("\n## 아직 배우지 않은 선행 개념(본문에서 쓰면 그 자리에서 2~3문장으로 다리를 놓고 [용어](term:id) 링크)\n")
	writeRefs(b, c.Unlearned)
	b.WriteString("\n## 연결 가능한 기존 개념([용어](term:id) 링크에 쓸 수 있는 id)\n")
	writeRefs(b, c.Linkable)
	writeCatalog(b, c.Catalog)
	writeGlossary(b, c.Glossary)
	if c.StyleHint != "" {
		b.WriteString("\n## 설명 방식 조정\n" + c.StyleHint + "\n")
	}
	writeMisconceptions(b, c.Misconceptions)
	writeLearnerSignals(b, c.LearnerSignals)
	b.WriteString(ExampleLanguageRule(c.ExampleLanguage))
	b.WriteString(LabRule(c.LabOS))
}

// LabRule tells the model which system @@lab blocks target (v3 §4.1).
func LabRule(os string) string {
	switch os {
	case "none":
		return "\n## 실습\n수강생은 실습 환경을 쓰지 않습니다. @@lab 블록을 쓰지 마세요.\n"
	case "macos", "linux":
		name := map[string]string{"macos": "macOS", "linux": "Linux"}[os]
		return fmt.Sprintf("\n## 실습\n수강생의 터미널은 %s 입니다. @@lab 은 os=%s(또는 두 시스템에서 같으면 os=any)로 씁니다.\n", name, os)
	}
	return "\n## 실습\n수강생마다 macOS·Linux 가 다릅니다. 명령이 두 시스템에서 같으면 @@lab os=any 하나, 다르면 os=macos 와 os=linux 두 블록을 연달아 씁니다.\n"
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

// writeGlossary lists the dictionary terms compactly ("id 이름" per line).
func writeGlossary(b *strings.Builder, refs []GlossaryRef) {
	if len(refs) == 0 {
		return
	}
	b.WriteString("\n## 용어 사전(지도에 없는 기본 용어. 본문에 나오면 첫 등장에 [표기](term:id) 링크를 겁니다. 링크를 누르면 사전 설명이 뜨므로 본문에서 따로 길게 정의하지 않고, 문장 흐름에 필요하면 괄호로 몇 단어만 덧붙입니다)\n")
	for _, r := range refs {
		fmt.Fprintf(b, "- %s %s\n", r.ID, r.Name)
	}
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

// writeRevisit lists the concepts to revisit (omitted when none).
func writeRevisit(b *strings.Builder, refs []ConceptRef) {
	if len(refs) == 0 {
		return
	}
	b.WriteString("\n## 다시 짚을 개념(배웠지만 확인에서 약했던 개념 — 본문에서 쓰면 첫 등장에 [용어](term:id) 링크 + 한 줄 상기 + 2~3문장 다리 설명)\n")
	writeRefs(b, refs)
}

// writeMisconceptions lists the learner's acknowledged misconceptions (omitted when none).
func writeMisconceptions(b *strings.Builder, list []Misconception) {
	if len(list) == 0 {
		return
	}
	b.WriteString("\n## 이 수강생이 인정한 오개념(관련 내용이 나오면 이 생각이 왜 틀렸는지 바로잡을 것)\n")
	for _, m := range list {
		fmt.Fprintf(b, "- %s: %s\n", m.ConceptName, m.Text)
	}
}

// writeLearnerSignals lists where the learner got stuck in the previous version (v3 §6).
func writeLearnerSignals(b *strings.Builder, list []string) {
	if len(list) == 0 {
		return
	}
	b.WriteString("\n## 이전 버전에서 이 수강생이 막힌 곳(이 부분은 설명 순서·예·연습을 바꿔 더 쉽게 따라오게 할 것)\n")
	for _, s := range list {
		fmt.Fprintf(b, "- %s\n", s)
	}
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
