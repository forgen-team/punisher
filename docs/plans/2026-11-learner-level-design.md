# 학습자 수준 · 읽기 중 질문 · 0단계/심화 확장 설계

작성 2026-11-20. 시안: https://claude.ai/artifact/TMXbbM9KyNQdmXRBSdhcpe (현재 화면 기준, 바뀌는 곳 파란 점선)

## 배경

- 백지 초보에게 너무 어렵다: 강의 독자가 "웹 FE/BE 실무자"로 고정돼 있다(`internal/prompts/lecture.go` SystemTeacher 9번, `review.go` 초보 독자 시뮬레이션).
- 이미 아는 개발자에게는 너무 쉽다: 깊이 3 개념이 전체의 약 8%(14개)이고 인프라·네트워크·보안은 0개.
- 읽다가 궁금한 것을 바로 풀기 어렵다: 구절 설명은 한 번 답하고 사라지고, 이어 묻기가 없다. 모바일에서 특히 약하다.

## 합의한 결정 (오너, 2026-11-20)

| 항목 | 결정 |
| --- | --- |
| 초보 범위 | 비개발자까지. 0단계 기초 트랙 13개 개념 추가 |
| 수준 반영 | 층 구조 — 수준은 **설명 밀도**만 바꾸고 가르치는 사실·목표는 같다. 깊이는 depth 1→2→3 다음 층 개념으로 쌓는다. 진도·복습 체계는 그대로 |
| 초기 세팅 | 설문만(1분, 진단 문항 없음). 틀린 자가 평가는 쓰면서 보정(쉽게 버튼 빈도·확인 결과), 설정에서 언제든 변경 |
| 깊이 3 | 30개 이상 추가(영역마다 약 6개) |
| 사용자 선택 | 모든 개념 정보를 보여 주되, 사용자가 "필요 없음"으로 직접 뺄 수 있다(추천·경로에서 제외, 지도에는 흐리게 남김) |
| 읽기 중 질문 | 문장 탭(모바일)·드래그(데스크톱) → 쉽게·깊게·질문·메모 → 이어 묻기 가능한 스레드, 심화 강의로 만들기 |
| 화면 | 현재 레이아웃 유지, 바뀌는 곳만 바꾼다 |
| 순서 | ① 프로필·프롬프트 → ② 0단계 YAML → ③ 깊이 3 YAML·필요 없음 → ④ 읽기 스레드 UI → ⑤ 모바일 다듬기 |

## ① 학습자 프로필 · 독자 수준 프롬프트

### 완료 조건

- 처음 연 사용자(`readerLevel` 이 빈 값)에게 강의 탭 위에 처음 설정 시트(3단계)가 뜨고, 끝내면 수준·영역별 아는 정도가 저장된다. '설명 가능' 영역의 깊이 1 개념은 이미 앎으로 표시된다. 건너뛰면 `practitioner` 로 저장된다.
- 지도 위 한 줄 안내(이미 아는 영역 표시)는 이 시트로 대체된다.
- 설정 › 학습에서 설명 수준을 바꿀 수 있다.
- 새로 만드는 강의·검수·구절 설명·질문 답·용어 풀이·확인 문항 프롬프트에 그 수준의 '독자 수준' 절이 들어간다(프롬프트 미리보기에서 보인다).
- 같은 대상이라도 수준이 다르면 새 강의를 만들고, 같은 수준이면 기존 강의를 재사용한다. 기존 강의는 `practitioner` 로 본다.
- 이전 버전 DB(001–017) 업그레이드 테스트가 통과한다.

### 계약 (서버 ↔ 웹)

`GET/PUT` 학습 설정(기존 엔드포인트)에 필드 2개 추가:

```ts
type ReaderLevel = "" | "beginner" | "novice" | "practitioner" | "advanced";
// "" = 아직 정하지 않음 → 처음 설정 시트를 띄운다. 생성은 practitioner 로 취급.
type Familiarity = "new" | "heard" | "can_explain";
interface LearningSettings {
  // …기존 필드
  readerLevel: ReaderLevel;
  areaFamiliarity: Partial<Record<PunisherArea, Familiarity>>; // 빈 객체 허용
}
```

- PUT 에서 `readerLevel` 은 `""` 이 아닌 4개 값만 받는다(잘못된 값 400 `INVALID_READER_LEVEL`). `areaFamiliarity` 의 키는 알려진 영역, 값은 3개 중 하나(400 `INVALID_AREA_FAMILIARITY`).
- 강의 응답에 `readerLevel` 필드(그 강의를 어떤 수준으로 썼는지)를 더한다.

| 수준 | 설문 문구 | 독자 가정 |
| --- | --- | --- |
| beginner | 개발은 처음이에요 | 컴퓨터·인터넷 일상 사용만. 코드·HTTP·JSON·SQL·터미널을 모른다 |
| novice | 코드를 조금 써 봤어요 | 변수·조건·반복·함수, 간단한 웹 페이지. HTTP·JSON·SQL·git·셸은 이름만 안다 |
| practitioner | 웹 개발을 실무로 해요 | 지금의 가정 그대로(HTTP·JSON·JS/TS·SQL CRUD·git·기초 셸) |
| advanced | CS 기본기는 탄탄해요 | 깊이 1 정의는 한 줄로 압축, 내부 구현·트레이드오프·장애 사례·측정 위주 |

### 서버

- 마이그레이션 018(SQLite): `user_learning_settings.reader_level TEXT NOT NULL DEFAULT ''`, `area_familiarity TEXT NOT NULL DEFAULT '{}'`; `lectures.reader_level TEXT NOT NULL DEFAULT 'practitioner'`; `generation_jobs.reader_level TEXT NOT NULL DEFAULT 'practitioner'` 와 활성 잡 부분 유니크 인덱스에 `reader_level` 추가; 대상 조회 인덱스에 `reader_level` 추가. 기존 행 변환 없음.
- 강의 조회·잡 조회(`LatestLectureForTarget`·`PublishedLectureCovering`·`ActiveJobForTarget`)에 `readerLevel` 조건을 더한다. i18n 의 `language` 와 같은 방식.
- 프롬프트: 독자 가정을 편집 가능한 블록 기본값에서 빼고, 고정 절 `prompts.ReaderProfile(level, locale)` 로 붙인다(FactualRules·DepthOneScope 와 같은 고정 규칙). 고정 절에 "가르치는 사실과 목표는 수준과 무관하게 같다. 바뀌는 것은 설명 밀도뿐"을 넣는다. 사용자가 고친 블록에 옛 가정이 남아 있어도 고정 절이 우선한다고 적는다. 검수의 '초보 독자 시뮬레이션'도 같은 수준 가정으로. 프롬프트 버전 올림.
- 영어 로캘 문구도 함께.

### 웹

- `features/punisher/first-setup`: 강의 탭 위 BottomSheet 3단계(시안 Main·Onboard2·Onboard3). 2단계 '설명 가능' 영역은 기존 `useBatchMarkKnown` 으로 깊이 1 개념 이미 앎 처리.
- `KnownOnboardingPrompt`(지도 한 줄 안내)는 처음 설정을 마친 사용자에게 띄우지 않는다(`KnownAreasSheet` 는 지도 ⋯ 메뉴에 남김).
- 설정 › 학습 맨 위(언어 아래)에 설명 수준 선택.
- 한국어·영어 카탈로그.

### 검증

- 관련 단위 테스트(설정 저장·검증, 마이그레이션 업그레이드, 수준별 강의 재사용, 프롬프트 절 포함)만 로컬에서 돌리고, 전체는 push 뒤 CI.
- 실제 앱에서 처음 설정 → 강의 생성 프롬프트 미리보기 → 설정 변경 흐름을 화면으로 확인.

## ② 0단계 기초 트랙 (13개, 새 영역 `foundations`, depth 0)

근거: The Odin Project Foundations, roadmap.sh Internet, MIT Missing Semester, CS50 앞부분.

1. 컴퓨터가 프로그램을 실행한다는 것 2. 코드의 기본 재료(변수·조건·반복·함수) 3. 데이터를 담는 형태(숫자·문자열·리스트·사전) 4. 파일·폴더·경로 5. 터미널과 셸 명령 6. 인터넷과 웹(클라이언트·서버·URL·도메인) 7. 브라우저가 하는 일(HTML·CSS·JS) 8. HTTP 요청과 응답 맛보기 9. JSON과 API 10. 표와 SQL 맛보기 11. git으로 버전 관리하기 12. 호스팅과 배포 13. 에러 메시지 읽기와 디버깅 첫걸음

- `practitioner`·`advanced` 는 기본 숨김(지도에서 펼치기), `beginner` 는 기본 경로 시작점, `novice` 는 5·8·9·10·11 부터 추천.

## ③ 깊이 3 추가 (+33, 14 → 47) · 필요 없음

| 영역 | 추가 |
| --- | --- |
| 네트워크/웹 (0→6) | TCP 혼잡 제어(CUBIC·BBR) · 연결 큐와 백로그(SYN·accept 큐, SYN 쿠키) · TIME_WAIT와 임시 포트 고갈 · 헤드 오브 라인 블로킹과 QUIC · Nagle과 지연 ACK · 애니캐스트와 BGP 기초 |
| 인프라 (0→6) | CFS 쿼터와 CPU 스로틀링 · 스케줄링·우선순위·선점·축출(QoS) · 오토스케일링(HPA)과 피드백 루프 · 성능 분석 방법론(USE·플레임 그래프) · 서비스 메시와 사이드카 · 무중단 배포와 연결 드레이닝 |
| 보안/설계 (0→6) | 위협 모델링(STRIDE) · 제로 트러스트 · TLS 1.3 핸드셰이크와 0-RTT 위험 · 타이밍 공격과 상수 시간 비교 · 리프레시 토큰 회전과 탈취 탐지 · TOCTOU와 경쟁 조건 보안 결함 |
| DB (1→6) | B-Tree vs LSM 스토리지 엔진 · 쿼리 옵티마이저와 통계 · VACUUM과 테이블 블로트 · 갭 락과 넥스트 키 락 · 복제 지연과 읽기 일관성 |
| OS (1→5) | mmap과 페이지 폴트 · eBPF와 커널 추적 · epoll 내부와 C10K · 메모리 할당자와 단편화 |
| 자료구조 (2→5) | 스킵 리스트 · 캐시 교체 정책(LRU·LFU·ARC) · 확률적 자료구조(HyperLogLog·Count-Min Sketch) |
| 컴퓨터구조 (4→6) | SIMD와 벡터화 · TLB와 큰 페이지 |
| 분산 (6→7) | 일관된 해싱 |

- 근거: DDIA, Database Internals, Systems Performance, Kubernetes 문서, OWASP Threat Modeling, NIST SP 800-207, 네트워크 면접 자료.
- 필요 없음: 개념별 `dismissed` 상태(이미 앎과 별개). 추천·목표 경로에서 제외, 지도·목록에는 흐리게 남기고 되돌릴 수 있다.

## ④ 읽기 스레드 — 계약

기존 `lecture_questions`(강의 질문)를 넓혀 구절 스레드로 쓴다. 새 표를 만들지 않는다.

- 마이그레이션 021: `lecture_questions` 에 `quote TEXT NOT NULL DEFAULT ''`(고른 구절), `parent_id TEXT`(이어 묻기면 루트 질문 id), `mode TEXT NOT NULL DEFAULT 'question'`(`easier|deeper|question`). `kind` CHECK 은 건드리지 않는다(표 재생성 회피) — 스레드 질문은 `kind='question'`.
- `POST /v1/lectures/{id}/threads` `{anchor, quote, mode, question?}` → 루트 질문을 만들고 답한다. `mode=easier|deeper` 는 질문 문장 없이 구절만으로, `question` 은 `question` 필수. 답은 짧게(300~500자) 먼저, 사용자의 독자 수준 절 적용. 응답은 `Question`(아래 필드 포함).
- `POST /v1/questions/{id}/followups` `{question?, mode?}` → 같은 스레드에 이어 묻기(직전 대화·구절·섹션 맥락 포함). `mode=easier|deeper` 는 "직전 답을 더 쉽게/깊게".
- `GET /v1/lectures/{id}/questions` 의 각 항목에 `quote`, `parentId`, `mode` 를 더한다(평평한 목록, 웹이 스레드로 묶는다).
- `POST /v1/questions/{id}/deep-lecture` → 스레드(구절 + 대화)를 질문으로 즉석 강의를 시작한다(기존 '더 깊게' 처럼 `parentLectureId`·`parentAnchor` 로 원 강의 아래에 이어 붙임). 응답은 기존 즉석 강의 시작 응답과 같은 형태. 생성은 백그라운드, 웹은 상태 줄로 보여 준다.
- 수준 보정 신호: 최근 7일 `mode=easier` 루트·이어 묻기가 5회 이상이고 수준이 beginner 가 아니며 이번 주에 제안을 닫지 않았으면, `GET /v1/me/settings` 응답에 `levelSuggestion: {to: "<한 단계 낮은 수준>", reason: "easier_often"}` 을 준다. `POST /v1/me/level-suggestion/dismiss` 로 7일간 숨긴다. 수준을 바꾸는 것은 사용자가 설정 PUT 으로.
- `Question` JSON 추가 필드: `quote: string`, `parentId: string`(루트면 ""), `mode: "easier"|"deeper"|"question"`.
- 구현에서 정한 것(2026-11-20): `levelSuggestion` 은 설정 응답 최상위(`settings` 옆, 없으면 `null`), dismiss 는 204. 이어 묻기 항목의 `quote` 는 `""`(구절은 루트에만). 쉽게/깊게 루트의 `question` 에는 요청문("더 쉽게 설명해 주세요.")을 저장. 수준 제안은 "최근 7일과 마지막 수준 변경 시각 중 늦은 쪽부터" 센다(`reader_level_changed_at`) — 낮춘 직후 같은 기록으로 또 제안하지 않게. 세는 것은 답이 난(`status=answered`) 쉽게 항목만(실패·대기 중·중복 제출은 빼서 예산 소진·장애가 신호를 부풀리지 않게). 같은 스레드 턴을 20초 안에 다시 보내면(루트: 같은 강의·구절·모드·질문, 이어 묻기: 같은 루트·모드·질문) 새 항목 없이 먼저 만든 항목을 돌려준다. 예산·키 확인은 항목을 만들기 전에 한다(거절 시 실패 항목이 남지 않음). 구절 길이 2~1000자(기존 `MaxSelectionRunes`). 오류 코드 `INVALID_THREAD_MODE`·`QUESTION_REQUIRED`·`INVALID_QUOTE`. 심화 강의는 기존 즉석 강의(자유 질의 + autoStart) 흐름을 타서, 개념 해석이 안 되면 기존 선택 화면(`needs_selection`)이 뜬다.

## ④ 읽기 스레드 — 화면 (시안 ②·③ 줄)

- 모바일(`pointer: coarse`): 본문 문장을 탭하면 그 문장이 하이라이트되고 아래 액션 바(쉽게·깊게·질문·메모). 드래그 선택의 '이 부분 설명' 버튼은 데스크톱에서만.
- 데스크톱: 드래그 선택 → 도구 막대(쉽게·깊게·질문·메모).
- 쉽게/깊게/질문 → 스레드 시트(모바일: 화면 절반 높이, 위에 고른 문장이 보이게 / 데스크톱: 오른쪽 패널 '질문' 탭에서 그 스레드를 펼침). 답 아래 칩 `더 쉽게`·`더 깊게`, 이어 묻기 입력, `이 대화로 심화 강의 만들기`.
- 메모 → 기존 노트 API(`kind=highlight`, quote 포함).
- 스레드가 있는 구절은 본문에 밑줄 + `대화 n` 표시, 누르면 그 스레드를 연다. 강의 툴바의 질문 아이콘에 개수.
- 용어 시트: 기존 설명 아래 `더 쉽게`·`이어서 묻기`(용어를 구절로 스레드 시작), `나중에 읽기`(기존 학습 요청에 추가) + 기존 `이 개념 더 깊게`.
- 수준 보정 카드: `levelSuggestion` 이 있으면 강의 화면 위에 한 줄 카드 "설명을 더 풀어 쓸까요? [바꾸기] [괜찮아요]".
- 심화 강의 생성 중이면 강의 화면 위 상태 줄(작성 ✓ → 검수 중 → 고침), 끝나면 원 강의 아래 '이어진 강의'.

## ⑤ 모바일

시안 ②·③ 줄 그대로. 본문 14→16px, 문장 탭 액션 바, '이 부분 설명' 시트에 이어 묻기·더 쉽게/깊게·심화 강의로, 용어 시트 이어 묻기·나중에 읽기, 데스크톱 질문 탭을 구절별 스레드로, 지도 목록 보기.

## ⑥ AI 선택 — 클로드 · 코덱스 · 제미나이(Antigravity)

결정(오너, 2026-11-20): 설정에서 연결된 AI 하나를 골라 강의 생성·검수·질문 답·확인 채점을 모두 그 AI 로 한다. 기본값은 지금처럼 클로드. 제미나이는 Antigravity CLI(`agy`)만 지원(2026-06-18 부터 구독 사용자는 Gemini CLI 대신 agy). 같은 PR 에 넣는다.

- 구조: `services.LLM` 인터페이스는 그대로. `internal/claudecli` 옆에 `internal/codexcli`(`codex exec`, JSONL `--json`, `--sandbox read-only`, `--skip-git-repo-check`, 프롬프트는 stdin, 시스템 프롬프트는 본문 앞에), `internal/agycli`(`agy -p`, `--output-format json|stream-json`). 공통 동시성 제한·일일 토큰 한도·오류 코드 체계는 claudecli 의 것을 재사용(공용 패키지로 뺀다).
- 선택: `internal/llm` 에 provider 라우터 — 설정의 `aiProvider`(`claude|codex|agy`, 기본 claude)를 매 호출 읽어 해당 클라이언트로 보낸다. 앱 재시작 없이 바뀐다.
- 모델 등급: 지금의 `lecture`/`fast`/`check`/`regrade` 별칭을 제공자별로 매핑. 기본은 각 CLI 의 기본 모델(빈 값)로 두고 fast 만 가벼운 모델·낮은 추론 강도, 환경 변수로 덮어쓰기(`PUNISHER_CODEX_LECTURE_MODEL` 등).
- 상태: 기존 "Claude Code 연결" 상태를 제공자별로 — 설치 여부·버전·로그인 여부(`codex login status`, agy 는 확인 가능한 명령으로). `GET` 상태 API 가 세 제공자를 모두 돌려준다.
- 설정 화면: 설정 › 고급의 'Claude Code 연결'을 'AI 연결'로 바꾸고, 설치·로그인된 것만 고를 수 있는 선택 + 각 상태 줄. 상단 경고("claude 명령을 찾을 수 없습니다")도 고른 제공자 기준.
- 강의 기록: 강의·잡에 만든 제공자를 남긴다(`lectures.provider`, 기존 `model` 옆). 재사용 조건에는 넣지 않는다(같은 수준·언어면 어떤 AI 가 썼든 재사용).
- 검증: 각 CLI 의 가짜 바이너리로 단위 테스트, 실제로는 이 Mac 에 있는 claude·codex 로 짧은 호출 한 번씩(agy 는 미설치 — 가짜 바이너리 테스트만, 미확인으로 보고).

## 체크리스트

- [x] ① 서버: 마이그레이션 018·설정·강의/잡 수준 조건·프롬프트 고정 절·테스트 (종합 문제 프롬프트 포함)
- [x] ① 웹: 처음 설정 시트·설정 수준 선택·지도 안내 대체·카탈로그 (영역 선택은 라디오 그룹, 일괄 이미 앎 훅은 entities 로 이동)
- [x] ① 중간 점검(2026-11-20): 실제 앱에서 처음 설정 → 깊이 1 이미 앎 24개 처리 → 설정 화면 수준 선택 → 프롬프트 미리보기에 수준별 절 확인, 빈 수준 저장 400, 라이트·다크
- 남은 판단: 장 설계(`SystemChapter`)·영어 출력 규칙의 "실무 개발자" 문구는 그대로 둠(장 설계는 독자 대상이 아님, 영어 규칙은 기존 영어 사용자 출력 유지)
- [x] ② 0단계 YAML(+영어 오버레이)·마이그레이션 019(depth 0..3)·수준별 추천/지도 숨김 — 실제 앱 확인(초보 1~5번, 입문 5·8·9·10·11번, 실무·심화 숨김+펼치기)
- [x] ③ 깊이 3 YAML(+영어 오버레이·용어 풀이) 33개, 제품 고유 개념은 이름에 제품 표기
- [x] ③ 필요 없음(마이그레이션 020, 추천·frontier·미리 만들기·복습 제외, 선행 잠금은 유지) · 용어 '더 깊게' 연결 22개 갱신 — 실제 앱 확인(오늘 추천 시트에서 빼기, 지도 흐린 노드·되돌리기, 좁은 폭 버튼 줄 수정, g.lru → 캐시 교체 정책)
- [x] ④ 읽기 스레드(서버·화면) — 실제 앱 확인: 데스크톱 드래그 도구 막대 → 쉽게 스레드(실제 AI 답 560자) → 이어 묻기(같은 루트로 묶임), 모바일 절반 시트·대화 n·질문 개수, 대화 n 으로 열 때 구절을 시트 위로 올리도록 수정. 심화 강의가 읽던 강의로 되돌아가던 결함 수정 → 같은 개념이면 질문 해설로 대신 만든다(status "inquiry", inquiryId). 모바일 문장 탭(pointer: coarse)은 실기기 미확인
- [x] ⑤ 모바일 다듬기 — 실제 앱(390px) 확인: 본문은 원래 16px(점검 때 14px 은 위쪽 작은 제목 줄을 잘못 잰 것) → 행간만 1.75, 아래로 스크롤 시 고정 바 한 줄, 가로 넘침 없음, 지도 폰 기본 목록 보기(층별). 실기기 터치는 미확인
- [x] ⑥ AI 선택(클로드·코덱스·agy) 서버 → 화면 — 실제 앱 확인: 설정 AI 연결에서 Codex 로 바꿈 → 강의 1편 Codex 로 생성, 첫 시도 검수 통과·발행(약 1분 35초, provider=codex). 클로드·코덱스 실제 짧은 호출 성공. agy 는 미설치라 가짜 실행 파일 테스트만(미확인)
- [x] 전체 리뷰(데이터 안전·API·AI 실행·화면 4갈래, 지적 41건) → 수정 반영, 019 스키마 동일성 실측, 최종 실제 앱 확인(코덱스로 스레드 답, 넓은 화면 닫기)
- [ ] 커밋 → PR → CI 전체 테스트
