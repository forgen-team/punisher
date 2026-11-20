package prompts

import (
	"fmt"
	"strings"
)

// ReviewTask is the default fixed instruction of the review step (block
// review.task), strengthened by quality design v2.2 §3.3.
const ReviewTask = `
## 작업
당신은 강사가 아니라 엄격한 기술 검수자입니다. 초안을 처음부터 끝까지 두 번 읽습니다.

1차 — 독자 시뮬레이션: 당신은 아래 '독자 수준' 절의 독자입니다(그 절의 전제 지식만 압니다). 이 독자로서 읽다가 "이 말을 몰라서" 문장의 요점을 놓치는 곳을 모두 reader_gaps 에 적습니다. 정의가 그 문장이나 바로 다음 문장에 없고, 위쪽에서도 풀린 적이 없는 용어·약어·도구·명령 옵션·선행 메커니즘, 그리고 "당연히/알다시피" 류 전제 문장이 대상입니다. 같은 것은 첫 등장 한 번만 적습니다.

2차 — 기술 검수: 아래 항목을 하나씩 판정해 문제를 지적합니다.
- factual_error: 사실 오류(동작 설명, 명령어 옵션, 출력 형식, 기본값, 버전 의존 동작을 일반론처럼 쓴 경우 포함). 특히 다음은 major 이상입니다:
  · 개념 쌍의 혼동: 자리올림/오버플로, 상한 O/정확한 차수 Θ("비례"로 정의), 평균 사례/분할 상환, 코드 포인트/문자소/바이트, 행 수/시퀀스 카운터
  · 처방 추적 누락: 처방을 적용한 뒤에도 같은 장애가 남는데(예: 컬럼 타입만 바꾸고 시퀀스 상한·테이블 재작성·잠금을 빠뜨림) 해결된 것처럼 씀
  · 제품·모드 미표기 단정("DB는 거부한다" — 어떤 DB·어떤 모드인지 없음)
  · 진단 지표의 원천 오류(카운터가 원천인데 MAX(id) 같은 파생값을 지표로 제시)
  · 비결정적 수치(실행 시간·처리량)를 조건 없이 단정
- 코드·출력 대조: 각 코드 블록의 출력(@@log, @@pause 다음 블록, @@answer)이 그 코드의 실제 출력과 맞는지 단계별로 계산해 확인합니다. 틀리면 factual_error(major 이상, blocking). 실행해 보지 않고는 확인할 수 없으면 "확인 필요"로 적고 advisory 로 둡니다(틀렸다는 근거를 댈 수 있을 때만 blocking). 벤치 코드가 본문의 결론과 반대 결과를 내면 critical 입니다.
- structure: 핵심 정의 누락·지연(목표의 핵심 명사가 principles 첫머리에서 정식 정의되지 않음), 통합 논지가 하위 주제와 모순, 첫 장면의 숫자·현상을 judgment_next 에서 회수하지 않음, 공식만 단독 제시(작은 값 계산 없이). 모두 major 입니다.
- pedagogy: 손계산(@@trace) 없음, 예측 멈춤(@@pause) 없음, 직접 해 보기(@@exercise) 2개 미만, 같은 수치 3회 이상 반복, 한 불릿에 개념 여럿, 이미 배운 개념을 다시 길게 가르침, 실무 상황과 본문이 연결되지 않음. 앞의 다섯은 major 입니다.
- omission: 제공된 학습 목표 중 설명되지 않았거나 피상적으로만 다룬 것, 이해에 꼭 필요한데 빠진 단계
- assumed_knowledge: 전제 지식 누락 — reader_gaps 중 그 문장의 요점(판단 기준·핵심 원리)이 그 지식에 달린 것. 판단 섹션에서 처음 나오는 새 개념, 정의 없이 쓰인 약어, 다리 없이 건너뛴 선행 메커니즘이 전형입니다.
- overclaim: 과장·단정("항상", "절대", 근거 없는 수치), 확실하지 않은 구체값을 지어낸 것
- source: 출처가 공식 문서가 아님(블로그·Q&A·개인 사이트), 본문 주장과 무관한 링크, 존재가 의심되는 URL
- terminology: 같은 용어를 다르게 정의하거나 기존 개념과 모순되는 정의
- typo: 오타, 단어 중복(예: "INSERT INSERT INTO"), 닫히지 않은 코드블록·도식, 그대로 실행되지 않는 코드·명령. 실행이 깨지면 major, 표현만 어색하면 minor

severity 는 critical(틀린 지식을 가르침) / major(목표 미달·근거 부족, 위에서 major 로 정한 항목, 또는 핵심 문장이 정의 없는 전제에 기대 이해되지 않음) / minor(표현 개선, 곁가지 용어의 정의 누락)입니다. severity 는 중요도이고, 발행을 막는지는 gate 가 정합니다.

gate — 지적마다 발행을 막는지 정합니다:
- blocking: 학습자가 틀린 것을 배우거나, 따라 하면 실패하는 것만입니다. kind 는 factual_error / overclaim / typo 중 하나입니다.
  · 사실 오류: 동작·명령 옵션·출력 형식·기본값이 틀림, 위의 개념 쌍 혼동
  · 오도: 제품·모드 미표기 단정, 틀린 결론으로 이어지는 과잉 일반화, 진단 지표의 원천 오류, 비결정적 수치의 단정
  · 처방 미해결: 처방을 적용해도 같은 장애가 남는데 해결된 것처럼 씀
  · 에러·출력 불일치: 코드의 실제 출력·에러가 본문에 보인 것과 다름, 그대로 실행되지 않는 코드, 결론과 반대인 벤치 결과
- advisory: 고치면 좋아지지만 틀린 것을 가르치지는 않는 것입니다. 구성(structure), 학습 활동(pedagogy: 손계산·예측·연습 부족), 같은 수치 반복, 요약 불릿 구성, 용어 정의 누락(assumed_knowledge·terminology), 목표를 얕게 다룸(omission), 분량·깊이에 비해 많은 세부, 출처(source), 표현(minor typo)이 모두 advisory 입니다.
- "확인 필요": 실행하지 않고는 확인할 수 없다는 것만으로는 blocking 이 아닙니다. 본문이 조건 없이 단정했고 틀렸을 근거를 댈 수 있을 때만 blocking(factual_error)입니다. 제품·버전·"환경에 따라 다름" 같은 조건이 붙어 있거나 "확인할 것"으로 쓴 경우는 advisory 입니다.
- 버전·빌드·렌더러 호환(예: 도식 문법이 모든 렌더러 버전에서 그려지는지, 기본값이 빌드 옵션에 따라 다를 수 있는지)을 확인할 수 없다는 이유만으로는 blocking 이 아닙니다.
- 깊이 1 강의에 제품·버전 의존 세부(운영 마이그레이션, 드라이버별 반환 타입, 잠금 수준)가 있으면: 틀렸거나 조건 없이 단정했으면 blocking, 맞고 조건이 붙어 있으면 범위 초과로 advisory 입니다.
- 확신이 없는 지적은 advisory 로 두고 이유를 적습니다.

verdict 는 blocking 이 하나라도 있으면 "fail", 없으면 "pass" 입니다(advisory 만 있으면 pass).

JSON만 출력하세요:
{
  "verdict": "pass" | "fail",
  "summary": "한두 문장 총평",
  "reader_gaps": [
    {"section": "situation_question|principles|walkthrough|judgment_next", "quote": "막힌 문장 일부(그대로 인용)",
     "unknown": "모를 수 있는 용어·메커니즘", "why": "이 독자가 이걸 모를 수 있는 이유",
     "fix": "한 줄 정의 / 2~3문장 다리 / 개념 링크 중 무엇을 어떻게"}
  ],
  "issues": [
    {"severity": "critical|major|minor", "gate": "blocking|advisory", "kind": "factual_error|structure|pedagogy|omission|assumed_knowledge|overclaim|source|terminology|typo",
     "section": "situation_question|principles|walkthrough|judgment_next|header", "quote": "문제 문장 일부",
     "message": "무엇이 왜 문제인지", "suggestion": "어떻게 고칠지",
     "confidence": 0.0~1.0, "reader_impact": "blocking 이면 필수: 독자가 그대로 믿고 따라 하면 이르는 틀린 결과·판단 한 문장"}
  ]
}`

// legacyReviewNote is added when a stored lecture written before the material
// blocks is reviewed (backfill): judge the learning activities by content.
const legacyReviewNote = `
## 참고: 구조 블록 도입 전 강의
이 강의는 @@keypoint·@@trace·@@pause·@@exercise·@@summary 블록이 생기기 전에 쓰였습니다. 블록이 없다는 것 자체는 지적하지 말고, 손계산·예측 멈춤·직접 해 보기·핵심 정의가 본문 문장으로 있는지와 그 내용이 맞는지로 판정합니다. 사실 정확성 항목은 그대로 엄격하게 판정합니다.
`

// Review asks a separate reviewer pass to critique a lecture draft.
func Review(c LectureContext, markup string, set Set) string {
	return review(c, markup, set, false)
}

// ReviewLegacy is Review for a stored lecture without material blocks (backfill review).
func ReviewLegacy(c LectureContext, markup string, set Set) string {
	return review(c, markup, set, true)
}

func review(c LectureContext, markup string, set Set, legacy bool) string {
	var b strings.Builder
	writeContext(&b, c)
	b.WriteString("\n## 검수할 강의 초안\n<<<LECTURE\n" + markup + "\nLECTURE>>>\n")
	b.WriteString(set.Text(BlockReviewTask, c.vars()))
	writeReaderProfile(&b, c.Reader)
	writeDepthScope(&b, c.Depth)
	if legacy {
		b.WriteString(legacyReviewNote)
	}
	b.WriteString(BlockingRules)
	writeFactualRules(&b)
	return b.String()
}

// BlockingRules is the fixed rule of every review prompt (full review and
// targeted recheck) for blocking findings: code downgrades a blocking finding
// without enough confidence or a reader impact to advisory, so the fields are
// asked for even when the user edited the review block. Not user-editable.
const BlockingRules = `
## blocking 지적의 근거(코드가 확인합니다)
- gate 가 blocking 인 지적에는 반드시 "confidence"(0~1, 그 문장이 실제로 틀렸거나 독자를 오도한다고 확신하는 정도)와 "reader_impact"(독자가 그 문장을 그대로 믿고 따라 하면 어떤 틀린 결과·판단에 이르는지 한 문장)를 씁니다. confidence 가 0.7 미만이거나 reader_impact 가 비어 있으면 코드가 advisory 로 내립니다.
- 버전·빌드·렌더러 호환을 확인할 수 없다는 이유만으로는 blocking 이 아닙니다.
- 이미 조건(제품·버전·"환경에 따라 다름"·"확인할 것")이 붙어 그 조건 안에서 맞는 문장은 blocking 이 아닙니다. 조건이 있어도 핵심 주장이 틀렸거나 단정(예: "RST 가 오면 서버까지 도달한 것")이면 blocking 입니다.
- 사실 계열(factual_error·overclaim·typo)의 critical·major 지적은 gate 와 무관하게 코드가 차단 후보로 보고 재판정에 넘깁니다. 확신이 없으면 severity 를 minor 로 적습니다.
- 블록 끝의 "…(검수 입력 길이 제한으로 이하 생략…)" 표시는 이 검수 입력만 줄인 것입니다. 강의 원문은 잘리지 않았으므로 문장 잘림·미완결로 지적하지 않습니다.
`

// RecheckTask is the default instruction of a targeted re-review (block
// review.recheck): after a partial rewrite only the previous blocking findings
// and the changed blocks are judged; unchanged blocks keep the first verdict.
const RecheckTask = `
## 작업 — 표적 재검수
이 강의는 이전 검수에서 blocking 지적을 받고 부분 수정되었습니다. 강의 전체를 다시 검수하지 않습니다. 바뀌지 않은 블록은 이전 판정을 그대로 쓰므로 보지 않습니다.

1. "이전 blocking 지적" 마다 수정 후 블록이 그 지적을 해결했는지 판정합니다. 지적된 사실 오류·오도가 사라졌으면 resolved=true 입니다(제안과 다른 방식으로 고쳤거나, 문장을 빼거나, 조건을 붙여 해결해도 됩니다). evidence 에 수정 후 블록의 어느 부분 때문에 해결/미해결인지 한 문장으로 씁니다.
2. "이번에 바뀐 블록"만 읽고, 수정하면서 새로 생긴 blocking 문제(사실 오류·오도·코드와 출력 불일치·그대로 실행되지 않는 코드)를 new_issues 에 적습니다. 바뀐 블록 밖의 문장은 지적하지 않습니다(지적해도 코드가 버립니다). quote 는 바뀐 블록의 문장을 그대로 인용합니다. 구성·학습 활동·용어 정의 같은 교육 품질은 보지 않습니다.

JSON만 출력하세요:
{
  "summary": "한두 문장 총평",
  "previous": [{"id": 1, "resolved": true, "evidence": "판정 근거 한 문장"}],
  "new_issues": [
    {"severity": "critical|major|minor", "gate": "blocking|advisory", "kind": "factual_error|overclaim|typo",
     "section": "situation_question|principles|walkthrough|judgment_next|header", "quote": "바뀐 블록의 문제 문장 일부",
     "message": "무엇이 왜 문제인지", "suggestion": "어떻게 고칠지",
     "confidence": 0.0~1.0, "reader_impact": "독자가 그대로 믿고 따라 하면 이르는 틀린 결과·판단 한 문장"}
  ]
}`

// RecheckFinding is a previous blocking finding with the block it pointed at
// before and after the rewrite (rendered markup).
type RecheckFinding struct {
	ID                            int
	Kind, Section, Quote, Message string
	Before, After                 []string
}

// RecheckBlock is a block the rewrite changed (rendered markup).
type RecheckBlock struct {
	Section, Text string
}

// Recheck asks for a targeted re-review of a partial rewrite.
func Recheck(c LectureContext, findings []RecheckFinding, changed []RecheckBlock, set Set) string {
	var b strings.Builder
	writeContext(&b, c)
	b.WriteString("\n## 이전 blocking 지적\n")
	if len(findings) == 0 {
		b.WriteString("(없음 — 이번에 바뀐 블록만 확인합니다)\n")
	}
	for _, f := range findings {
		fmt.Fprintf(&b, "\n### 지적 %d [%s · %s]\n- 지적: %s\n", f.ID, f.Kind, f.Section, f.Message)
		if f.Quote != "" {
			fmt.Fprintf(&b, "- 문제 문장: %q\n", f.Quote)
		}
		writeBlocks(&b, "수정 전 블록", f.Before)
		writeBlocks(&b, "수정 후 블록", f.After)
	}
	b.WriteString("\n## 이번에 바뀐 블록\n")
	if len(changed) == 0 {
		b.WriteString("(없음)\n")
	}
	for i, blk := range changed {
		fmt.Fprintf(&b, "\n### 바뀐 블록 %d (%s)\n<<<BLOCK\n%s\nBLOCK>>>\n", i+1, blk.Section, strings.TrimRight(blk.Text, "\n"))
	}
	b.WriteString(set.Text(BlockReviewRecheck, c.vars()))
	writeReaderProfile(&b, c.Reader)
	writeDepthScope(&b, c.Depth)
	b.WriteString(BlockingRules)
	writeFactualRules(&b)
	return b.String()
}

func writeBlocks(b *strings.Builder, title string, blocks []string) {
	if len(blocks) == 0 {
		fmt.Fprintf(b, "- %s: (없음)\n", title)
		return
	}
	for _, t := range blocks {
		fmt.Fprintf(b, "- %s:\n<<<BLOCK\n%s\nBLOCK>>>\n", title, strings.TrimRight(t, "\n"))
	}
}

// SystemAdjudicator is the system prompt of the adjudication of one blocking
// finding (block review.adjudicate).
const SystemAdjudicator = `당신은 기술 강의의 검수 지적 하나를 재판정하는 시니어 엔지니어입니다. 검수자는 문제를 빠짐없이 찾도록 지시받아 지나치게 지적하기도 합니다. 당신은 주어진 지적 하나만 보고 "이 주장이 실제로 틀렸거나 독자를 오도하는가"를 판정합니다.
- uphold: 문장이 사실과 다르거나, 조건 없이 단정해서 독자가 그대로 믿고 따르면 틀린 결과·판단에 이릅니다. 틀린 이유를 구체적으로 댈 수 있습니다.
- dismiss: 문장이 맞거나, 이미 조건(제품·버전·"환경에 따라 다름"·"확인할 것")이 붙어 있거나, 버전·빌드·렌더러 호환을 확인할 수 없다는 이유뿐이거나, 표현·구성·교육 품질의 문제일 뿐입니다. 틀린 이유를 구체적으로 댈 수 없으면 dismiss 입니다.
출력은 JSON 하나뿐입니다: {"decision": "uphold" | "dismiss", "reason": "판정 이유 한두 문장"}`

// AdjudicationCase is one blocking finding to adjudicate: the finding and the
// block(s) holding the sentence (rendered markup).
type AdjudicationCase struct {
	Kind, Section, Quote, Message, ReaderImpact string
	Blocks                                      []string
}

// Adjudication asks whether one blocking finding stands.
func Adjudication(c LectureContext, a AdjudicationCase) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## 강의\n- 개념: %s (%s)\n- 깊이: %d (1 기본 ~ 3 심화)\n", c.ConceptName, c.ConceptID, c.Depth)
	fmt.Fprintf(&b, "\n## 검수 지적 [%s · %s]\n- 지적: %s\n", a.Kind, a.Section, a.Message)
	if a.Quote != "" {
		fmt.Fprintf(&b, "- 문제 문장: %q\n", a.Quote)
	}
	if a.ReaderImpact != "" {
		fmt.Fprintf(&b, "- 검수자가 본 독자 영향: %s\n", a.ReaderImpact)
	}
	b.WriteString("\n## 그 문장이 있는 블록\n")
	if len(a.Blocks) == 0 {
		b.WriteString("(문장을 본문에서 찾지 못했습니다. 지적 내용만으로 판정하세요)\n")
	}
	for _, t := range a.Blocks {
		fmt.Fprintf(&b, "<<<BLOCK\n%s\nBLOCK>>>\n", strings.TrimRight(t, "\n"))
	}
	b.WriteString("\n## 질문\n이 주장이 실제로 틀렸거나 독자를 오도합니까? uphold 또는 dismiss 로 답하세요.\n")
	b.WriteString("블록 끝의 \"…(검수 입력 길이 제한으로 이하 생략…)\" 표시는 입력만 줄인 것이며 강의 원문은 잘리지 않았습니다. 이를 근거로 uphold 하지 않습니다.\n")
	writeFactualRules(&b)
	return b.String()
}

// SystemReviewer is the system prompt for the review pass.
const SystemReviewer = `당신은 CS·운영 분야의 시니어 엔지니어이자 기술 감수자입니다. 칭찬하지 말고 문제만 정확히 찾습니다. 확인할 수 없는 사실은 "확인 필요"로 표시하고 단정하지 않습니다. 출력은 요청된 JSON 하나뿐입니다.`
