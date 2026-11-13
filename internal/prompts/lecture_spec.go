package prompts

// OutlineTask is the default fixed instruction of the outline step (block
// outline.task): the 20-minute plan of quality design v2.2 §3.2.
const OutlineTask = `
## 작업
위 개념으로 약 20분 분량 강의의 개요를 설계하세요. 섹션은 반드시 다음 4개, 이 순서입니다.
- situation_question (약 2분): 실무 장면 + 문제의 코드·로그(8~10줄). 원인은 아직 밝히지 않습니다. 오늘의 질문은 장면의 숫자·현상에서 끌어냅니다(목차 같은 질문 금지).
- principles (약 9분): ① 핵심 정의(목표의 핵심 명사를 정식 정의 한 문장 + 예 + 비예) ② 작은 값으로 손계산(trace) ③ 직접 해 보기(숫자만 바꾼 변형)와 풀이 ④ 실제 규모로 확장·환경별 차이(표 권장) ⑤ 흔한 오해
- walkthrough (약 6분): 첫 장면을 그대로 재현하는 코드 → 출력 예측 멈춤 → 출력. 처방은 전후 비교(diff)와 "처방 후 같은 장애가 사라지는지" 단계별 추적. 틀린 풀이에서 오류 찾기 연습
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
    {"kind": "principles", "title": "...", "minutes": 9, "key_points": ["..."], "planned_blocks": ["keypoint", "trace", "exercise", "answer", "markdown", "misconception"]},
    ...
  ],
  "misconceptions": [{"statement": "학습자가 할 법한 문장", "counterexample": "계산 가능한 반례", "model": "올바른 모델"}],
  "sources": [{"title": "...", "url": "https://..."}],
  "uncertain_points": ["확신이 없어 본문에서 단정하지 않을 내용"],
  "terms_to_define": [{"term": "본문에 나올 용어·약어·도구", "where": "처음 나올 섹션", "how": "한 줄 정의 | 2~3문장 다리 | 링크(개념 id)"}]
}
규칙:
- thesis_check 에 holds=false 가 하나라도 있으면 논지를 고쳐 모든 하위 주제에서 참이 되게 한 뒤 출력합니다(예: "정수는 고정 폭"이라는 논지는 UTF-8 가변 길이 하위 주제와 모순).
- key_definitions 는 목표(objectives)의 핵심 명사 전부입니다. 비유나 별칭으로 정의를 대신하지 않습니다.
- procedural_prereqs 는 principles 첫머리에서 2~3문장 + 예 하나로 먼저 가르칩니다.
- misconceptions 는 최대 2개입니다.
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
@@exercise
숫자만 바꾼 변형 문제
@@answer
풀이(단계별)
@@markdown
실제 규모로 확장·환경별 차이(GFM 표 권장)
@@chart
{"kind":"line","title":"…","x":{"label":"…","values":[1,2,3]},"y":{"label":"…","scale":"linear"},"series":[{"name":"…","values":[1,4,9]}],"note":"…"}
@@misconception
① 학습자가 할 법한 문장 ② 계산 가능한 반례 ③ 올바른 모델
@@sources
- [문서 제목](https://공식-문서-URL)
@@section walkthrough minutes=6 title=섹션 제목
@@code lang=bash
첫 장면을 그대로 재현하는 코드·명령
@@pause
위 코드를 실행하면 무엇이 출력될까요?
@@log lang=text
실제 도구 출력 형태의 출력
@@diff lang=sql
{"before":"처방 전","after":"처방 후","note":"무엇이 왜 바뀌는지"}
@@markdown
처방 적용 후 같은 장애가 사라지는지 단계별 추적
@@exercise
틀린 풀이를 보여 주고 오류를 찾게 하는 문제
@@answer
오류와 바른 풀이
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
- @@pause 는 1개 이상이고 질문 한 줄입니다. 바로 다음 블록은 그 질문의 답이 되는 @@log 또는 @@code 입니다(출력을 코드 바로 아래 먼저 보여 주지 않습니다).
- @@summary 는 judgment_next 에 정확히 1개, 불릿 3~6개입니다.
- @@trace / @@chart / @@diff 의 본문은 JSON 객체 하나뿐입니다(앞뒤 설명 문장 금지, 코드펜스 없이). 앱이 데이터를 그리므로 HTML·JS 는 쓰지 않습니다.
  - trace: columns 2~6개, rows 2~16개, 모든 행의 칸 수 = columns 수, 칸은 모두 문자열.
  - chart: kind line|bar, y.scale linear|log, series 1~5개, x.values 2~20개, 각 series.values 길이 = x.values 길이, 값은 유한한 수(log 이면 양수).
  - diff: before ≠ after, 각 60줄 이하, lang 지정(@@diff lang=sql).
- principles 와 walkthrough 에는 각각 @@sources 블록이 최소 1개 있어야 하며 https 공식 문서만 씁니다.
- @@code / @@log 는 lang 을 반드시 지정합니다(bash, go, sql, yaml, text 등).
- 도식(sequenceDiagram, flowchart, graph TD 등)은 반드시 따로 @@mermaid 블록에 씁니다. @@markdown 안에 mermaid 코드펜스나 도식 문법 줄을 넣으면 화면에 그림이 아니라 코드로 보입니다. @@mermaid 블록 안에는 코드펜스 없이 도식 문법만 씁니다.
- 비교표는 별도 블록 없이 @@markdown 안의 GFM 표로 씁니다.
- 개념 링크는 마크다운 문장 안의 [표시할 용어](term:개념id) 하나뿐입니다. 개념id 는 "연결 가능한 기존 개념"·"아직 배우지 않은 선행 개념"·"지식 지도의 다른 개념" 목록의 id만 쓰고, 용어가 처음 나오는 곳에 한 번만 겁니다. 이 강의의 대상 개념 자신에게는 링크하지 않습니다(그 자리에서 정의합니다). 문장 중간에 "@@term"을 쓰지 않습니다(@@ 는 줄 맨 앞 지시어 전용).
- @@ 표기는 줄의 맨 앞에서만 씁니다. 문장 안에 @@term 같은 표기를 쓰면 화면에서 그 자리가 빈칸이 되고 검증에 실패합니다.
- @@term 블록은 목록에 없는 새 개념을 제안할 때만 씁니다(proposed 와 새 slug, 영역.이름 형식, 소문자-하이픈). 섹션 끝에 몰지 말고 그 용어가 처음 나온 문단 바로 다음에 둡니다.
- 명령어 예시에서 처음 쓰는 도구와 옵션은 주석이나 바로 다음 문장에서 무엇을 하는지 한 줄로 밝힙니다.
- 마지막 줄은 반드시 @@end 입니다.

분량·문장 규칙:
- 본문 약 8,500자(공백 제외, 6,500~11,000) + 코드·로그 약 2,500자 + 위 구조 블록.
- 한 문단은 5문장 이하, 한 불릿에는 개념 하나. 같은 수치의 요약은 강의 전체에서 2회 이하.
- 목표의 핵심 명사는 principles 첫머리에서 @@keypoint kind=definition 으로 정식 정의합니다. 비유·별칭으로 대신하지 않습니다.
- 한 강의 한 논지: 제목과 도입의 논지가 모든 하위 주제에서 참이어야 합니다. 쓰기 전에 확인하고, 어긋나면 논지를 바꿉니다.
- 오해(@@misconception)는 ① 학습자가 할 법한 문장 ② 계산 가능한 반례 ③ 올바른 모델 순서로, 강의당 최대 2개입니다.
- 공식만 따로 제시하지 말고 작은 값으로 먼저 계산해 보입니다(@@trace).
- 금지: 생성 과정을 드러내는 문구("MDN이 아닌", "다른 강의에서 다룹니다", "이 강의 생성 규칙에 따라" 등), 내부 식별자 노출(섹션 kind 이름 situation_question·principles·walkthrough·judgment_next, 목표 id, 개념 id 를 본문 문장에 쓰지 않음), 대상 개념 자신으로의 term: 링크.`
