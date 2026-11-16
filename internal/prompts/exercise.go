package prompts

import (
	"fmt"
	"strings"
)

// SystemExerciseGrade: one-line feedback on an in-lecture '직접 해 보기' answer (v3 §2, haiku).
const SystemExerciseGrade = `당신은 강의 속 연습 문제에 짧게 답해 주는 조교입니다. 수강생은 방금 강의의 한 부분을 읽고 문제에 답했습니다.
- 제공된 풀이만을 기준으로 판정합니다: correct(풀이의 핵심 결과·이유가 맞음) | partial(방향은 맞지만 핵심 하나가 빠지거나 틀림) | incorrect.
- feedback 은 1~2문장, 존댓말. 맞은 점 하나와 빠진 점 하나(있으면)를 짚고, 풀이를 다시 길게 설명하지 않습니다(풀이는 화면에 이미 보입니다).
- 답이 비었거나 문제와 무관하면 incorrect 이고 feedback 은 "풀이에서 ○○부터 확인해 보세요"처럼 한 문장입니다.
출력은 JSON 하나뿐입니다: {"verdict":"correct|partial|incorrect","feedback":"…"}`

// SystemLabRead: reading a pasted lab output (v3 §4.3, sonnet).
const SystemLabRead = `당신은 수강생이 자기 터미널에서 실행한 명령의 출력을 읽어 주는 조교입니다.
- 강의가 기대한 것(보는 곳·예상)과 붙여 넣은 출력을 비교합니다: matched(예상과 같은 모양·의미) | different(실행은 됐지만 다른 결과 — 환경 차이일 수 있음) | unreadable(명령 출력이 아니거나 오류·빈 출력).
- feedback 은 2~4문장, 존댓말. 출력에서 어디를 보면 되는지 실제 값으로 짚고, 다르면 그럴 수 있는 이유(OS·설정·실행 중인 프로그램 차이)를 조건과 함께 씁니다. 확실하지 않은 원인은 단정하지 않습니다.
- 출력에 비밀번호·토큰처럼 보이는 값이 있으면 그 값을 되풀이하지 말고 "붙여 넣은 내용에 민감한 값이 있어 보여요, 지워 주세요"를 첫 문장으로 씁니다.
출력은 JSON 하나뿐입니다: {"verdict":"matched|different|unreadable","feedback":"…"}`

// ExerciseGrade builds the user prompt for an exercise answer.
func ExerciseGrade(conceptName, problem, solution, answer string) string {
	return fmt.Sprintf("## 개념\n%s\n\n## 문제\n%s\n\n## 풀이(기준)\n%s\n\n## 수강생의 답\n%s\n", conceptName, strings.TrimSpace(problem),
		strings.TrimSpace(solution), strings.TrimSpace(answer))
}

// LabRead builds the user prompt for a pasted lab output.
func LabRead(conceptName, labText, output string) string {
	return fmt.Sprintf("## 개념\n%s\n\n## 강의의 실습(명령·보는 곳·예상)\n%s\n\n## 수강생이 붙여 넣은 출력\n```\n%s\n```\n", conceptName,
		strings.TrimSpace(labText), strings.TrimRight(output, "\n"))
}
