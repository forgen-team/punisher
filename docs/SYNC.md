# witems ↔ 단독판 동기화

Punisher 기능은 witems(`apps/punisher-server`, `apps/web/src/**/punisher/**`)와 이 저장소에 **같이** 들어갑니다.
witems가 원본이고, 단독판은 정해진 규칙으로 옮겨 온 사본입니다. 이 문서는 이식할 때마다 따르는 순서와,
일부러 다르게 둔 부분(갈라진 곳) 목록입니다.

## 기준 커밋 기록

| 단독판 | witems 기준 커밋 | 비고 |
| --- | --- | --- |
| v0.1.0 (`0c03f16`) | `1574a0a` | 최초 이식 (NOTICE) |
| v0.2.0 | `98308e6` | 학습 요청 완료 판정·즉시 해석, '더 깊게' 연결, 강의 프롬프트 편집 |
| v0.3.1 | `86faccc` | (+6296884: remarkCjkStrong 을 UI kit `shared/ui/kit/lib` 로 이동, RSS 적용분 제외) |
| v0.3.1 | `bea5102` | 여백 정리(c72203f·216d16b·2357f42·266de35·d784205), Field break-keep(fcab7af), 넓은 화면 6커밋(eeab342·4a76756·d59f75d·9e50424·27e2e10·717290a), ← 규칙(6242513), 굵게 렌더(43fbb32), MermaidOutsideBlock·prompts 2026-11-09.3(bea5102). 셸 폭을 witems 학습 레이아웃(최대 1600px, px-4→fold 6→lg 8)에 맞춤. 단독판 MapControls·LearningSettingsForm 수정은 witems 가 같은 문제를 고쳐 witems 판으로 되돌림 |
| v0.3.0 (+3차) | `c37c9a6` | e9959f6: 시도 기록(마이그레이션 008, attempt_log JSON TEXT), 즉석 자동 재작성 1회(StageRewriting, 다시 시도 버튼), mermaid 본문 삼킴 검출. 단독판 전용 `TestUnknownDirectiveIsRejectedAndRegenerated`(ac4f369)는 witems 의 `TestReviewLoopAttemptLogKeepsFailedParse`·`TestInstantRewritesOnceAfterBadMarkup` 와 겹쳐 삭제 |
| v0.3.0 | `b15c724` | 50f68f6 다음 챕터(마이그레이션 007, visibleTo 사용자 격리, 챕터 고르기·지도 챕터 보기·하루 학습량), 1bc2ab4 중 UI kit(HeightReserve·시트 하단 여백)만. 제외: 534141d(학습 요청 흔적 정리 — 단독판은 요청 UI 유지), 1bc2ab4 의 RSS·오늘 홈 부분, rss-server |
| v0.2.1 | `dfb61eb` | 81399aa(프롬프트 2026-11-09.2·인라인 용어·펜스 검증·reader_gaps·마이그레이션 006), 7485785(하단 여백), e7b8111·5e5c77a(헤더 정리, 단독판 셸에 맞게 일부만). 제외: f192b0e(학습 요청 대기열 UI 제거), 7f4bc8a(글 해설, RSS 전용) |
| (다음, `mastery` 브랜치) | `01041fd` | 숙달 확인(P1)을 커밋별로 이식: b697c29(숙달 순수 규칙 `learning/mastery.go`·확인 검증 `internal/check`·확인 프롬프트 블록), 8bf187c(마이그레이션 011 SQLite 포팅·store `sqlite_check*.go`/`sqlite_mastery.go`·확인 서비스·`/v1/checks`·`/v1/objectives/{id}/relearn`·`/v1/mastery`·sweep·`PUNISHER_CHECK_*` 설정), 01041fd(확인 화면 `/check/:id`·레벨 점·지도 숙달 링). 제외: eb57da0(설계 문서 v2.1, witems `docs/plans` 전용)와 witems 작업 트리의 P2(간격 복습) 미커밋 변경. 업그레이드 테스트 `TestUpgradeFromV040`(v0.4.0/v0.5.0 DB 001–010 → 011) 추가 |
| (다음, `mastery` 브랜치) | `cc678da` | 간격 복습(P2)을 커밋별로 이식(witems `punisher-mastery` 브랜치, worktree `witems-mastery`): 7bf2bd8(순수 규칙 `learning/review.go`, 복습 서비스·`POST /v1/reviews`·`GET /v1/reviews/summary`·`/v1/today` 의 `review`·`/v1/checks/history`·`/v1/mastery/metrics`, store `sqlite_reviews.go`·`sqlite_review_history.go`·`sqlite_review_variants.go`, 변형 미리 생성은 기존 스케줄러 `tick` 에 들어감 → `PUNISHER_SCHEDULER_ENABLED` 로 켜고 끔, sweep 에서 지난 날 복습 세션 abandoned), b1786a9(`GET /v1/mastery?conceptIds=` 일괄 조회·store `ConceptsMastery`, 확인 API 400 원인 code), cc678da(복습 세션·자기 채점·오늘 복습 카드·기록 '확인·복습' 탭·기억 유지 지표·설정 복습 필드·지도 전체 숙달 링, 문서 제목 '확인·복습'). 새 마이그레이션 없음(011 에 `queued`·`reused_item` 이 이미 있음). SQLite 대체 동작 테스트 `TestReviewCountsKSTBoundary`·`TestConceptsMasteryBatch` 추가 |
| (다음, `mastery` 브랜치) | `89d7330` | 사전 진단(P3)을 커밋별로 이식(witems `punisher-mastery`, worktree `witems-mastery`): d8db54b(마이그레이션 012 SQLite 포팅 — `pretest_enabled INTEGER DEFAULT 1`·`verify_queued INTEGER DEFAULT 0`, 순수 규칙 `learning/diagnostic.go`·`check/diagnostic.go`, 예측 질문·'이미 앎' 확인·배치 진단 서비스, `POST /v1/checks` kind `pretest`·`known_verify`·`placement`(부족하면 409 `NO_PLACEMENT`), `POST /v1/concepts/{id}/known {queueVerify}`, 진단 대기열 문항은 기존 스케줄러 `tick` 에서 생성, 복습에 하루 2개까지 혼입, 설정 `pretestEnabled`, 프롬프트 블록 `check.pretest`, store `sqlite_diagnostic.go`), 89d7330(예측 질문 패널·'이미 앎' 짧은 확인·배치 진단 시작/결과·처음 예측 → 지금 답·설정 토글; 지도 상세 시트·즉석 강의 화면은 import 한 곳만 손으로). 업그레이드 테스트 `TestUpgradeFromV011`(001–011 → 012)·`TestUpgradeFromV040` 에 `pretestEnabled` 확인, SQLite 대체 동작 테스트 `TestMigration012Pretest`·`TestKnownVerifySQLiteSemantics` 추가 |
| (다음, `mastery` 브랜치) | `83a1007` | 개인화(P4)와 화면 점검 수정을 커밋별로 이식(witems `punisher-mastery`, worktree `witems-mastery`): 0e5303f(마이그레이션 013 SQLite 포팅 — `user_misconceptions` CREATE TABLE·`(user_id, turn_id)` 유니크 인덱스·status CHECK 없음, `POST /v1/misconceptions {turnId}`(201/200/404)·`GET /v1/misconceptions?status=`, 문항 결과 반영 트랜잭션(`sqlite_check_evidence.go`)에서 unaided_correct → 그 목표의 open 오개념 resolved, `learning.WeakConcept`·`Target.LectureContext`(M6 게이트: Owner 있을 때만 '다시 짚을 개념'·인정한 오개념 블록), 질문 해설·이어 묻기·조교 프롬프트에 오개념, 프롬프트 버전 2026-11-13.1, `/v1/today` 의 `recheck`(`sqlite_today.go` `lapsedConcepts`), 웹 '맞아요, 그렇게 알았어요'·'○○ 다시 확인' 카드·기록의 고친 오개념). store `sqlite_misconceptions.go`, Store 인터페이스 5개 추가. 업그레이드 테스트 `TestUpgradeFromV012`(001–012 → 013), SQLite 테스트 `TestMigration013Misconceptions`·`TestMisconceptionSQLiteSemantics` 추가. 83a1007(확인 화면 정답 한 번만·새 문항 자동 스크롤 `useFollowScroll`, 설정 PUT 400 의 원인별 code 와 웹 `describeSettingsError`, 복습 테스트 만기 `dueToday()`(KST 00~01시에도 overdue 0), store `SweepStaleChecksFor(user)`(Store 인터페이스 추가, 운영 `SweepStaleChecks` 는 전역 그대로)·테스트의 `AbandonStaleReviews(ctx, user, …)` — 단독판 `AbandonStaleReviews` 는 이미 사용자 인자를 받음). 새 마이그레이션 없음 |

| (다음, `mastery` 브랜치) | `414b61e` | 숙달 리뷰 수정·화면 점검·강의 품질 v2.2 를 커밋별로 이식(witems `punisher-mastery`, worktree `witems-mastery`). **이식**: 8ab8fa0(마이그레이션 014 SQLite 포팅 — `user_objective_mastery.last_evidence_turn_id TEXT`·`check_turns` 인덱스 2개, 이의 충돌 표지(`lockedMastery.Mark`·`saveMastery(…, mark)`·`learning.KeepSince`), 새 판정 턴 id 를 `gen_random_uuid()` 대신 `NewID()` 로 미리 발급, `FinishCheckItem`·`StartCheckTurns` 의 `otherPendingTurn`/`checkCurrentItem`(채점 중 닫기 409·커서 밖 답 거부·자기 채점 답 1개), `LoadCheckSnapshot`(repeatable-read → 트랜잭션 하나), 진단 세션의 대기 한도(`NewCheckSession.MaxPending`·`RetryCheckSession`·`pendingChecksSQL` 에 preparing 진단), `$n::uuid` → TEXT 비교(`validID` 404 는 그대로), 오개념 `left()` → `substr`·보류 턴 거부, 기억 유지 표본에서 known_verify 제외, `LoadToday(ctx, user, now)`, panic 복구, `/v1/today` 의 `recheck`(witems 가 고쳐 단독판 전용 줄 삭제)), c048e59(한국어 줄바꿈·`scrollbar-gutter: stable` 을 단독판 `web/src/styles/globals.css` 에, 토큰 `--color-bg-selected`·`--color-bg-muted`(`design-tokens.css`)·`bg-canvas-selected`·`bg-canvas-muted`(`tailwind.preset.js`)·segmented-tabs, 지도 첫 화면·결과 위치·제목만 원칙·다크 대비, 챕터 고르기 `xl:max-w-none`(단독판 `NewChapterPage.tsx` 도 같은 기준), 단독판 남은 합니다체 → 해요체(오늘 위젯 빈 상태 2곳·학습 요청 빈 목록·503 안내)), c97eab4(`ListCheckHistory` 에 known_verify, `kind=known_verify`), a731745(구조 블록 6종·생성 단계 화면·검수 상태·계산 시간·세션 합계 — `InstantStreamView` 가 witems 판과 같아져 f247dec 의 `GenerationJob.startedAt` 타입도 같이 들어옴), a38f52a(프롬프트 v2.2 — 사실 정확성 고정 문구 `prompts.FactualRules`), fd22f5b(강의 자료 블록·기계 검사·시간 계산), b813700(마이그레이션 015 SQLite 포팅 — `lectures.minutes_estimated INTEGER DEFAULT 0`·`review_requested_at TEXT`·통과 검수 부분 인덱스, 즉석 강의 검수 파이프라인 `instant_run.go`(SSE 단계 이벤트, 실패 사유는 `llmFailReason` = `DescribeLLMError`), 소급 검수 `BackfillReviews` 를 단독판 스케줄러 `tick` 에 연결·`PUNISHER_BACKFILL_REVIEW_ENABLED`(기본 켜짐, 스케줄러가 켜져 있을 때만), 확인 409 `LECTURE_NOT_REVIEWED`, `PUNISHER_LECTURE_MAX_TOKENS` 기본 24000, store `sqlite_lecture_review.go`(`RequestLectureReview`·`LecturesToReview`·`SetLectureReviewOutcome`), `contentMinutes`, 다시 배우기가 표지를 새 `NewID()` 로), 41bc848·81d4a3d(`LectureSummaries` — `= ANY` → `json_each`)·997b1a9·f4ffbf4·414b61e(연결 마무리). **제외**: 202741c(witems 설계 문서 `docs/plans`), 73fce8a(docker-compose — 단독판에 없음). 업그레이드 테스트 `TestUpgradeFromV013`(001–013 → 014 → 015, 한 릴리스씩), 단독판 테스트 `TestStandaloneLibraryKeepsFactualRules`(사용자 프롬프트를 쓰는 라이브러리 강의의 개요·본문·검수 프롬프트에 사실 정확성 규칙). 웹 렌더 테스트용 devDependencies(`@testing-library/react`·`dom`·`jest-dom`, `jsdom`)와 `vitest.config.ts` 의 jsdom·`src/vitest.setup.ts` 추가 |
| (다음, `mastery` 브랜치) | `bc3af4b` | 검수 발행 차단 기준 분리를 이식(witems `punisher-mastery`, worktree `witems-mastery`, 범위 `414b61e..bc3af4b`). **이식**: bc3af4b(검수 지적 `gate` blocking/advisory — 교육 품질 kind 는 항상 advisory, advisory 만 남은 초안 발행·advisory 반영 재작성 1회·실패 시 이전 통과본 발행, 기계 검사의 블록 구성·분량 advisory(`lecture.Issue.Advisory`), 부분 수정 재작성 `prompts.LectureRevision`·drift 비교 `services/revision_drift.go`, 깊이 1 범위 규칙·출력 예시 규칙, 시도당 초안 7분·검수 3분·즉석 30분·라이브러리 25분·stale = 최장 + 5분, `store.AttemptRecord` 에 `duration_ms`·`advisory`·`drift`·`published`(SQLite 는 attempt_log JSON TEXT 라 마이그레이션 없음), 프롬프트 2026-11-13.3), 486dca8 중 punisher-server 분(예측 질문·튜터·재채점·Q&A·용어 풀이·구절 설명을 `llm.ShortUnlessBackground(ctx)` lane 으로). **단독판 고유(손으로)**: `internal/llm/priority.go` 에 `PriorityShort`·`ShortUnlessBackground`, `internal/claudecli/limiter.go` 를 gateway 3-lane limiter 로 교체(short 전용 1자리, 동시성 3 이상), `claudecli.Config.Concurrency` 기본 2 → 3·`ShortQueue`, `PUNISHER_CONCURRENCY` 기본 3, 테스트 `TestShortCallNotBehindLongCalls`·`TestLimiterShortLane`. 테스트의 PostgreSQL 쿼리 2곳(`TestInstantKeepsPassingDraft`)은 `testDB(...).QueryRowContext` + `?` 로. **제외**: 486dca8 의 claude-gateway·go-shared 테스트·배포 문서, 설계 문서(`docs/plans`) |
| (다음, `mastery` 브랜치) | `288f66c` | 검수 수렴 구조를 이식(witems `punisher-mastery`, worktree `witems-mastery`, 범위 `bc3af4b..288f66c`). **이식**: 288f66c(2차 이후 표적 재검수 `prompts.Recheck`·블록 `review.recheck` — 이전 blocking 지적의 수정 전·후 블록과 바뀐 블록만, 바뀐 블록 밖 새 지적은 `outside_changed` 로 advisory, 바뀐 블록 50% 초과·출력 해석 불가면 `full_fallback`; blocking 지적의 `confidence`(0.7 미만 advisory)·`reader_impact` 필수와 고정 규칙 `prompts.BlockingRules`; 차단 지적 재판정 `prompts.Adjudication`·블록 `review.adjudicate`(지적당 최대 3개 병렬, short lane, LectureModel, dismiss → advisory); 재작성 지시는 차단 지적 + 남은 시도가 있을 때 advisory 최대 8줄; 즉석 강의 advisory 재작성 끔(`LoopFuncs.NoAdvisoryRewrite`); `lecture.RenderBlock`; `store.AttemptRecord` 에 `review_mode`·`recheck`·`adjudication`(SQLite attempt_log JSON TEXT 라 마이그레이션 없음); 프롬프트 2026-11-13.4). **손으로**: `store/sqlite_jobs.go` 의 `AttemptRecord` 필드, `instant_pipeline_db_test.go` 의 rejected hunk 2개(witems 판을 받아 PostgreSQL 쿼리를 `testDB(...).QueryRowContext` + `?` 로, 개념 조회·cleanup 은 단독판 그대로) |
| (다음, `mastery` 브랜치) | `7a91a8f` | **이식**: 7a91a8f(재검수·재판정 입력 블록을 표시 없이 2000자에서 자르던 문제 — 한도 8000자, 자를 때 `quotedCutMarker` 표시, 검수·재판정 프롬프트에 생략 표시는 결함 아님을 명시; 프롬프트 2026-11-13.5). 실측에서 멀쩡한 문장을 '잘림'으로 판정해 발행이 막힌 원인 |
| (다음, `mastery` 브랜치) | `382be1a` | **이식**: 382be1a(사실 계열 critical·major 지적은 검수자 gate·확신도·독자 영향과 무관하게 차단 후보, 재판정 dismiss로만 해제 — 실측에서 major overclaim이 gate advisory로 발행된 사고 대응; 프롬프트 2026-11-13.6) |
| (다음, `mastery` 브랜치) | `56bd668` | 디자인·UX 최종 재점검 23건을 이식(witems `punisher-mastery`, 커밋 하나만 `56bd668^..56bd668` — 그 앞 `56d64ac`(테스트)·`799b35e`(문서)는 단독판에 해당 없음). **서버**: 강의 전체 시간 = 섹션 분의 합(`learning.EstimateMinutes`, `WithSectionMinutes` 가 합계도 돌려줘 `GET /v1/lectures/{id}` 의 `estimatedMinutes` 를 맞춤), 이미 만든 강의가 있는 오늘 추천은 그 강의 시간(`services/today.go` `recommendedLectureMinutes`, `TodayInput.LectureMinutes`), 추천 이유 문구 순서. **웹**: 지도 첫 화면 맞춤 후보·하한 0.85·고른 개념으로 이동·남은 높이 flex·미니맵·숙달 점, trace·diff·pause·흔한 오해·keypoint·mermaid·출처·표(TableFrame), 세션 시간 줄 위치·추천 이유 2줄, 프롬프트 적용 범위 팝오버·비교, 챕터 고르기 체크박스. **손으로**: `.rej` 는 갈라진 곳 파일(KnowledgeMapWidget·TodayWidget·RecommendationRow(`item.reason`)·prompt-edit `constants.ts`·`lecture_quality_db_test.go`)만. witems `globals.css`(li·dt·dd `text-wrap: pretty`)는 `web/src/styles/globals.css` 에 손으로. witems 셸 상단 바의 한 단 상세 → 목록 뒤로가기(`navigation.ts` `detailParam`·`MobileTopBar`)는 단독판 `PromptSettingsPage`·`BackLink replace` 로 대응 |
| v0.6.1 | `910f7ea` | 확인 세션에서 카드가 하나뿐이면 카드 위 '문항 n/N'을 숨김(상단 바와 중복) — `isShownItem`·`showIndex`. 그대로 적용(`git apply -p4 --directory=web/src`, punisher 경로만). 같은 커밋의 `handoff-metrics/BarChart.tsx` 는 단독판에 없음 |
| v0.6.2 | `2b80f8d` | 다시 만든 강의가 공개되면 이어보기는 새 버전을 가리킨다(`learning.ReplaceRegenerated`·`learning/resume.go` 그대로 복사, 저장소는 `regeneratedVersions` 를 `sqlite_today.go` 에 손으로 — `::text` 없음, `rawJSON`) + 검수 배너는 새 버전이 있으면 '다시 만든 강의 보기'(`lib/newerVersion.ts` 복사, `FlaggedBanner` 는 import 줄만 단독판 경로) |

다음 이식의 시작점은 **표의 마지막 witems 커밋**입니다. 이식을 마치면 이 표에 한 줄을 추가합니다.

## 순서 (체크리스트)

1. **변경 뽑기**
   ```sh
   scripts/witems-sync.sh ~/repo/witems-workspace/witems-claude <마지막 기준> <새 기준>
   ```
   범위 안에 단독판에 해당하지 않는 커밋(RSS 전용, 학습 요청 대기열 제거 등)이 섞여 있으면 범위 대신 커밋별로 뽑아 골라 적용합니다:
   `scripts/witems-sync.sh <witems> <커밋>^ <커밋> .sync/pick-<커밋>` (v0.2.1 에서 사용).
   '갈라진 곳' 파일이 `.rej` 로 남으면 `git merge-file -L standalone -L base -L witems <단독판 파일> <witems 이전판> <witems 새판>` 로 3-way 병합한다(witems 판은 같은 변환 규칙을 거친 것, v0.3.1 에서 사용).
   `.sync/<범위>/` 에 `commits.txt` · `files.txt`(보고서용 목록) · `server.patch` · `web.patch` · `kit.patch` · `store.diff` · `web-other.txt` 가 생깁니다.
2. **서버 적용**: `git apply --recount --reject .sync/<범위>/server.patch` → `*.rej` 를 손으로 반영하고 지웁니다(대개 아래 '갈라진 곳' 파일).
3. **저장소 포팅(손으로)**: `store.diff` 를 보고
   - `postgres_*.go` → `internal/store/sqlite_*.go` (같은 함수 이름·시그니처. `Store` 인터페이스 `internal/store/store.go` 에도 추가)
   - `migrations/NNN_*.sql` → 같은 번호의 SQLite 파일. 규칙은 아래 'SQL 변환표'.
   - `postgres_*_test.go` → `sqlite_*_test.go` (임시 DB, `newTestStore`). `PUNISHER_TEST_DATABASE_URL` 로 건너뛰는 테스트를 그대로 두지 않습니다.
   - 새 마이그레이션이 있으면 `TestUpgradeFromV010` 처럼 **이전 릴리스 DB → 새 버전** 업그레이드 테스트를 늘립니다.
4. **웹 적용**: `git apply --recount --reject --directory=web .sync/<범위>/web.patch`, `git apply --recount --reject .sync/<범위>/kit.patch`.
   `web-other.txt` 의 `app/` 라우트 변경은 `web/src/pages/`·`web/src/app/App.tsx` 에 손으로 옮깁니다(Next 페이지 → React Router 라우트).
5. **확인**: `make test` (go vet·go test·typecheck·lint·vitest) → `make build` → 실제 실행(`dist/punisher --data-dir <임시> --no-browser`)과 브라우저 확인.
6. **기록**: 위 표, `CHANGELOG.md`, README 의 '바뀐 점'을 갱신하고 커밋 메시지에 witems 범위를 적습니다.

> **확인 방법**: 적용이 끝난 뒤 `git apply -R --recount --check .sync/<범위>/web.patch` 를 돌리면, 실패하는 파일은
> 아래 '갈라진 곳' 목록에 있는 파일뿐이어야 합니다. 목록에 없는 파일이 실패하면 빠진 변경이 있다는 뜻입니다.

## 자동 변환 규칙 (스크립트가 처리)

| witems | 단독판 |
| --- | --- |
| `github.com/wooo-jin/witems/apps/punisher-server` | `github.com/wooo-jin/punisher` |
| `packages/go-shared/llm`, `llmjson` | `internal/llm`, `internal/llmjson` |
| `*store.PostgresStore` | `store.Store` (인터페이스) |
| `@witems/ui` | `@/src/shared/ui/kit` |
| `next/link`, `next/navigation` | `@/src/shared/lib/router` (같은 API의 React Router 래퍼) |
| `/workspace/punisher/...` | `/...` |
| `/api/punisher/v1` | `/v1` |
| `/settings/learning`, `/settings/learning/prompts` | `/settings`, `/settings/prompts` |
| `"use client";` | 삭제 |

## SQL 변환표 (손으로)

| PostgreSQL | SQLite (이 저장소) |
| --- | --- |
| `UUID ... DEFAULT gen_random_uuid()` | `TEXT`, Go 에서 `store.NewID()` |
| `BIGSERIAL` | `INTEGER PRIMARY KEY AUTOINCREMENT` |
| `TIMESTAMPTZ DEFAULT NOW()` | `TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))` — 쿼리 안의 `NOW()` 는 store 가 자동 변환 |
| `TEXT[]`, `JSONB` | JSON `TEXT` (스캔: `strs()`, `rawJSON()`) |
| `x = ANY($1)` | `x IN (SELECT value FROM json_each($1))` |
| `arr[1]`, `cardinality(arr)` | `json_extract(arr, '$[0]')`, `json_array_length(arr)` |
| `a @> b` (배열 포함) | `NOT EXISTS (SELECT 1 FROM json_each(b) w WHERE w.value NOT IN (SELECT value FROM json_each(a)))` |
| `DISTINCT ON (k) ... ORDER BY k, t` | `ROW_NUMBER() OVER (PARTITION BY k ORDER BY t)` 후 `rn = 1` |
| `bool_and(x)` | `MIN(x) = 1` |
| `GREATEST(a, b)` | `MAX(a, b)` |
| `now() - interval '30 minutes'` | `strftime('%Y-%m-%dT%H:%M:%fZ', t, '-30 minutes')` 또는 Go 에서 계산한 시각 |
| `jsonb_path_exists(...)` | `json_each`/`json_extract` 조합 |
| `::text`, `::uuid`, `FOR UPDATE`, `pg_advisory_xact_lock` | 삭제 (연결 1개로 직렬화) |
| `ADD COLUMN IF NOT EXISTS`, `ADD CONSTRAINT` | `ADD COLUMN` 만 (마이그레이션은 한 번만 실행). 제약 변경이 필요하면 새 테이블에 처음부터 넣는다 |
| 기존 표의 `DROP/ADD CONSTRAINT`(CHECK 변경) | 표 재생성(12단계): 파일 첫 줄 `-- punisher:foreign_keys=off` → `CREATE TABLE x_vN` → `INSERT … SELECT` → `DROP TABLE x` → `RENAME` → 인덱스 재생성. 러너가 외래 키를 끄고 실행한 뒤 `foreign_key_check` 로 확인(007 이 예). 외래 키를 켠 채 `DROP TABLE` 하면 참조 행이 CASCADE 로 지워지므로 반드시 표시를 단다 |

## 갈라진 곳 (일부러 다르게 둔 부분)

이식할 때 이 파일들은 witems 쪽 변경을 **손으로** 합칩니다.

| 파일 | 단독판에서 다른 점 | 이유 |
| --- | --- | --- |
| `internal/phone/*`, `internal/http/router.go` `localOnly`, `web/public/manifest.webmanifest` | 폰 연결(Tailscale Serve, launchd). witems 에 없는 단독판 전용 | 단독판은 로그인 없이 내 Mac 에서만 돌아서, 내 기기 접속을 따로 열어야 함 |
| `internal/store/*` , `migrations/*` | SQLite 구현 | 단독판 원칙(서버·DB 설치 없음) |
| `migrations/004_lecture_links.sql` | witems 005 의 `kind IN ('deeper','regenerated')` 를 처음부터 포함 | SQLite 는 CHECK 제약을 바꿀 수 없음. v0.1.0 DB 에는 004·005 가 둘 다 새로 적용되므로 결과가 같음 |
| `internal/services/prompts.go` `libraryTarget` | `Cfg.LibraryUsesUserPrompts`(단독판 `true`)면 라이브러리 강의도 사용자 프롬프트 사용 | witems 라이브러리는 여러 사용자가 공유 → 기본값 고정. 단독판 라이브러리(추천 미리 만들기)는 그 한 사람만 읽음 |
| `internal/services/personal.go` `Target.LectureContext` (P4 M6 게이트) | 코드는 witems 와 같음 — 사용자 프롬프트를 쓰는 단독판 라이브러리도 `Owner` 를 비우므로(libraryTarget) 약한 개념 분리·인정한 오개념 블록을 넣지 않는다. `personal_db_test.go` 가 `LibraryUsesUserPrompts` false/true 둘 다 확인 | 프롬프트 블록(사용자 편집)과 개인 학습 기록은 다른 문제다. 라이브러리 강의는 저장돼 다시 읽히고 '다시 만들기' 전까지 고정이라, 그날의 약점·오개념은 즉석·다시 만들기 강의에만 반영하는 witems 결론을 그대로 둔다 |
| `internal/services/prompts_db_test.go` | 임시 SQLite 로 실행 + 위 규칙 테스트 추가 | |
| `internal/services/check.go` `checkItemKey` (check.items 문항 캐시) | 코드는 witems 와 같음(기본 블록 = 공유 키, 편집한 블록 = 그 사용자 키 `created_for`). 단독판에선 라이브러리 강의의 문항도 사용자의 check.items 블록으로 만든다 — `LibraryUsesUserPrompts` 와 같은 결론이라 별도 분기를 두지 않고 `TestStandaloneLibraryCheckItemsUseOwnerPrompts` 로 고정 | 단독판 라이브러리는 한 사람만 읽으므로 사용자 프롬프트를 써도 된다 |
| `internal/services/service.go` `Config`, `check_run.go`·`check_session.go` 실패 기록 | `Config` 끝에 `LibraryUsesUserPrompts` 가 있어 witems 의 `Check*` 필드 hunk 가 `.rej` 로 남는다(손으로 그 아래에 붙임). 확인 세션·턴의 실패 문구는 `checkErrorText`(= `DescribeLLMError`, 아니면 `err.Error()`)로 저장(4곳 + P3 진단 문항 생성 실패 1곳 — b813700 에서 `diagnostic.go` → `diagnostic_items.go` 로 옮겨짐) | 확인 화면이 `session.error` 를 그대로 보여 준다. 로컬 CLI 안내 |
| `internal/store/sqlite_check*.go`, `sqlite_mastery.go` | `pg_advisory_xact_lock`(문항 생성 키·사용자 대기 한도)·`FOR UPDATE` 는 빼고 트랜잭션 하나로(연결 1개라 직렬화). `clock_timestamp()` 는 `clockNow()`(밀리초, 호출마다 증가 — 이의 제기 충돌 표시). 유니크 위반은 제약 이름 대신 컬럼 이름(`check_turns.session_id`·`check_sessions.session_key` → `ErrCheckBusy`)으로 구분, `insertTurn` 의 savepoint 재시도 없음(`t.ID` 가 있으면 그 id 로 넣음 — 새 판정 턴 id 는 `FinishCheckItem` 이 `NewID()` 로 미리 발급). `LoadCheckSnapshot` 은 repeatable-read 대신 트랜잭션 하나, `otherPendingTurn` 의 `IS DISTINCT FROM NULLIF($2,'')::uuid` → `id <> $2`, `ReopenObjective` 의 표지 `gen_random_uuid()` → `NewID()`. `markObjectiveChanges` 는 witems 와 같은 EXISTS 문(LEAST→MIN) | SQLite 에 advisory lock·DML CTE·제약 이름 오류가 없음 |
| `internal/services/check_db_test.go`, `review_db_test.go`, `mastery_batch_db_test.go`, `internal/store/sqlite_mastery_test.go` | `Pool()` 대신 임시 SQLite(`DB()`, `?` 자리표시자), 정리(Cleanup) 없음. SQLite 대체 동작 테스트(`TestCheckStoreSQLiteSemantics`, `TestReviewCountsKSTBoundary`) 추가. 시각을 당기는 SQL 은 `NOW() - INTERVAL` 대신 `strftime(…, 'now', '-1 hour')`, `dueToday()` 의 `GREATEST(…, KST 0시::timestamptz)` 는 `MAX(strftime(…), '<store.FormatTime(KSTDayStart)>')`(같은 형식 TEXT 라 사전순 비교). `TestConceptsMasteryBatch` 는 `masteryFixture` 가 고정 id 라 다른 사용자 개념을 직접 넣는다 | |
| `internal/store/sqlite_reviews.go`, `sqlite_review_history.go`, `sqlite_review_variants.go` | `COUNT(*) FILTER` → `COUNT(CASE WHEN … THEN 1 END)`(행이 없어도 0), `DISTINCT ON` → `ROW_NUMBER() OVER (PARTITION BY objective_id …)` 후 `rn = 1`(문항 열은 다시 `JOIN check_items`), `JOIN LATERAL (… LIMIT 1)` → 상관 서브쿼리 + `IS NOT NULL`, `ReleaseQueuedReviews` 의 advisory lock·`FOR UPDATE` 는 빼고 트랜잭션 하나. 같은 `created_at` 의 순서는 `rowid` 로 정한다(`LatestCheckSession`·변형 원본). 시각은 store 형식(UTC TEXT 밀리초)으로 넘기고 KST 날짜 경계는 `learning.KSTDayStart(now)` 를 UTC 로 바꿔 비교 | SQLite 에 FILTER·DISTINCT ON·LATERAL·advisory lock 이 없음 |
| `internal/store/sqlite_misconceptions.go`, `sqlite_today.go` `lapsedConcepts` | `INSERT … ON CONFLICT DO NOTHING RETURNING` → `NewID()` 로 넣고 `RowsAffected` 로 created 판단 후 다시 읽기, `grade->>'misconception'`·`btrim` → `json_extract`·`trim`, `NULLS LAST` → `resolved_at IS NULL` 정렬 키, 같은 시각은 `rowid DESC`. 해소(`resolveMisconceptions`)는 `applyItemEvidence` 의 `conn`(같은 트랜잭션). `bool_or(…)` → `MAX(…)`(0/1, 전부 NULL 이면 `sql.NullBool`) | SQLite 에 DML RETURNING-충돌 구분·`->>`·`NULLS LAST`(구버전)·bool_or 가 없음 |
| `internal/store/sqlite_diagnostic.go` | `INSERT … RETURNING` CTE → `NewID()` 로 넣고 다시 읽기(대상 목표가 없으면 `RowsAffected` 0 → `ErrNotFound`). `jsonb_array_elements_text(item_ids)` → `json_each(item_ids)`, 챕터 초안 `jsonb_array_elements … WITH ORDINALITY` → `json_each(draft, '$.concepts')`(순서 = `key + 1`). `DISTINCT ON` → `ROW_NUMBER()`(`KnownVerifyPicks` 는 문항 열을 다시 `JOIN check_items`, 같은 시각은 `rowid`), `LEFT JOIN LATERAL (… LIMIT 1)` → 상관 서브쿼리, `KnownVerifyTargets` 의 `array_agg(… ORDER BY)` → 정렬된 행을 Go 에서 개념별로 묶고 `limit` 은 개념 수. `clock_timestamp()` → `clockNow()`(한 번의 일괄 표시에 같은 시각) | SQLite 에 LATERAL·ORDINALITY·array_agg 정렬·DML CTE 가 없음 |
| `internal/store/sqlite_lecture_review.go`, `sqlite_lectures.go`, `migrations/015_lecture_quality.sql` | `NULLS LAST` → `x IS NULL` 앞 정렬 키, `minutes_estimated` 는 INTEGER 0/1(`CASE WHEN l.minutes_estimated THEN NULL ELSE l.content END` 그대로), `LectureSummaries` 의 `id::text = ANY($1)` → `json_each` | SQLite 에 BOOLEAN·`NULLS LAST`(구버전)·배열이 없음 |
| `internal/services/check_race_db_test.go`, `instant_pipeline_db_test.go`, `lecture_quality_db_test.go`, `check_summary_db_test.go`, `internal/http/handlers/today_test.go` | `Pool()` 대신 `testDB(s)`(`?` 자리표시자), 정리 없음, `RETURNING id::text` → `NewID()` 로 넣기, `gen_random_uuid()` → `store.NewID()`, `NOW() ± INTERVAL` → `strftime(…, 'now', '±n minutes')`, `jsonb_set(… || …)` → `json_insert(…, '$[#]')`. `today_test.go` 는 임시 SQLite·인증 컨텍스트 없이 | |
| `internal/services/prompts_db_test.go` `TestStandaloneLibraryKeepsFactualRules` | 단독판 전용 테스트: `LibraryUsesUserPrompts` 로 개요·본문·검수 블록을 고친 라이브러리 강의(`runLibraryJob`)의 세 프롬프트에 사실 정확성 규칙이 붙는다 | 규칙은 코드가 붙이는 고정 문구(사용자 편집 블록 아님) |
| `internal/http/handlers/settings_test.go` `TestPutSettingsBadRequestCode` | `sharedauth.WithUserID` 없이 요청(단독판 `GetUserID` 는 고정 로컬 사용자) | 인증 없음 |
| `internal/services/personal_db_test.go` | `Pool()` 대신 `testDB(s)`(`?` 자리표시자), 정리 없음, `NOW() - INTERVAL` → `strftime(…, 'now', '-1 second')`, `user_goals` 는 `NewID()`·`json_array(?)`. 라이브러리 단언은 `LibraryUsesUserPrompts` false/true 둘 다 | |
| `internal/services/diagnostic_db_test.go` | `Pool()` 대신 `testDB(s)`(`?` 자리표시자), `placementFixture` 는 `NewID()` 로 챕터·초안을 넣고 정리 없음. 결과 JSON 은 SQLite 가 압축해 저장하므로 `"relearn":true`(witems JSONB 는 `"relearn": true`) | |
| `internal/store/sqlite_links_prompts_test.go` `TestUpgradeFromV010` | 옛 DB 시드를 `applySeed` 로만 넣는다 | 지금의 `SyncCurriculum` 은 011 표(`check_items`)가 있어야 하는 `markObjectiveChanges` 를 같이 돈다 |
| `internal/http/router.go` | 인증·CORS·rate limit 없음, `localOnly`, `/v1/status`, 웹 UI 서빙 | 단일 사용자·127.0.0.1 |
| `internal/services/instant_run.go` `llmFailReason` | 즉석 강의 LLM 실패의 사유(단계 `failed` 의 `reasons`·job.error)를 `DescribeLLMError` 한국어로, 아니면 witems 와 같은 `ReasonGenerationError` | 로컬 CLI 안내(미설치·미로그인·한도) |
| `internal/http/handlers/learn.go` 등 오류 문구 | Claude Code 미설치/미로그인 한국어 안내(`services.DescribeLLMError`) | 게이트웨이 대신 로컬 CLI |
| `internal/claudecli/*` | claude-gateway 실행기 이식본 | witems `apps/claude-gateway` 가 바뀌면 같이 확인 |
| `web/src/features/punisher/prompt-edit/ui/constants.ts` `EDITOR_LABELS.appliesTo` | 제목 툴팁: "추천 강의 미리 만들기에도 적용, 이미 만든 강의는 그대로"(witems "라이브러리 강의는 기본값 그대로"). c048e59 로 `PromptSettingsWidget` 의 안내 상자가 없어져 그 위젯은 witems 와 같다 | 위 라이브러리 규칙 |
| `web/src/widgets/punisher/today/TodayWidget.tsx` | '학습 요청' 카드에 `AddStudyRequestForm`(직접 추가) | witems 는 RSS '30분 학습'이 요청을 만듦. 단독판엔 RSS 가 없음 |
| `web/src/widgets/punisher/today/ui/StudyRequestList.tsx` | 빈 목록 안내 문구(해요체) | 같은 이유 |
| `web/src/shared/api/punisher/endpoints.ts`, `entities/punisher/model/lectures.ts` | `createStudyRequest` / `useCreateStudyRequest` 추가 | 같은 이유 |
| `web/src/shared/api/punisher/client.ts` | 로그인 리다이렉트·API 키 안내 제거, 503(Claude 미설치/미로그인) 안내(해요체) | 인증 없음 |
| `web/src/shared/lib/{router.tsx,ThemeSync.tsx,theme.ts}`, `web/src/app`, `web/src/pages` | 단독판 셸 | Next.js 대신 Vite + React Router |
| `web/src/widgets/punisher/lecture-viewer/lib/highlight.ts` | shiki 언어 29개만 | 바이너리 크기 |
| `web/src/widgets/punisher/lecture-viewer/ui/LectureTopBar.tsx` | 뒤로(←) 버튼을 모든 폭에서 보임(witems 는 lg 이상만). 들어온 곳으로 가는 규칙(`canGoBack`)은 witems 와 같음 | 단독판 셸 상단 바에는 [←]가 없음 |
| `web/src/widgets/punisher/nav/LearnNav.tsx` | '학습' 제목을 md 이상에서 보임(witems 는 lg) | 단독판 셸 상단 바는 화면 제목을 보이지 않음 |
| `web/src/widgets/punisher/today/ui/StudyRequestList.tsx` 외 학습 요청 UI | witems f192b0e(대기열 UI 제거)·534141d(지도·추천의 학습 요청 흔적 정리)를 **가져오지 않음** — 지도의 '요청됨' 표시·`?request=`·대기 목록 유지 | 단독판은 직접 입력 학습 요청을 유지 |
| `web/src/widgets/punisher/today/TodayWidget.tsx`, `today/ui/index.ts` | witems 판에서 '글 해설' 자리(3단의 셋째 열)를 '학습 요청'(AddStudyRequestForm + StudyRequestList)으로 바꾼 것만 다름(import·주석 포함). `ui/index.ts` 는 `ExplanationRequestList` 대신 `StudyRequestList` 를 내보낸다. P2 의 복습 카드(`ReviewCard`·`reviewCardState`)와 P4 의 '다시 확인' 줄(`RecheckRow`·`todayRows`)은 witems 와 같다(P4 는 import 두 줄·`ui/index.ts` 내보내기 한 줄만 손으로). 빈 상태 두 곳('추천할 다음 강의가 없어요'·'오늘 추천이 없어요')은 해요체(witems 는 아직 합니다체 — witems 가 고치면 같아짐). a731745 의 `sessionTimeLabel` import 는 witems 와 같다 | 이식 때는 witems 판을 가져와 그 한 곳만 바꾸는 편이 3-way 병합보다 쉽다(P2 는 `git merge-file` 로 해도 import 한 곳만 충돌) |
| `web/src/widgets/punisher/knowledge-map/KnowledgeMapWidget.tsx`, `ui/ConceptDetailSheet.tsx`, `ui/ConceptNode.tsx`, `ui/MapCanvas.tsx` | lg+ 위젯 높이 `100dvh-14rem`(witems `11.25rem` — 단독판 셸의 상단 바·main 여백이 다름). 요청 관련(`PendingRequestList`, `requests` prop, `?request=`, 노드의 `requested`·'학습 요청됨') 줄만 다름. 숙달 링은 witems 와 같이 `useMasteryBatch`(전체 노드) → `masteredIds`. P3 의 '이미 앎' 짧은 확인(`useKnownChoice`·`KnownVerifyChoice`)은 witems 와 같다(import 줄만 `MapRequest` 때문에 손으로) | 이식 때 `git merge-file`(단독판 HEAD / witems 이전 / witems 새판)로 3-way 병합 |
| `cmd/punisher/main.go`, `internal/config` (확인 모델) | `PUNISHER_CHECK_MODEL`(기본 = 강의 모델 `sonnet`), `PUNISHER_CHECK_REGRADE_MODEL`(기본 = 채점에 안 쓰는 쪽, 보통 `haiku`), `PUNISHER_DAILY_CHECK_TOKEN_LIMIT`(0 = 끔). witems 의 Anthropic 공급자 분기는 없음 | 단독판은 로컬 claude CLI 하나 |
| `web/src/pages/LearnPages.tsx` `CheckPage`, `web/src/app/App.tsx`, `AppShell.tsx` | witems `app/(main)/workspace/punisher/check/[id]/page.tsx` → 라우트 `/check/:id`, 문서 제목 '확인·복습'(P2: 확인·복습 세션 공용). ← 는 `CheckView` 의 `LectureTopBar`(강의로)가 맡는다 | Next 페이지 → React Router |
| `cmd/punisher/main.go`, `internal/config` | RSS 신호(`Signals`, `RSSServerURL`) 연결 없음 → `Service.Signals == nil` 이라 다음 챕터 제안의 RSS 관심사·트렌드(c)는 꺼짐 | 단독판에는 RSS 서버가 없음 |
| `web/src/widgets/punisher/lecture-viewer/LectureViewer.tsx` | 강의를 못 불러올 때 '강의 화면으로 돌아가기' 링크 | 단독판 셸에 뒤로 버튼이 없음 |
| `web/src/app/AppShell.tsx`, `web/src/shared/ui/BackLink.tsx`, `web/src/pages/*` | 셸 폭·좌우 여백은 witems 학습 레이아웃과 같음(1600px, px-4/fold 6/lg 8). 아래 여백은 셸 한 곳. 하위 화면 맨 위 '← 상위'(witems DesktopBackLink 와 같은 들어온 곳 규칙, 단 모든 폭에서 보임). `PromptSettingsPage` 는 한 단 배치에서 `?block=` 이 있으면 '← 강의 프롬프트'(`BackLink replace`, 목록으로) — witems 는 셸 상단 바(`navigation.ts` `detailParam`)가 맡음 | 단독판 셸에 상단 바 ← 가 없음 |
| `web/src/styles/globals.css` | witems `apps/web/app/globals.css` 의 Punisher 관련 전역 규칙(c048e59: `scrollbar-gutter: stable`, body `word-break: keep-all`·`overflow-wrap: anywhere`, 제목 `text-wrap: balance`·문단 `pretty`, 코드 `word-break: normal`)을 손으로 옮김. 스크립트가 뽑지 않는 파일이라 witems 의 globals.css 변경은 따로 확인한다 | 단독판 셸 CSS |
| `web/src/widgets/punisher/knowledge-map/ui/MapPanels.tsx` | '분류 중인 학습 요청' 패널(`PendingRequestList`)과 그 설명 문구. 영역별 커버리지·다음 공부 패널은 witems 와 같이 제목만(커버리지 뜻은 `COVERAGE_HINT` 툴팁) | 단독판 학습 요청 UI |
| `web/package.json`, `web/vitest.config.ts`, `web/src/vitest.setup.ts` | witems 렌더 테스트(`@testing-library/react`·`jest-dom`)를 그대로 돌리려고 devDependencies 와 jsdom 환경·정리(setup)를 둔다 | witems `createBaseVitestConfig` 대응 |
| `web/src/widgets/punisher/check-view/__tests__/resultPosition.test.tsx`, `knowledge-map/__tests__/conceptLectures.test.tsx` | `vi.mock("next/navigation")` 대신 `vi.mock("@/src/shared/lib/router")` 로 `useRouter`·`Link`(일반 `<a>`)를 바꾼다 — 단독판 `Link` 는 React Router 라 라우터 없이 렌더하면 실패 | next/link 는 라우터 없이 렌더됨 |
| `web/src/styles/brand.css`, `web/src/shared/ui/brand/*`, `assets/*`, `web/public/*` | forgen 패밀리 브랜딩(오렌지 액센트·로고) | 단독판 정체성. `design-tokens.css` 는 witems 원본 그대로 두고 `brand.css` 에서 액센트만 덮어쓴다 |

## 앞으로 들어올 것

- **강의 기본 프롬프트 고도화**(witems learning-polish, 전제 지식 누락 방지): witems 에 병합되면 `internal/prompts/blocks.go` 의
  기본값(`Blocks[].Default`)과 `prompts.Version` 을 witems 와 **글자 그대로** 맞춥니다. 스크립트의 `server.patch` 에 들어오므로
  2번 단계에서 그대로 적용되고, 버전 문자열이 바뀌었는지 `go test ./internal/prompts/` 로 확인합니다.
