package prompts

// OutlineTask is the default fixed instruction of the outline step (block
// outline.task): the 20-minute plan of quality design v2.2 §3.2.
const OutlineTask = `
## 작업
위 개념으로 약 20분 분량 강의의 개요를 설계하세요. 섹션은 반드시 다음 4개, 이 순서입니다.
- situation_question (약 2분): 실무 장면 + 문제의 코드·로그(8~10줄). 원인은 아직 밝히지 않습니다. 오늘의 질문은 장면의 숫자·현상에서 끌어냅니다(목차 같은 질문 금지).
- principles (약 9분): ① 핵심 정의(목표의 핵심 명사를 정식 정의 한 문장 + 예 + 비예) ② 작은 값으로 손계산(trace) ③ 그 정의에 붙는 흔한 오해 ④ 직접 해 보기(guided: 풀이 일부를 주고 채우기)와 풀이 ⑤ 실제 규모로 확장·환경별 차이(표 권장) ⑥ 섹션 끝 회상(recall) 1~2개
- walkthrough (약 6분): 첫 장면을 그대로 재현하는 코드 → 출력 예측 멈춤 → 출력. 처방은 전후 비교(diff)와 "처방 후 같은 장애가 사라지는지" 단계별 추적. 첫 장면 코드의 한 줄을 바꾸면 출력이 어떻게 달라지는지 묻는 수정 연습(modify), 틀린 풀이에서 오류 찾기 연습(error), 섹션 끝 회상(recall) 1~2개. 수강생이 자기 터미널에서 실행해 볼 수 있는 읽기 전용 명령(포트·프로세스·메모리·DNS·HTTP 헤더 확인 등)이 있으면 실습(lab) 1개를 planned_blocks 에 넣습니다
- judgment_next (약 3분): 첫 장면의 숫자를 다시 계산해 닫기, 판단 기준(표), 관측 방법(지표의 원천·도구), 한 장 요약, 다음 개념(새 개념은 한 줄 정의 또는 링크)

JSON만 출력하세요(설명 문장 금지):
{
  "title": "강의 제목(질문형 또는 상황형, 40자 이내)",
  "new_scope": ["이번 강의에서 새로 다루는 범위(이미 배운 개념과 구분)"],
  "objective_ids": ["다룰 목표 id — 제공된 목표 전부"],
  "prerequisites_referenced": ["링크로만 참조할 기존 개념 id"],
  "thesis": "제목과 도입이 내세우는 강의 전체의 논지 한 문장",
  "thesis_check": [{"subtopic": "하위 주제", "holds": true, "why": "이 하위 주제에서도 논지가 참인 이유(거짓이면 무엇이 어긋나는지)"}],
  "key_definitions": [{"term": "목표의 핵심 명사", "definition": "정식 정의 한 문장", "example": "예", "non_example": "비예"}],
  "procedural_prereqs": [{"skill": "본문을 따라가려면 계산할 수 있어야 하는 것(예: 2진수 덧셈)", "teach": "2~3문장으로 먼저 가르칠 내용", "example": "예 하나"}],
  "scene_numbers": ["첫 장면의 숫자·현상(judgment_next 에서 다시 계산해 닫을 값)"],
  "sections": [
    {"kind": "situation_question", "title": "...", "minutes": 2, "key_points": ["..."], "planned_blocks": ["markdown", "log"]},
    {"kind": "principles", "title": "...", "minutes": 9, "key_points": ["..."], "planned_blocks": ["keypoint", "trace", "misconception", "exercise", "answer", "markdown", "recall"]},
    ...
  ],
  "misconceptions": [{"statement": "학습자가 할 법한 문장", "counterexample": "계산 가능한 반례", "model": "올바른 모델", "after_definition": "이 오해를 바로 뒤에 둘 key_definitions 의 term"}],
  "exercises": [{"section": "principles|walkthrough", "kind": "guided|why|error|free|modify", "objective_id": "목표 id", "task": "문제 한 줄"}],
  "sources": [{"title": "...", "url": "https://..."}],
  "uncertain_points": ["확신이 없어 본문에서 단정하지 않을 내용"],
  "terms_to_define": [{"term": "본문에 나올 용어·약어·도구", "where": "처음 나올 섹션", "how": "한 줄 정의 | 2~3문장 다리 | 링크(개념 id)"}]
}
규칙:
- thesis_check 에 holds=false 가 하나라도 있으면 논지를 고쳐 모든 하위 주제에서 참이 되게 한 뒤 출력합니다(예: "정수는 고정 폭"이라는 논지는 UTF-8 가변 길이 하위 주제와 모순).
- key_definitions 는 목표(objectives)의 핵심 명사 전부입니다. 비유나 별칭으로 정의를 대신하지 않습니다.
- procedural_prereqs 는 principles 첫머리에서 2~3문장 + 예 하나로 먼저 가르칩니다.
- misconceptions 는 최대 2개입니다.
- exercises 는 첫 연습이 guided, walkthrough 에 modify 1개, 마지막이 error 또는 free 입니다. why 는 2개 이하입니다.
- 분 합계는 약 20분(15~25)입니다.`

// MarkupSpec is the output-format contract for lecture bodies (block lecture.format).
const MarkupSpec = `
## 출력 형식(엄격)
아래 마크업만 출력합니다. JSON·코드펜스로 감싸지 말고, 지시 줄은 반드시 줄 맨 앞 "@@"로 시작합니다.

@@lecture
title: 강의 제목
objectives: 목표id1, 목표id2        ← 제공된 목표 id를 모두 포함
new_scope: 이번에 새로 다루는 범위 한 줄 (여러 줄 가능)
prerequisites: 링크로만 참조한 기존 개념 id들(쉼표 구분, 없으면 비움)
@@section situation_question minutes=2 title=섹션 제목
@@markdown
실무 장면(원인은 아직 밝히지 않음). 오늘의 질문은 장면의 숫자·현상에서 끌어냅니다.
처음 나오는 개념은 문장 안에서 [프로세스](term:os.process)처럼 링크하고, 같은 문장에서 한 줄로 뜻을 풀어 씁니다.
@@log lang=text
문제의 로그·코드 8~10줄
@@section principles minutes=9 title=섹션 제목
@@markdown
절차적 선행 지식(따라가려면 계산할 수 있어야 하는 것)을 2~3문장 + 예 하나로 먼저.
@@keypoint kind=definition
**핵심 명사**는 …이다(정식 정의 한 문장). 예: … 비예: …
@@trace
{"caption":"작은 값으로 따라가기","columns":["단계","값","설명"],"rows":[["1","…","…"],["2","…","…"]],"note":"…"}
@@misconception
① 학습자가 할 법한 문장 ② 계산 가능한 반례 ③ 올바른 모델
@@exercise kind=guided
풀이 일부를 주고 나머지를 채우게 하는 문제
@@answer
풀이(단계별)
@@markdown
실제 규모로 확장·환경별 차이(GFM 표 권장)
@@chart
{"kind":"line","title":"…","x":{"label":"…","values":[1,2,3]},"y":{"label":"…","scale":"linear"},"series":[{"name":"…","values":[1,4,9]}],"note":"…"}
@@recall objective=목표id
본문을 다시 보지 않고 답할 질문 한 줄
떠올릴 것: 좋은 답에 들어갈 핵심 1~2가지
@@sources
- [문서 제목](https://공식-문서-URL)
@@section walkthrough minutes=6 title=섹션 제목
@@code lang=bash
첫 장면을 그대로 재현하는 코드·명령
@@pause
위 코드를 실행하면 마지막 줄에 어떤 값이 나올까요?
@@log lang=text
실제 도구 출력 형태의 출력
@@diff lang=sql
{"before":"처방 전","after":"처방 후","note":"무엇이 왜 바뀌는지"}
@@markdown
처방 적용 후 같은 장애가 사라지는지 단계별 추적
@@lab os=macos
$ lsof -nP -iTCP:3000 -sTCP:LISTEN
보는 곳: 출력에서 무엇을 보면 되는지 한 줄
예상: 출력의 모양(실제 값은 환경마다 다름)
@@exercise kind=modify
첫 장면 코드의 한 줄을 바꾸면 출력이 어떻게 달라지는지 묻는 문제
@@answer
달라지는 출력과 그 이유
@@exercise kind=error
틀린 풀이를 보여 주고 오류를 찾게 하는 문제
@@answer
오류와 바른 풀이
@@recall objective=목표id
본문을 다시 보지 않고 답할 질문 한 줄
떠올릴 것: 좋은 답에 들어갈 핵심 1~2가지
@@sources
- [문서 제목](https://...)
@@section judgment_next minutes=3 title=섹션 제목
@@markdown
첫 장면의 숫자를 다시 계산해 닫기, 판단 기준(표), 관측 방법(지표의 원천·도구), 다음에 배울 개념([개념 이름](term:개념id) — 왜 이어지는지 한 줄씩)
@@summary
- 한 장 요약 불릿 3~6개(불릿 하나에 개념 하나)
@@end

블록 규칙:
- 섹션은 정확히 4개, 위 순서. minutes 합계 약 20(15~25).
- principles 에는 @@keypoint kind=definition 이 1개 이상, @@trace 가 1개 이상 있어야 합니다. @@keypoint 의 kind 는 definition | caution | rule 중 하나입니다.
- @@exercise 는 강의 전체에 2개 이상이고, 각 @@exercise 바로 다음 블록은 그 풀이 @@answer 입니다.
- @@pause 는 1개 이상이고 질문 한 줄입니다. 바로 다음 블록은 그 질문의 답이 되는 @@log 또는 @@code 입니다(출력을 코드 바로 아래 먼저 보여 주지 않습니다). 학습자가 한 줄로 답할 수 있게 예측 대상(값·순서·에러 여부 중 하나)을 구체적으로 묻습니다.
- @@exercise 의 kind 는 guided(풀이 일부를 주고 채우기) | why(규칙 바로 뒤 "왜 그런가요?") | error(틀린 풀이에서 오류 찾기) | free(혼자 풀기) | modify(장면 코드 한 줄을 바꾸면 출력이 어떻게 달라지나) 중 하나입니다. 첫 연습은 guided, walkthrough 에는 @@pause 뒤에 modify 1개, 마지막 연습은 error 또는 free 입니다. why 는 @@keypoint kind=rule·caution 바로 뒤에만, 강의당 2개 이하입니다.
- @@recall(방금 읽은 걸 떠올려 보기)은 principles 와 walkthrough 끝(@@sources 바로 앞)에 각각 1~2개입니다. 첫 줄은 본문을 다시 보지 않고 답할 질문 한 줄, 다음 줄은 "떠올릴 것:"으로 시작하는 좋은 답의 핵심 1~2가지입니다. objective= 에는 그 질문이 다루는 목표 id 를 씁니다. 그 섹션에서 다룬 내용만 묻고 새 사실을 더하지 않습니다.
- @@misconception 은 그 오해와 관련된 정의(@@keypoint kind=definition)나 손계산 바로 뒤에 둡니다. 섹션 끝에 몰아 두지 않습니다.
- 실행 출력 예시(@@pause 다음 출력, 코드 아래 @@log)는 언어 표준만으로 결정되는 코드에만 붙입니다. 라이브러리·DB 드라이버를 거친 값의 타입·형식은 출력 예시로 쓰지 않습니다.
- 깊이 1 강의에는 운영 마이그레이션·드라이버별 반환 타입·잠금 수준 같은 제품·버전 의존 세부를 본문 주장으로 넣지 않습니다. 필요하면 판단 기준에 "확인할 것"으로만 씁니다.
- @@summary 는 judgment_next 에 정확히 1개, 불릿 3~6개입니다.
- @@lab(선택, walkthrough 또는 judgment_next 에 0~1곳)은 수강생이 자기 터미널에서 직접 실행해 볼 읽기 전용 명령 하나입니다. 본문은 "$ 명령" 한 줄, "보는 곳:" 한 줄, "예상:" 한 줄입니다. 명령은 하나만(파이프 |, 리다이렉트 >, ;, &&, $(…), 변수, 와일드카드 금지), 다음 도구만 씁니다: lsof, ss, netstat, ps -e -o <열>, vm_stat, free, uptime, uname, getconf, ulimit, sysctl(읽기), df, du, ls, stat, file, cat /proc/<시스템 파일>, dig, nslookup, host, ping -c <≤5>, traceroute -m <≤20>, curl -sI/-o /dev/null -w(URL 하나). sudo·파일 쓰기·프로세스 종료·스크립트 실행(python, node 등)은 쓰지 않습니다. 비밀이 담길 수 있는 경로(~/.ssh, .env 등)는 쓰지 않습니다.
- @@trace / @@chart / @@diff 의 본문은 JSON 객체 하나뿐입니다(앞뒤 설명 문장 금지, 코드펜스 없이). 앱이 데이터를 그리므로 HTML·JS 는 쓰지 않습니다.
  - trace: columns 2~6개, rows 2~16개, 모든 행의 칸 수 = columns 수, 칸은 모두 문자열.
  - chart: kind line|bar, y.scale linear|log, series 1~5개, x.values 2~20개, 각 series.values 길이 = x.values 길이, 값은 유한한 수(log 이면 양수).
  - diff: before ≠ after, 각 60줄 이하, lang 지정(@@diff lang=sql).
- principles 와 walkthrough 에는 각각 @@sources 블록이 최소 1개 있어야 하며 https 공식 문서만 씁니다.
- @@code / @@log 는 lang 을 반드시 지정합니다(bash, go, sql, yaml, text 등).
- 도식(sequenceDiagram, flowchart, graph TD 등)은 반드시 따로 @@mermaid 블록에 씁니다. @@markdown 안에 mermaid 코드펜스나 도식 문법 줄을 넣으면 화면에 그림이 아니라 코드로 보입니다. @@mermaid 블록 안에는 코드펜스 없이 도식 문법만 씁니다.
- 비교표는 별도 블록 없이 @@markdown 안의 GFM 표로 씁니다.
- 개념 링크는 마크다운 문장 안의 [표시할 용어](term:개념id) 하나뿐입니다. 개념id 는 "연결 가능한 기존 개념"·"아직 배우지 않은 선행 개념"·"지식 지도의 다른 개념"·"용어 사전"(g.로 시작) 목록의 id만 쓰고, 용어가 처음 나오는 곳에 한 번만 겁니다. 이 강의의 대상 개념 자신에게는 링크하지 않습니다(그 자리에서 정의합니다). 문장 중간에 "@@term"을 쓰지 않습니다(@@ 는 줄 맨 앞 지시어 전용).
- @@ 표기는 줄의 맨 앞에서만 씁니다. 문장 안에 @@term 같은 표기를 쓰면 화면에서 그 자리가 빈칸이 되고 검증에 실패합니다.
- @@term 블록은 목록에 없는 새 개념을 제안할 때만 씁니다(proposed 와 새 slug, 영역.이름 형식, 소문자-하이픈). 섹션 끝에 몰지 말고 그 용어가 처음 나온 문단 바로 다음에 둡니다.
- 명령어 예시에서 처음 쓰는 도구와 옵션은 주석이나 바로 다음 문장에서 무엇을 하는지 한 줄로 밝힙니다.
- 마지막 줄은 반드시 @@end 입니다.

분량·문장 규칙:
- 본문 약 7,000자(공백 제외, 6,500~9,000) + 코드·로그 약 1,800자 + 위 구조 블록. 앱이 계산한 읽기 시간이 25분을 넘으면 다시 쓰게 됩니다.
- 한 문단은 5문장 이하, 한 불릿에는 개념 하나. 같은 수치의 요약은 강의 전체에서 2회 이하.
- principles 의 정식 정의(@@keypoint kind=definition)는 4개 이하이고, 정의를 셋 이상 연달아 두지 않습니다(사이에 손계산·연습·회상). 상호작용 블록(@@trace·@@pause·@@exercise·@@recall·@@chart·@@diff·@@lab) 없이 본문을 1,800자 넘게 이어 쓰지 않습니다.
- 목표의 핵심 명사는 principles 첫머리에서 @@keypoint kind=definition 으로 정식 정의합니다. 비유·별칭으로 대신하지 않습니다.
- 한 강의 한 논지: 제목과 도입의 논지가 모든 하위 주제에서 참이어야 합니다. 쓰기 전에 확인하고, 어긋나면 논지를 바꿉니다.
- 오해(@@misconception)는 ① 학습자가 할 법한 문장 ② 계산 가능한 반례 ③ 올바른 모델 순서로, 강의당 최대 2개입니다.
- 공식만 따로 제시하지 말고 작은 값으로 먼저 계산해 보입니다(@@trace).
- 금지: 생성 과정을 드러내는 문구("MDN이 아닌", "다른 강의에서 다룹니다", "이 강의 생성 규칙에 따라" 등), 내부 식별자 노출(섹션 kind 이름 situation_question·principles·walkthrough·judgment_next, 목표 id, 개념 id 를 본문 문장에 쓰지 않음), 대상 개념 자신으로의 term: 링크.`
