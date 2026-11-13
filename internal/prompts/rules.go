package prompts

import "strings"

// FactualRules is the fixed factual-accuracy block of every generation prompt
// (quality design v2.2 §3.1): lecture outline/body/review, check items,
// tutor, pretest diagnostics, review variants, Q&A and explanations. It is a
// safety rule, so it is not a user-editable block: code appends it to the
// user prompt after the (possibly overridden) instruction blocks, and the
// prompt preview shows it in place.
const FactualRules = `
## 사실 정확성 규칙
- 확실하지 않은 수치·버전·기본값·명령 옵션·에러 문구는 쓰지 않는다. 쓸 때는 제품과 버전(또는 "버전에 따라 다름")을 함께 쓴다.
- 제품·설정·모드에 따라 달라지는 동작을 일반론으로 단정하지 않는다(예: "DB는 거부한다" → 어떤 DB·어떤 모드인지).
- 에러·로그는 그 에러를 실제로 내는 명령과 짝을 맞춘다. 처방을 제시하면 그 처방 적용 후 같은 장애가 사라지는지 단계별로 추적한다.
- 환경에 따라 달라지는 실행 시간·처리량은 출력 예시에 숫자로 쓰지 않거나 [예시] 라벨과 조건을 붙인다.
- 주어진 강의 본문이 있으면(문항·튜터·조교) 그 밖의 새 사실을 단정하지 않는다. 본문이 틀려 보이면 따라 쓰지 말고 표시한다.
- 구분이 필요한 개념 쌍을 섞지 않는다: 자리올림/오버플로, 상한 O/정확한 차수 Θ, 평균 사례/분할 상환, 코드 포인트/문자소(grapheme)/바이트, 행 수/시퀀스 카운터.
- 확신할 수 없는 주장은 빼거나, 조건(제품·버전·"환경에 따라 다름")을 붙이거나, "확인할 것"으로 쓴다. 단정하지 않는다.
- 실행 출력 예시는 언어 표준만으로 결정되는 코드에만 붙인다. 라이브러리·DB 드라이버가 돌려주는 값의 타입·형식(예: 드라이버가 bigint 를 문자열로 주는지)은 출력 예시로 쓰지 않고 무엇을 확인할지로 쓴다.
`

// writeFactualRules appends FactualRules to a prompt being built.
func writeFactualRules(b *strings.Builder) { b.WriteString(FactualRules) }
