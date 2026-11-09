# witems ↔ 단독판 동기화

Punisher 기능은 witems(`apps/punisher-server`, `apps/web/src/**/punisher/**`)와 이 저장소에 **같이** 들어갑니다.
witems가 원본이고, 단독판은 정해진 규칙으로 옮겨 온 사본입니다. 이 문서는 이식할 때마다 따르는 순서와,
일부러 다르게 둔 부분(갈라진 곳) 목록입니다.

## 기준 커밋 기록

| 단독판 | witems 기준 커밋 | 비고 |
| --- | --- | --- |
| v0.1.0 (`0c03f16`) | `1574a0a` | 최초 이식 (NOTICE) |
| v0.2.0 | `98308e6` | 학습 요청 완료 판정·즉시 해석, '더 깊게' 연결, 강의 프롬프트 편집 |
| v0.2.1 | `dfb61eb` | 81399aa(프롬프트 2026-11-09.2·인라인 용어·펜스 검증·reader_gaps·마이그레이션 006), 7485785(하단 여백), e7b8111·5e5c77a(헤더 정리, 단독판 셸에 맞게 일부만). 제외: f192b0e(학습 요청 대기열 UI 제거), 7f4bc8a(글 해설, RSS 전용) |

다음 이식의 시작점은 **표의 마지막 witems 커밋**입니다. 이식을 마치면 이 표에 한 줄을 추가합니다.

## 순서 (체크리스트)

1. **변경 뽑기**
   ```sh
   scripts/witems-sync.sh ~/repo/witems-workspace/witems-claude <마지막 기준> <새 기준>
   ```
   범위 안에 단독판에 해당하지 않는 커밋(RSS 전용, 학습 요청 대기열 제거 등)이 섞여 있으면 범위 대신 커밋별로 뽑아 골라 적용합니다:
   `scripts/witems-sync.sh <witems> <커밋>^ <커밋> .sync/pick-<커밋>` (v0.2.1 에서 사용).
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

## 갈라진 곳 (일부러 다르게 둔 부분)

이식할 때 이 파일들은 witems 쪽 변경을 **손으로** 합칩니다.

| 파일 | 단독판에서 다른 점 | 이유 |
| --- | --- | --- |
| `internal/store/*` , `migrations/*` | SQLite 구현 | 단독판 원칙(서버·DB 설치 없음) |
| `migrations/004_lecture_links.sql` | witems 005 의 `kind IN ('deeper','regenerated')` 를 처음부터 포함 | SQLite 는 CHECK 제약을 바꿀 수 없음. v0.1.0 DB 에는 004·005 가 둘 다 새로 적용되므로 결과가 같음 |
| `internal/services/prompts.go` `libraryTarget` | `Cfg.LibraryUsesUserPrompts`(단독판 `true`)면 라이브러리 강의도 사용자 프롬프트 사용 | witems 라이브러리는 여러 사용자가 공유 → 기본값 고정. 단독판 라이브러리(추천 미리 만들기)는 그 한 사람만 읽음 |
| `internal/services/prompts_db_test.go` | 임시 SQLite 로 실행 + 위 규칙 테스트 추가 | |
| `internal/http/router.go` | 인증·CORS·rate limit 없음, `localOnly`, `/v1/status`, 웹 UI 서빙 | 단일 사용자·127.0.0.1 |
| `internal/http/handlers/learn.go` 등 오류 문구 | Claude Code 미설치/미로그인 한국어 안내(`services.DescribeLLMError`) | 게이트웨이 대신 로컬 CLI |
| `internal/claudecli/*` | claude-gateway 실행기 이식본 | witems `apps/claude-gateway` 가 바뀌면 같이 확인 |
| `web/src/widgets/punisher/prompt-settings/PromptSettingsWidget.tsx` | 안내 문구: "추천 강의 미리 만들기 포함" | 위 라이브러리 규칙 |
| `web/src/widgets/punisher/today/TodayWidget.tsx` | '학습 요청' 카드에 `AddStudyRequestForm`(직접 추가) | witems 는 RSS '30분 학습'이 요청을 만듦. 단독판엔 RSS 가 없음 |
| `web/src/widgets/punisher/today/ui/StudyRequestList.tsx` | 빈 목록 안내 문구 | 같은 이유 |
| `web/src/shared/api/punisher/endpoints.ts`, `entities/punisher/model/lectures.ts` | `createStudyRequest` / `useCreateStudyRequest` 추가 | 같은 이유 |
| `web/src/shared/api/punisher/client.ts` | 로그인 리다이렉트·API 키 안내 제거, 503(Claude 미설치/미로그인) 안내 | 인증 없음 |
| `web/src/shared/lib/{router.tsx,ThemeSync.tsx,theme.ts}`, `web/src/app`, `web/src/pages` | 단독판 셸 | Next.js 대신 Vite + React Router |
| `web/src/widgets/punisher/lecture-viewer/lib/highlight.ts` | shiki 언어 29개만 | 바이너리 크기 |
| `web/src/widgets/punisher/lecture-viewer/ui/LectureTopBar.tsx` | 뒤로(←) 버튼을 모든 폭에서 보임(witems 는 lg 이상만) | 단독판 셸 상단 바에는 [←]가 없음 |
| `web/src/widgets/punisher/nav/LearnNav.tsx` | '학습' 제목을 md 이상에서 보임(witems 는 lg) | 단독판 셸 상단 바는 화면 제목을 보이지 않음 |
| `web/src/widgets/punisher/today/ui/StudyRequestList.tsx` 외 학습 요청 UI | witems f192b0e(대기열 UI 제거)를 **가져오지 않음** | 단독판은 직접 입력 학습 요청을 유지 |
| `web/src/styles/brand.css`, `web/src/shared/ui/brand/*`, `assets/*`, `web/public/*` | forgen 패밀리 브랜딩(오렌지 액센트·로고) | 단독판 정체성. `design-tokens.css` 는 witems 원본 그대로 두고 `brand.css` 에서 액센트만 덮어쓴다 |

## 앞으로 들어올 것

- **강의 기본 프롬프트 고도화**(witems learning-polish, 전제 지식 누락 방지): witems 에 병합되면 `internal/prompts/blocks.go` 의
  기본값(`Blocks[].Default`)과 `prompts.Version` 을 witems 와 **글자 그대로** 맞춥니다. 스크립트의 `server.patch` 에 들어오므로
  2번 단계에서 그대로 적용되고, 버전 문자열이 바뀌었는지 `go test ./internal/prompts/` 로 확인합니다.
