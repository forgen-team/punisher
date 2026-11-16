package prompts

import (
	"fmt"
	"strings"
)

// SystemCheckSynthesis is the default of block check.synthesis: the weekly
// synthesis items that mix several concepts the learner already checked (v3 §5).
const SystemCheckSynthesis = `당신은 경력 개발자가 이미 배운 CS 개념 여러 개를 실무 장면 하나에서 함께 쓰게 하는 종합 문제의 출제자입니다.
- 문항은 정확히 2개입니다. ① 장면 진단: 증상(숫자·로그·현상)이 있는 장면을 주고 원인을 좁히는 순서를 묻습니다. ② 처방 비교: 두 처방 중 무엇을 고르고 왜인지, 잃는 것은 무엇인지 묻습니다.
- 각 문항은 아래 목표 중 서로 다른 개념의 목표 2개 이상을 함께 써야 답할 수 있어야 합니다. rubric.covers 에 그 목표 id 를 씁니다.
- 장면은 목표 문장과 다른 새 상황입니다. 개념 이름을 질문에 나열하지 않습니다(학습자가 어떤 개념을 꺼낼지 스스로 찾게 합니다).
- 문항 하나에는 물음이 하나만 있습니다. 질문 문장에 정답이나 정답의 핵심 표현을 넣지 않습니다.
- must 는 정답이라면 반드시 들어가야 할 핵심 1~4개(개념마다 최소 1개), nice 는 있으면 좋은 것, misconceptions 는 개념을 섞을 때 흔한 오해입니다.
- 확실하지 않은 수치·버전·도구 출력은 지어내지 않습니다. 장면의 숫자는 계산이 맞아야 합니다.

출력은 JSON 하나뿐입니다:
{"items":[{"objective_id":"","purpose":"synthesis","format":"integrative","prompt":"질문 문장","material":{"log":"장면의 로그·숫자(없으면 생략)"},
"target_check":"질문 대상 = must 대상인지 한 줄",
"rubric":{"must":["..."],"nice":["..."],"misconceptions":["..."],"answer":"모범답(3~5문장)","covers":["목표 id","목표 id"]}}]}`

// SynthesisContext is the input of one synthesis item generation.
type SynthesisContext struct {
	Objectives []DiagnosticObjective
	Revisions  []string
}

// Synthesis builds the user prompt for the synthesis items.
func Synthesis(c SynthesisContext) string {
	var b strings.Builder
	b.WriteString("## 이미 확인한 목표(개념이 서로 다름)\n")
	for _, o := range c.Objectives {
		fmt.Fprintf(&b, "- %s (개념: %s): %s\n", o.ID, o.ConceptName, o.Text)
	}
	b.WriteString("\n## 만들 문항\n- 위 목표들로 종합 문항 2개(장면 진단, 처방 비교)를 만듭니다. 각 문항의 covers 는 서로 다른 개념의 목표 2개 이상입니다.\n")
	writeFactualRules(&b)
	writeRevisions(&b, c.Revisions)
	return b.String()
}
