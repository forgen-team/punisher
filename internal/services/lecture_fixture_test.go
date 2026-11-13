package services

import "strings"

// proseFiller is reader prose of about chars characters without spaces.
func proseFiller(chars int) string {
	const sentence = "정수는 고정된 비트 폭 안에서만 값을 담습니다. "
	return strings.Repeat(sentence, chars/22+1)
}

// lectureMarkupFor is a new-format lecture (v2.2 material blocks, 20 minutes,
// prose in the §4 volume bounds) covering objectiveIDs: it passes Validate
// and lecture.MachineCheck.
func lectureMarkupFor(objectiveIDs []string) string {
	return `@@lecture
title: 32비트 정수가 넘치면
objectives: ` + strings.Join(objectiveIDs, ", ") + `
new_scope: 고정 폭 정수의 범위
@@section situation_question minutes=2 title=주문 번호가 음수가 됐다
@@markdown
` + proseFiller(1500) + `
@@section principles minutes=9 title=고정 폭 정수
@@keypoint kind=definition
**부호 없는 이진수**는 각 자리의 2의 거듭제곱을 더한 값입니다.
@@trace
{"caption":"4비트 덧셈","columns":["단계","값"],"rows":[["1","0111"],["2","1000"]]}
@@exercise
0110 + 0011 의 4비트 결과는?
@@answer
1001 입니다.
@@markdown
` + proseFiller(2500) + `
@@sources
- [doc](https://example.org/a)
@@section walkthrough minutes=6 title=장면 재현
@@code lang=sql
SELECT 2147483647::int + 1;
@@pause
위 쿼리는 무엇을 출력할까요?
@@log lang=text
ERROR:  integer out of range
@@exercise
아래 풀이에서 틀린 곳을 찾으세요.
@@answer
자리올림과 넘침을 혼동했습니다.
@@markdown
` + proseFiller(2000) + `
@@sources
- [doc](https://example.org/b)
@@section judgment_next minutes=3 title=판단과 다음
@@markdown
` + proseFiller(1000) + `
@@summary
- 고정 폭 정수는 범위가 있다
- 넘침은 자리올림과 다르다
- 시퀀스 상한을 함께 본다
@@end`
}
