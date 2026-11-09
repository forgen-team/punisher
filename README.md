# Punisher

Claude Code로 나만의 CS 기본기 강의를 만들어 읽는 학습 앱입니다.
오늘 공부할 개념을 추천받고, 강의를 실시간으로 생성해 읽고, 모르는 용어나 문장을 바로 물어볼 수 있어요.
지식 지도에서 8개 영역(자료구조 · 컴퓨터구조 · OS · 네트워크/웹 · DB · 분산 · 인프라 · 보안/설계), 168개 개념의 진도를 한눈에 봅니다.

- 내 Mac에서만 돌아갑니다. 서버·Docker·DB 설치나 API 키, 회원가입은 필요 없어요.
- 강의 생성과 질문 답변에는 **내 Mac에 로그인된 Claude Code**(`claude` 명령)를 씁니다. 사용량은 내 Claude 구독에서 차감돼요.

## 1. 준비물

1. macOS (Apple Silicon / Intel 모두 됨)
2. **Claude Code** 설치 + 로그인
   ```sh
   curl -fsSL https://claude.ai/install.sh | bash
   claude          # 처음 한 번 실행해서 안내대로 로그인한 뒤 /exit 로 나오기
   ```
   `claude auth status` 를 실행했을 때 `"loggedIn": true` 가 나오면 준비 끝입니다.

## 2. 설치

1. 받은 `punisher` 파일(또는 압축을 푼 폴더)을 준비합니다.
2. 터미널에서 그 폴더로 이동해 설치 스크립트를 실행합니다.
   ```sh
   sh install.sh            # punisher 를 ~/bin 에 복사하고 PATH 를 설정
   ```
   스크립트 없이 직접 해도 됩니다.
   ```sh
   mkdir -p ~/bin && cp punisher ~/bin/ && chmod +x ~/bin/punisher
   xattr -d com.apple.quarantine ~/bin/punisher   # "확인되지 않은 개발자" 경고 방지
   ```

## 3. 실행

```sh
punisher
```

- 브라우저가 자동으로 `http://127.0.0.1:4710` 을 엽니다 (4710이 사용 중이면 빈 포트를 골라 그 주소를 터미널에 보여 줘요).
- 이미 켜져 있으면 새로 켜지 않고 열려 있는 화면을 다시 엽니다.
- 끄려면 실행한 터미널 창에서 `Ctrl+C`.
- 휴대폰처럼 좁은 화면에서도 쓸 수 있게 만들었지만, 보안상 **이 Mac에서만** 접속됩니다(127.0.0.1).

옵션:

| 옵션 | 설명 |
| --- | --- |
| `--port 4800` | 포트 지정 |
| `--no-browser` | 브라우저 자동 열기 끄기 |
| `--data-dir 경로` | 데이터 폴더 바꾸기 |
| `--version` | 버전 확인 |

## 4. 사용법

- **강의** 탭: 오늘의 추천을 누르고 `강의 시작` → 강의가 실시간으로 써지고, 다 되면 저장돼요. 위 입력창에 궁금한 것(예: "Pod가 죽었다")을 써도 됩니다.
- 강의 안에서 밑줄 친 **용어**를 누르면 짧은 설명이, 문장을 **드래그해 선택**하면 `이 부분 설명` 버튼이 나옵니다. 오른쪽 위 말풍선에서 질문·메모를 남길 수 있어요.
- 다 읽으면 맨 아래 `완료` (이미 아는 내용이면 `이미 앎`).
- **지도** 탭: 전체 개념 지도와 영역별 진도, 다음에 공부하면 좋은 개념. 목표를 정하면 거기까지의 경로를 보여 줘요.
- **기록** 탭: 완료한 강의 · 개념 · 질문 · 메모.
- **설정**(오른쪽 위 톱니바퀴): 예시 코드 언어(TypeScript · JavaScript · Go · Java · Kotlin · Python), 화면 테마, Claude Code 연결 상태, 데이터 위치.

앱을 켜 두면 1시간마다 다음 추천 강의 1개를 미리 만들어 둡니다(직접 누른 강의가 항상 먼저 처리돼요). 끄려면 `PUNISHER_SCHEDULER_ENABLED=false punisher`.

## 5. 데이터는 어디에?

```
~/Library/Application Support/Punisher/punisher.db
```

강의 · 진도 · 질문 · 메모가 이 파일 하나(SQLite)에 들어 있습니다.

- 백업: 앱을 끄고 `Punisher` 폴더를 통째로 복사
- 처음부터 다시: 앱을 끄고 `punisher.db*` 파일들을 휴지통으로

## 6. 업데이트

새 `punisher` 파일을 받아 2번 설치를 다시 하면 됩니다(`sh install.sh`). 데이터는 그대로 유지되고, 필요한 DB 변경은 실행할 때 자동으로 적용돼요.

## 7. 문제 해결

| 증상 | 해결 |
| --- | --- |
| 화면 위에 "Claude Code(claude 명령)를 찾을 수 없습니다" | Claude Code 설치(1번) 후 화면의 `다시 확인`. 특이한 위치에 설치했다면 `PUNISHER_CLAUDE_BIN=/경로/claude punisher` |
| "Claude Code에 로그인되어 있지 않습니다" | 터미널에서 `claude` 실행 → 로그인 → 화면의 `다시 확인` |
| "Claude 사용 한도에 도달했습니다" | 구독 사용량 한도입니다. 몇 시간 뒤 다시 시도. 이미 만든 강의는 계속 읽을 수 있어요 |
| `punisher: command not found` | 새 터미널 창을 열거나 `~/bin/punisher` 로 실행 |
| "확인되지 않은 개발자" 경고로 안 열림 | `xattr -d com.apple.quarantine ~/bin/punisher` 또는 시스템 설정 → 개인정보 보호 및 보안 → "그래도 열기" |
| 강의 생성이 오래 걸림 | 강의 하나에 보통 2~5분 걸립니다. 화면을 닫아도 생성은 계속되고, 다시 열면 이어서 보여요 |
| 도식이 원문(코드)으로 보임 | AI가 그린 도식 문법이 틀린 경우입니다. 내용은 그대로 읽을 수 있어요 |

## 개발자용

```sh
make build     # web 빌드(→ internal/webui/dist) + macOS 유니버설 바이너리 dist/punisher
make test      # go vet + go test + 프론트 typecheck/lint
cd web && npm run dev   # 프론트 개발 서버(:5173), /v1 은 punisher --no-browser(:4710) 로 프록시
```

- Go 1.24+, Node 20+ 필요. SQLite 는 순수 Go 드라이버(modernc.org/sqlite)라 CGO 없이 교차 컴파일됩니다.
- 구조: `cmd/punisher`(진입점) · `internal/store`(SQLite) · `internal/claudecli`(로컬 claude 실행기) · `internal/services`(강의 생성·질문) · `web/`(Vite + React + React Router).
- 원본 출처는 [NOTICE](NOTICE) 를 보세요.
