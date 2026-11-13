package prompts

import (
	"fmt"
	"strings"
)

// SystemCheckPretest is the default of block check.pretest: the diagnostic
// questions asked before learning — predictions at a lecture start, the short
// '이미 앎' verification and the new-chapter placement (design §5 P3).
const SystemCheckPretest = `당신은 학습자가 배우기 전에 무엇을 알고 있는지 보는 사전 진단 문항을 쓰는 출제자입니다. 강의 본문은 없고 개념과 목표만 주어집니다.
- 목표가 다루는 핵심 하나를 묻습니다. 개념과 목표만 보고 답이 정해지는 문항만 씁니다.
- 문항 하나에는 물음이 하나만 있습니다. 질문 문장에 정답이나 정답의 핵심 표현을 넣지 않습니다.
- pretest(예측): 배우기 전에 결과·원인을 짐작하게 하는 문항입니다. format 은 choice(보기 3~4개) 또는 predict(한 줄로 답하는 예측)입니다.
- known_verify(이미 앎 확인): 이미 안다고 한 사람이 자기 말로 짧게 답할 문항입니다. format 은 recall|predict|apply, 채점 기준 must 는 1~4개입니다.
- placement(배치 진단): format 은 choice 입니다. 그 개념을 아는 사람만 고를 수 있는 문항입니다.
- choice 보기는 3~4개이고 정답은 하나입니다. 오답은 그럴듯한 흔한 오해로 씁니다. 정답 보기만 길거나 자세하지 않게, 보기끼리 길이와 말투를 맞춥니다. "모두 맞다" 같은 보기는 쓰지 않습니다.

출력은 JSON 하나뿐입니다:
{"items":[{"objective_id":"목표 id","purpose":"pretest|known_verify|placement","format":"choice|predict|recall|apply",
"prompt":"질문 문장","material":{"choices":["보기1","보기2","보기3"]},
"rubric":{"must":["..."],"nice":[],"misconceptions":["..."],"answer":"모범답(choice 는 생략 가능)","correct":0}}]}`

// DiagnosticObjective is one objective a diagnostic asks about.
type DiagnosticObjective struct {
	ID          string
	ConceptName string
	Text        string
}

// DiagnosticContext is the input of one diagnostic item generation.
type DiagnosticContext struct {
	Purpose    string // pretest | known_verify | placement
	Objectives []DiagnosticObjective
	// Count: items to write (pretest only; the others write one per objective).
	Count     int
	Revisions []string
}

var diagnosticTasks = map[string]string{
	"pretest":      "- 강의를 읽기 전 예측 문항(purpose pretest)을 정확히 %d개 만듭니다. 위 목표마다 적어도 1개, format 은 choice 또는 predict 입니다.\n",
	"known_verify": "- 목표마다 이미 앎 확인 문항(purpose known_verify) 1개씩 만듭니다. format 은 recall|predict|apply 입니다.\n",
	"placement":    "- 목표마다 배치 진단 문항(purpose placement, format choice) 1개씩 만듭니다. 위 순서(쉬운 개념 → 어려운 개념)대로 씁니다.\n",
}

// Diagnostic builds the user prompt of a diagnostic item generation (the
// check.pretest block is the system prompt).
func Diagnostic(c DiagnosticContext) string {
	var b strings.Builder
	b.WriteString("## 목표\n")
	for _, o := range c.Objectives {
		fmt.Fprintf(&b, "- %s (개념: %s): %s\n", o.ID, o.ConceptName, o.Text)
	}
	b.WriteString("\n## 만들 문항\n")
	task := diagnosticTasks[c.Purpose]
	if strings.Contains(task, "%d") {
		task = fmt.Sprintf(task, c.Count)
	}
	b.WriteString(task)
	writeRevisions(&b, c.Revisions)
	return b.String()
}
