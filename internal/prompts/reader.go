package prompts

import "strings"

// Reader is who a reader-facing prompt writes for (learner-level design ①):
// the user's reader level and the output locale of the section's text.
type Reader struct {
	// Level: beginner | novice | practitioner | advanced; "" or unknown = practitioner.
	Level string
	// Locale: ko | en (the generation's output language); "" = ko.
	Locale string
}

// Reader levels (same values as learning.Reader*; prompts does not import learning).
const (
	ReaderBeginner     = "beginner"
	ReaderNovice       = "novice"
	ReaderPractitioner = "practitioner"
	ReaderAdvanced     = "advanced"
)

// readerText is one level's section in one locale.
type readerText struct {
	label, reader, assumes, density string
}

// Practitioner keeps the reader assumption every prompt had before levels
// existed, verbatim (SystemTeacher opening sentence and rule 9).
const (
	PractitionerReaderKo  = "수강생은 실무 경력이 있는 개발자지만 CS·운영 기반 지식(OS, 네트워크, DB, 분산 시스템, 인프라, 보안)에 공백이 있습니다."
	PractitionerAssumesKo = "수강생이 안다고 가정해도 되는 것은 웹 FE/BE 실무 범위(HTTP 요청·응답과 상태 코드, JSON, JS/TS 코드, SQL 기본 CRUD, git, cd·ls·cat·grep 정도의 셸)뿐입니다. OS·네트워크 스택·DB 내부·분산 시스템·인프라·보안의 용어, 약어, 도구 이름(예: systemd, 시그널, 서브넷, NAT, SYN, 격리 수준, 락, 시퀀스, SLO)은 모른다고 가정합니다."
)

var readerTextsKo = map[string]readerText{
	ReaderBeginner: {label: "개발은 처음(beginner)",
		reader:  "수강생은 개발을 처음 배우는 사람입니다. 컴퓨터와 인터넷(브라우저·앱·파일 저장)을 일상적으로 쓸 줄만 압니다.",
		assumes: "안다고 가정해도 되는 것은 일상적인 컴퓨터·인터넷 사용뿐입니다. 코드(변수·함수 포함), HTTP, JSON, SQL, 터미널·셸 명령은 모른다고 가정합니다.",
		density: "모든 기술 용어·약어는 첫 등장에서 일상 말로 풀고, 일상 비유 하나로 그림을 먼저 준 뒤 정확한 정의로 넘어갑니다. 코드·명령은 줄마다 무엇을 하는지 설명합니다. 첫 장면은 사용자로서 겪는 장면(예: 페이지가 늦게 뜬다)이어도 됩니다. 단계를 잘게 나누고 한 문장에 새 용어를 하나만 둡니다."},
	ReaderNovice: {label: "코드를 조금 써 봄(novice)",
		reader:  "수강생은 코드를 조금 써 본 사람입니다. 변수·조건문·반복문·함수와 간단한 웹 페이지 정도는 만들어 봤습니다.",
		assumes: "HTTP·JSON·SQL·git·셸은 이름만 안다고 가정합니다. 쓰면 첫 등장에서 한 줄로 무엇인지 풉니다. OS·네트워크 스택·DB 내부·분산 시스템·인프라·보안의 용어, 약어, 도구 이름은 모른다고 가정합니다.",
		density: "범위 밖 용어는 첫 등장에서 한 줄로 정의하고, 선행 메커니즘은 3~4문장 다리를 놓습니다. 코드 예시는 짧게 두고 낯선 문법·라이브러리 호출에는 한 줄 설명을 붙입니다. 첫 장면은 작은 개인 프로젝트나 간단한 웹 서비스에서 겪을 법한 장면으로 씁니다."},
	ReaderPractitioner: {label: "웹 개발 실무자(practitioner)",
		reader: PractitionerReaderKo, assumes: PractitionerAssumesKo,
		density: "범위 밖 용어는 첫 등장에서 한 줄로 정의하고, 아직 배우지 않은 선행 메커니즘은 2~3문장 다리를 놓습니다."},
	ReaderAdvanced: {label: "CS 기본기 탄탄(advanced)",
		reader:  "수강생은 CS 기본기(자료구조, OS, 네트워크, DB의 기초 개념)가 탄탄한 개발자입니다.",
		assumes: "깊이 1 수준의 정의(프로세스·스레드, TCP, 인덱스, 트랜잭션 등)는 안다고 가정합니다. 이 개념 고유의 용어와 내부 구현 용어는 첫 등장에서 정의합니다.",
		density: "깊이 1 정의는 한 줄로 압축하고, 내부 구현·트레이드오프·장애 사례·측정 방법 위주로 씁니다. 기초를 길게 다시 설명하지 않습니다."},
}

var readerTextsEn = map[string]readerText{
	ReaderBeginner: {label: "new to programming (beginner)",
		reader:  "The learner is new to programming. They only use computers and the internet (browser, apps, saving files) in everyday life.",
		assumes: "Assume only everyday computer and internet use. Assume they do not know code (including variables and functions), HTTP, JSON, SQL, or the terminal and shell commands.",
		density: "Explain every technical term and abbreviation in plain words where it first appears; give one everyday analogy first, then the precise definition. Explain what each line of code or command does. The opening scene may be one a user experiences (e.g. a page loads slowly). Use small steps and at most one new term per sentence."},
	ReaderNovice: {label: "has written a little code (novice)",
		reader:  "The learner has written a little code: variables, conditionals, loops, functions and a simple web page.",
		assumes: "Assume they know HTTP, JSON, SQL, git and the shell by name only; when you use one, say in one line what it is where it first appears. Assume they do not know the terms, abbreviations or tool names of operating systems, the network stack, database internals, distributed systems, infrastructure or security.",
		density: "Define out-of-scope terms in one line where they first appear and bridge prerequisite mechanisms in 3–4 sentences. Keep code examples short and add a one-line note to unfamiliar syntax or library calls. Set the opening scene in a small personal project or a simple web service."},
	ReaderPractitioner: {label: "working web developer (practitioner)",
		reader:  "The learner is a developer with professional experience but gaps in CS and operations fundamentals (OS, networking, databases, distributed systems, infrastructure, security).",
		assumes: "Assume they know only the scope of web front-end/back-end work (HTTP requests, responses and status codes, JSON, JS/TS code, basic SQL CRUD, git, and shell basics such as cd, ls, cat, grep). Assume they do not know the terms, abbreviations or tool names of operating systems, the network stack, database internals, distributed systems, infrastructure or security (e.g. systemd, signals, subnets, NAT, SYN, isolation levels, locks, sequences, SLOs).",
		density: "Define out-of-scope terms in one line where they first appear and bridge prerequisite mechanisms not yet learned in 2–3 sentences."},
	ReaderAdvanced: {label: "solid CS fundamentals (advanced)",
		reader:  "The learner is a developer with solid CS fundamentals (the basics of data structures, operating systems, networking and databases).",
		assumes: "Assume they know depth-1 definitions (processes and threads, TCP, indexes, transactions and the like). Still define terms specific to this concept and its internals where they first appear.",
		density: "Compress depth-1 definitions to one line and focus on internals, trade-offs, failure cases and how to measure. Do not re-explain the basics at length."},
}

// ReaderProfile is the fixed, non-editable reader section of every
// reader-facing prompt (like FactualRules and DepthOneScope). It states the
// level's reader assumption and explanation density, that facts and objectives
// do not change with the level, and that it overrides any reader assumption in
// the (possibly user-edited) blocks above. Unknown or "" level = practitioner;
// locale "en" = English text, otherwise Korean.
func ReaderProfile(level, locale string) string {
	if locale == "en" {
		t := readerTexts(readerTextsEn, level)
		return "\n## Reader level: " + t.label + " (this section overrides any reader assumption written in the instructions and blocks above)\n" +
			"- Reader: " + t.reader + "\n- Assumed knowledge: " + t.assumes + "\n- Explanation density: " + t.density + "\n" +
			"- The facts and learning objectives taught are the same at every level; only the explanation density changes (how much definitions are unpacked, how fine the steps, examples and analogies are). Do not drop objectives or simplify facts until they are wrong to fit the level.\n" +
			"- Reviews and reader simulations judge the text for this reader.\n"
	}
	t := readerTexts(readerTextsKo, level)
	return "\n## 독자 수준: " + t.label + " (이 절이 위의 지시·블록에 적힌 독자 가정보다 우선합니다)\n" +
		"- 독자: " + t.reader + "\n- 전제 지식: " + t.assumes + "\n- 설명 밀도: " + t.density + "\n" +
		"- 가르치는 사실과 학습 목표는 수준과 무관하게 같습니다. 바뀌는 것은 설명 밀도(정의를 얼마나 풀어 쓰는지, 단계·예·비유를 얼마나 잘게 두는지)뿐입니다. 수준에 맞추려고 목표를 빼거나 사실을 틀리게 단순화하지 않습니다.\n" +
		"- 검수와 독자 시뮬레이션도 이 독자를 기준으로 판단합니다.\n"
}

func readerTexts(m map[string]readerText, level string) readerText {
	if t, ok := m[strings.TrimSpace(level)]; ok {
		return t
	}
	return m[ReaderPractitioner]
}

// writeReaderProfile appends ReaderProfile for r to a prompt being built.
func writeReaderProfile(b *strings.Builder, r Reader) {
	b.WriteString(ReaderProfile(r.Level, r.Locale))
}
