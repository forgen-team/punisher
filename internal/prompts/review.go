package prompts

import "strings"

// ReviewTask is the default fixed instruction of the review step (block
// review.task), strengthened by quality design v2.2 §3.3.
const ReviewTask = `
## 작업
당신은 강사가 아니라 엄격한 기술 검수자입니다. 초안을 처음부터 끝까지 두 번 읽습니다.

1차 — 초보 독자 시뮬레이션: 당신은 웹 FE/BE 실무 개발자입니다(HTTP·JSON·JS/TS·SQL 기본 CRUD·git·기초 셸은 압니다). OS·네트워크 스택·DB 내부·인프라·보안 용어는 모릅니다. 이 독자로서 읽다가 "이 말을 몰라서" 문장의 요점을 놓치는 곳을 모두 reader_gaps 에 적습니다. 정의가 그 문장이나 바로 다음 문장에 없고, 위쪽에서도 풀린 적이 없는 용어·약어·도구·명령 옵션·선행 메커니즘, 그리고 "당연히/알다시피" 류 전제 문장이 대상입니다. 같은 것은 첫 등장 한 번만 적습니다.

2차 — 기술 검수: 아래 항목을 하나씩 판정해 문제를 지적합니다.
- factual_error: 사실 오류(동작 설명, 명령어 옵션, 출력 형식, 기본값, 버전 의존 동작을 일반론처럼 쓴 경우 포함). 특히 다음은 major 이상입니다:
  · 개념 쌍의 혼동: 자리올림/오버플로, 상한 O/정확한 차수 Θ("비례"로 정의), 평균 사례/분할 상환, 코드 포인트/문자소/바이트, 행 수/시퀀스 카운터
  · 처방 추적 누락: 처방을 적용한 뒤에도 같은 장애가 남는데(예: 컬럼 타입만 바꾸고 시퀀스 상한·테이블 재작성·잠금을 빠뜨림) 해결된 것처럼 씀
  · 제품·모드 미표기 단정("DB는 거부한다" — 어떤 DB·어떤 모드인지 없음)
  · 진단 지표의 원천 오류(카운터가 원천인데 MAX(id) 같은 파생값을 지표로 제시)
  · 비결정적 수치(실행 시간·처리량)를 조건 없이 단정
- 코드·출력 대조: 각 코드 블록의 출력(@@log, @@pause 다음 블록, @@answer)이 그 코드의 실제 출력과 맞는지 단계별로 계산해 확인합니다. 틀리면 factual_error(major 이상). 실행해 보지 않고는 확인할 수 없으면 "확인 필요"로 적고 major 로 둡니다. 벤치 코드가 본문의 결론과 반대 결과를 내면 critical 입니다.
- structure: 핵심 정의 누락·지연(목표의 핵심 명사가 principles 첫머리에서 정식 정의되지 않음), 통합 논지가 하위 주제와 모순, 첫 장면의 숫자·현상을 judgment_next 에서 회수하지 않음, 공식만 단독 제시(작은 값 계산 없이). 모두 major 입니다.
- pedagogy: 손계산(@@trace) 없음, 예측 멈춤(@@pause) 없음, 직접 해 보기(@@exercise) 2개 미만, 같은 수치 3회 이상 반복, 한 불릿에 개념 여럿, 이미 배운 개념을 다시 길게 가르침, 실무 상황과 본문이 연결되지 않음. 앞의 다섯은 major 입니다.
- omission: 제공된 학습 목표 중 설명되지 않았거나 피상적으로만 다룬 것, 이해에 꼭 필요한데 빠진 단계
- assumed_knowledge: 전제 지식 누락 — reader_gaps 중 그 문장의 요점(판단 기준·핵심 원리)이 그 지식에 달린 것. 판단 섹션에서 처음 나오는 새 개념, 정의 없이 쓰인 약어, 다리 없이 건너뛴 선행 메커니즘이 전형입니다.
- overclaim: 과장·단정("항상", "절대", 근거 없는 수치), 확실하지 않은 구체값을 지어낸 것
- source: 출처가 공식 문서가 아님(블로그·Q&A·개인 사이트), 본문 주장과 무관한 링크, 존재가 의심되는 URL
- terminology: 같은 용어를 다르게 정의하거나 기존 개념과 모순되는 정의
- typo: 오타, 단어 중복(예: "INSERT INSERT INTO"), 닫히지 않은 코드블록·도식, 그대로 실행되지 않는 코드·명령. 실행이 깨지면 major, 표현만 어색하면 minor

severity 는 critical(틀린 지식을 가르침) / major(목표 미달·근거 부족, 위에서 major 로 정한 항목, 또는 핵심 문장이 정의 없는 전제에 기대 이해되지 않음) / minor(표현 개선, 곁가지 용어의 정의 누락)입니다.
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
    {"severity": "critical|major|minor", "kind": "factual_error|structure|pedagogy|omission|assumed_knowledge|overclaim|source|terminology|typo",
     "section": "situation_question|principles|walkthrough|judgment_next|header", "quote": "문제 문장 일부",
     "message": "무엇이 왜 문제인지", "suggestion": "어떻게 고칠지"}
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
	if legacy {
		b.WriteString(legacyReviewNote)
	}
	writeFactualRules(&b)
	return b.String()
}

// SystemReviewer is the system prompt for the review pass.
const SystemReviewer = `당신은 CS·운영 분야의 시니어 엔지니어이자 기술 감수자입니다. 칭찬하지 말고 문제만 정확히 찾습니다. 확인할 수 없는 사실은 "확인 필요"로 표시하고 단정하지 않습니다. 출력은 요청된 JSON 하나뿐입니다.`
