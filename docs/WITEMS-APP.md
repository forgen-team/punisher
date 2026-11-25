# Witems 앱 모드

Punisher는 두 가지로 돈다.

| | 단독판 | Witems 앱 모드 |
|---|---|---|
| 실행 | `punisher` 바이너리(내 컴퓨터, 127.0.0.1) | 컨테이너(`PUNISHER_MODE=witems`), Witems 셸 안의 `/apps/punisher` |
| 로그인 | 없음(이 컴퓨터·Tailscale 오너만, `localOnly`) | Witems 로그인. `/v1` 은 `WITEMS_SESSION` 이 오너 계정일 때만(다른 계정 403) |
| 화면 | 내장 화면(`internal/webui`, React Router) | Module Federation remote(`internal/witems/dist`, `./mount`) — Witems 계약 `docs/APP-CONTRACT.md`(witems 저장소) |
| 생성 | 로컬 `claude` CLI | 호스트 claude-gateway(`CLAUDE_GATEWAY_URL`) |
| 저장 | SQLite(`punisher.db`) | 같음(컨테이너 볼륨 `/data`) |
| RSS 연동 | 없음 | 오늘의 글 해설, 개념의 최근 읽은 글, `/start`(RSS에서 강의 시작), 챕터 제안 신호 |

앱 안의 사용자는 두 모드 모두 `local` 한 명이다. Witems 모드는 그 앞에서 Witems 세션이 오너인지만 확인한다.

Witems 의 다른 서비스(rss-server 의 '이 개념 공부하기'·질문 만들기, mcp-server 도구)는 Docker 내부 주소(`http://witems-app-punisher:8080/v1/…`)로 `X-Internal-Token`(= `WITEMS_INTERNAL_TOKEN`) + `X-Internal-User-Id` 를 보내 부른다. 이 경로도 오너만 통과하고, nginx 가 브라우저 요청에서 이 머리글을 지우므로 밖에서는 쓸 수 없다.

## 경로 (컨테이너 :8080)

| 경로 | 내용 | 로그인 |
|---|---|---|
| `/static/…` | 화면 묶음(remoteEntry.js·청크). 이름 고정 파일은 `private, no-cache, max-age=0, must-revalidate` + ETag, 해시 청크는 1년 | 없음 |
| `/manifest.json`, `/healthz` | Witems 셸이 앱 목록을 모을 때 | 없음 |
| `/v1/…` | API | 오너 세션 |

nginx(Witems)가 `/apps-static/punisher/X` → `/static/X`, `/apps-api/punisher/v1/X` → `/v1/X` 로 잇는다(컨테이너 이름 `witems-app-punisher`).

## 환경변수

| 변수 | 필수 | 설명 |
|---|---|---|
| `PUNISHER_MODE=witems` | 예 | 이미지 기본값 |
| `AUTH_SERVICE_URL` | 예 | Witems project-setup (`/v1/auth/me`) |
| `PUNISHER_OWNER_USER_ID` | 예 | 쓸 수 있는 Witems 사용자 id |
| `PUNISHER_WITEMS_ORIGIN` | 예 | 예: `https://witems.kr`. 다른 `Origin` 요청은 403 |
| `CLAUDE_GATEWAY_URL`, `WITEMS_INTERNAL_TOKEN` | 예 | claude-gateway |
| `PUNISHER_RSS_SERVER_URL` | 아니오 | 챕터 제안의 RSS 신호(비우면 끔) |
| `PUNISHER_LOCALE` | 아니오 | 설정을 저장한 적 없는 사용자의 언어(Witems 는 `ko`) |
| `PUNISHER_LISTEN`, `PUNISHER_DATA_DIR` | 아니오 | 기본 `0.0.0.0:8080`, `/data` |

나머지 `PUNISHER_*`(모델·한도·스케줄러)는 단독판과 같다.

## 빌드

- `make witems-web` — 화면 묶음을 `internal/witems/dist` 에(컴포넌트에서 CSS 를 import 하면 셸 `<head>` 로 새므로 실패시킨다. CSS 는 `web/src/styles/globals.css` 에서).
- `docker build .` — 화면 묶음 + `punisher` + `punisher-witems-import` 이미지. CI(`.github/workflows/image.yml`)가 main·`v*` 태그를 `ghcr.io/forgen-team/punisher` 로 올린다.

## 화면을 Witems 안에서 쓸 때 지킬 것

- `document.body` 대신 `portalRoot()`(`shared/config/runtime`)에 포털을 연다.
- API 주소는 `punisherApiBase()` 로 만든다(`/v1` 을 직접 쓰지 않는다).
- `<html>`(lang·data-theme)은 셸의 것이다 — 앱 모드에서 쓰지 않는다(읽기는 된다).
- Witems 의 다른 화면(RSS 등)으로 가는 링크는 일반 `<a href>`(앱 라우터를 거치지 않음).
- 화면은 Shadow DOM 안이다. `document.getElementById`·`querySelector`·`document.activeElement` 대신 `shared/lib/dom-root`
  (`getAppElementById`·`queryApp(All)`·`activeAppElement`)를, document 에 단 리스너의 안/밖 판정은 `eventWithin`(composedPath)을,
  선택 글자는 `selectionRangeIn` 을 쓴다(`shared/lib/__tests__/shadowDom.test.tsx`).
- `window.location`(hash 포함)은 셸 주소다 — 앱 주소는 라우터(`useLocation`)로 읽고, 강의 안 자리는 `?at=<앵커>`(`lectureAnchorHref`).
- 셸 레이아웃 변수(`--app-header-offset`·`--app-bottom-offset`·`--app-topbar-h`)는 셸 것을 쓴다 — Witems 빌드가 앱 CSS 의 정의를 뺀다(`witems/scope.ts`).
- 화면 제목은 `useScreenTitle`, 페이지 동작(이동·설정)은 `usePageActions`(`shared/lib/screen-chrome`). WitemsShell 이 `ctx.setTitle`·`ctx.setActions` 로
  보낸다(동작 끝에 '학습 설정', kind `settings`). `ctx.setActions` 가 없는 구버전 셸이면 보내지 않고 탭 옆 설정 링크·강의 ⋯ 메뉴의 이동 항목으로 폴백한다.
- 화면 제목·← 는 셸이 그린다: Witems 모드에서 앱은 본문 h1(강의·해설 제목, 설정 제목)과 자기 ←(`BackLink`, 강의 상단 바 ←·제목)를 그리지 않는다.
- API 의 401 은 "세션 없음"에만 쓴다(셸이 `/login?redirect=` 로 보낸다). 오너가 아닌 계정은 403, 로그인 확인 서버 장애는 503,
  AI 제공자 인증 실패(CLI·gateway 의 401)는 503 `GEN_UNAVAILABLE`, 그 밖의 AI 실패는 502(`handlers/status_codes_test.go`).
- 확인은 `confirmAction`, 되돌리기는 `notifyUndo`(`shared/ui/kit` feedback — `FeedbackHost` 가 portal 에 그림). `window.confirm`·`alert` 는 쓰지 않는다.

## 처음 옮길 때 (Witems PostgreSQL → SQLite, 한 번)

```sh
docker run --rm --network <witems database 네트워크> -v <볼륨>:/data \
  -e WITEMS_PUNISHER_PG_URL=postgres://… -e PUNISHER_OWNER_USER_ID=<오너 id> \
  --entrypoint punisher-witems-import ghcr.io/forgen-team/punisher:<태그>
```

앱과 같은 순서(마이그레이션 → 커리큘럼 시드)로 새 DB 를 만들고, 표마다 같은 열을 옮긴다(시각 → UTC 텍스트, bool → 0/1, uuid·jsonb → 텍스트, 배열 → JSON). 오너는 `local` 로 바뀌고 다른 사용자의 행은 세어서 건너뛴다. 이미 `punisher.db` 가 있으면 아무것도 하지 않고, 실패하면 만들던 파일을 지운다.
