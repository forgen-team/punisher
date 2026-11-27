<p align="center">
  <img src="assets/banner.png" alt="punisher" width="100%"/>
</p>

<p align="center">
  <strong>매일 30분, 내 AI 코딩 도구로 만드는 나만의 CS 기본기 강의.</strong><br/>
  내 컴퓨터(macOS·Linux)에서만 돌아가고, 내 Claude Code·Codex·Antigravity 계정으로 강의를 만듭니다.
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"/></a>
  <img src="https://img.shields.io/badge/macOS-universal-lightgrey.svg" alt="macOS universal"/>
  <img src="https://img.shields.io/badge/Linux-amd64%20%7C%20arm64-lightgrey.svg" alt="Linux amd64 | arm64"/>
  <a href="https://github.com/forgen-team/punisher/actions/workflows/ci.yml"><img src="https://github.com/forgen-team/punisher/actions/workflows/ci.yml/badge.svg" alt="CI"/></a>
  <a href="https://github.com/forgen-team/forgen"><img src="https://img.shields.io/badge/forgen-family-f97316.svg" alt="Part of the forgen family"/></a>
</p>

<p align="center">
  <a href="README.md">English</a> &middot; 한국어
</p>

---

Claude Code·Codex·Antigravity(Gemini) 중 내가 쓰는 AI 코딩 도구로 나만의 CS 기본기 강의를 만들어 읽는 학습 앱입니다.
오늘 공부할 개념을 추천받고, 강의를 실시간으로 생성해 읽고, 모르는 용어나 문장을 바로 물어볼 수 있어요.
지식 지도에서 8개 영역(자료구조 · 컴퓨터구조 · OS · 네트워크/웹 · DB · 분산 · 인프라 · 보안/설계), 168개 개념의 진도를 한눈에 봅니다.

- 내 컴퓨터에서만 돌아갑니다. 서버·Docker·DB 설치나 API 키, 회원가입은 필요 없어요.
- 강의 생성과 질문 답변에는 **내 컴퓨터에 로그인된 AI CLI** 하나를 씁니다: Claude Code(`claude`), Codex(`codex`), Antigravity(`agy`, Gemini). 사용량은 그 도구의 내 구독에서 차감돼요. 처음에는 설치된 것을 자동으로 고르고(Claude Code → Codex → Antigravity 순), 설정 › AI 연결에서 바꿀 수 있어요.
- 처음 열면 개발 경험(처음 · 코드 조금 · 웹 실무 · CS 탄탄)과 영역별로 아는 정도를 물어요. 답에 맞춰 설명을 얼마나 풀어 쓸지와 어디서부터 시작할지가 정해져요(개발이 처음이면 0단계 기초 트랙부터).
- 처음 설치하면 맛보기 강의 몇 편이 들어 있어서, 구독을 쓰지 않고 먼저 읽어 볼 수 있어요(확인하기 문항을 새로 만들 때는 구독을 써요).

> **AI가 만든 강의라 틀릴 수 있어요.** 모든 강의는 사실 오류 검수를 통과해야 발행되지만, 출처와 검수 표시를 확인하고 수치·버전에 따라 다른 내용은 한 번 더 확인하세요.

<p align="center">
  <img src="assets/screenshots/today.png" alt="오늘" width="49%"/>
  <img src="assets/screenshots/lecture.png" alt="강의 속 직접 해 보기" width="49%"/>
  <img src="assets/screenshots/check.png" alt="확인하기" width="49%"/>
  <img src="assets/screenshots/map.png" alt="지식 지도" width="49%"/>
</p>

## 1. 준비물

1. macOS (Apple Silicon / Intel) 또는 Linux (amd64 / arm64)
2. AI 코딩 CLI **하나**를 설치하고 로그인합니다. 셋 중 무엇이든 돼요.

   **Claude Code** ([공식 안내](https://docs.claude.com/en/docs/claude-code)) — 설치 스크립트는 받아서 내용을 확인한 뒤 실행하세요.
   ```sh
   curl -fsSL https://claude.ai/install.sh -o claude-install.sh
   less claude-install.sh      # 무엇을 하는지 확인
   bash claude-install.sh
   claude                      # 처음 한 번 실행해서 안내대로 로그인한 뒤 /exit 로 나오기
   ```
   `claude auth status` 를 실행했을 때 `"loggedIn": true` 가 나오면 준비 끝입니다.

   **Codex** ([공식 안내](https://developers.openai.com/codex)) — `npm install -g @openai/codex` 후 `codex login`. `codex login status` 가 로그인됨을 보여 주면 준비 끝입니다. Codex 는 호출마다 자체 지시문이 길어 같은 작업에도 사용량이 더 들어요.

   **Antigravity CLI** (`agy`, Gemini — [공식 안내](https://antigravity.google/docs/cli/headless)) — 설치 후 터미널에서 `agy` 를 한 번 실행해 로그인하세요. (2026-06 부터 Gemini 구독은 Gemini CLI 대신 Antigravity CLI 를 써요.)

## 2. 설치

1. 받은 `punisher-<버전>-macos-universal.zip` 을 더블클릭해 압축을 풉니다(안에 `punisher`, `install.sh`, 이 설명서가 있어요). Linux는 `tar xzf punisher-<버전>-linux-amd64.tar.gz`(ARM이면 `-arm64`).
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
- 기본으로는 **이 Mac에서만** 접속됩니다(127.0.0.1). 내 폰에서 보려면 [8. 폰에서 보기](#8-폰에서-보기)를 따라 하세요.

옵션:

| 옵션 | 설명 |
| --- | --- |
| `--port 4800` | 포트 지정 |
| `--no-browser` | 브라우저 자동 열기 끄기 |
| `--data-dir 경로` | 데이터 폴더 바꾸기 |
| `--version` | 버전 확인 |

## 4. 사용법

- **강의** 탭: 오늘의 추천을 누르고 `강의 시작` → 강의가 실시간으로 써지고, 다 되면 저장돼요. 위 입력창에 궁금한 것(예: "Pod가 죽었다")을 써도 됩니다.
- 강의 안에서 밑줄 친 **용어**를 누르면 짧은 설명이 나와요(`더 쉽게` · `이어서 묻기` · `나중에 읽기`). 문장을 **드래그해 선택**(폰에서는 **탭**)하면 `쉽게 · 깊게 · 질문 · 메모`가 나오고, 그 구절에 대한 대화가 이어져요. 대화에서 `이 대화로 심화 강의 만들기`도 돼요. 대화가 있는 구절에는 `대화 n` 표시가 붙어요.
- 필요 없는 개념은 지도·추천에서 `필요 없음`으로 뺄 수 있어요(되돌리기 가능).
- 다 읽으면 맨 아래 `완료` (이미 아는 내용이면 `이미 앎`). 용어 설명의 `이 개념 더 깊게`로 연 강의는 원 강의 아래에 이어져 보여요.
- **학습 요청**(강의 탭 오른쪽/아래 카드): 공부하고 싶은 주제를 적고 `추가` → 개념을 찾아 그 강의를 미리 만들어 둬요. `시작`(주제부터) 또는 `기초부터`(모르는 선행 개념부터). 주제 강의를 끝내면 `완료 N`으로 접히고, ⋯ 메뉴로 목록에서 뺄 수 있어요.
- **지도** 탭: 전체 개념 지도와 영역별 진도, 다음에 공부하면 좋은 개념. 목표를 정하면 거기까지의 경로를 보여 줘요.
- **직접 해 보기**(강의 안): 내 답을 한 줄이라도 써야 풀이가 열리고, 튜터가 짧게 평을 달아 줘요. 바쁘면 `건너뛰고 풀이 보기`.
- **내 터미널에서 해 보기**(강의 안, 있는 강의만): 읽기 전용 명령 하나(`lsof`, `dig`, `curl -I` 등)를 복사해 내 터미널에서 실행하고, 결과를 붙여 넣으면 튜터가 어디를 보면 되는지 읽어 줘요. 앱은 명령을 실행하지 않아요. 붙여 넣기 전에 비밀번호·토큰은 지우세요.
- **확인하기**(강의 끝): 짧게 답하며 이해했는지 확인해요. 튜터가 정답 대신 질문을 좁혀 가요. 답하기 전에 `확실 · 대충 · 모름`을 고르면, 확실했는데 틀린 답은 결과에 '착각'으로 따로 보여요. 학습 목표마다 숙달 레벨이 쌓여요.
- **복습**(강의 탭 맨 앞 카드): 확인한 목표를 간격을 두고 다시 물어요. 기본은 답을 쓴 뒤 모범답을 보고 직접 채점해요. 잘 기억하는 목표는 간격이 조금씩 길어지고, 자주 틀리는 목표는 짧아져요.
- **이번 주 종합 문제**(오늘의 추천 맨 앞, 주 1회): 확인한 개념 3~4개를 실무 장면 하나에 섞어서 물어요. 숙달 상자는 움직이지 않고, 통과하면 '설명 가능' 조건이 채워지고 틀리면 관련 복습이 앞당겨져요.
- **고칠 곳**(강의 메뉴): 내가 막힌 섹션(틀리거나 건너뛴 연습, 예상과 다른 실습, 질문이 몰린 곳)이 보여요. `이 내용으로 다시 만들기`를 누르면 그 내용을 반영해 새 버전을 만들어요.
- **기록** 탭: 완료한 강의 · 개념 · 질문 · 메모, 확인·복습 기록과 기억 유지 지표.
- **언어 / Language**(설정 > 학습 맨 위): 한국어·English. 화면 문구와 새로 만드는 강의·문항·튜터 답의 언어가 함께 바뀌어요(이미 만든 강의는 그 언어 그대로). 처음에는 OS 언어(`LANG`, macOS 는 시스템 언어)가 한국어면 한국어, 아니면 영어로 시작하고, `PUNISHER_LOCALE=ko punisher` 로 정할 수도 있어요.
- **설정**(오른쪽 위 톱니바퀴): 예시 코드 언어(TypeScript · JavaScript · Go · Java · Kotlin · Python), 하루 학습량, 숙달 기준·복습 간격·복습 간격 자동 조절·하루 복습 시간·복습 채점 방식, 실습 환경(macOS·Linux·쓰지 않음), 강의 전 예측 질문, 화면 테마, 설명 수준, AI 연결(Claude Code · Codex · Antigravity), 데이터 위치.
- **설정 > 고급 > 강의 프롬프트**: 강의를 만드는 지시문을 단계별로 보고 고쳐요(기본값 비교 · 미리보기 · 이력 20개). 고친 뒤 강의 메뉴의 `새 프롬프트로 다시 만들기`로 지금 강의의 새 버전을 만들 수 있어요.

강의는 사실 오류 검수를 통과해야 발행돼요. 생성 중에는 '계획 → 작성 → 검수 → 고침' 단계만 보이고, 보통 4~8분 걸려요. 검수 전에 앱이 SQL 예시를 메모리 SQLite 에서 돌려 보고, 본문의 계산을 다시 하고, 연습·예측 문제를 풀이를 가린 채 따로 풀게 하고, 공식 문서 출처 링크가 열리는지 확인해 검수자에게 넘겨요(외부 명령은 실행하지 않아요). 끄려면 `PUNISHER_LECTURE_VERIFY=false`(따로 풀기), `PUNISHER_LINK_CHECK=false`(링크 확인), `PUNISHER_INSTANT_OUTLINE=false`(즉석 강의의 짧은 계획).

강의를 읽는 중에는 섹션 끝 '방금 읽은 걸 떠올려 보기'에 답해 보고 한 번 눌러 스스로 채점해요. 바로 숙달로 치지 않고 하루 뒤 복습에 넣어요(복습이 밀려 있으면 대기열로). 예측 질문은 한 줄 예측이나 확신도를 남겨야 출력이 열리고, 한 장 요약은 핵심을 먼저 떠올려 적은 뒤 열려요(적은 글은 저장하지 않아요).

앱을 켜 두면 1시간마다 다음 추천 강의 1개를 미리 만들고, 검수 기록이 없는 기존 강의를 최대 2편씩 검수해요(직접 누른 강의가 항상 먼저 처리돼요). 모두 끄려면 `PUNISHER_SCHEDULER_ENABLED=false punisher`, 기존 강의 검수만 끄려면 `PUNISHER_BACKFILL_REVIEW_ENABLED=false punisher`.

## 5. 데이터는 어디에?

```
macOS: ~/Library/Application Support/Punisher/punisher.db
Linux: ~/.local/share/punisher/punisher.db   ($XDG_DATA_HOME 이 있으면 그 아래, 예전 ~/.punisher 가 있으면 그대로)
```

강의 · 진도 · 질문 · 메모 · 연습 답이 이 파일 하나(SQLite)에 들어 있습니다.

- 내보내기: `punisher export` (또는 설정 > 고급 > `데이터 내보내기`) → `punisher-backup-날짜.zip`. 앱이 켜져 있어도 돼요.
- 가져오기: 앱을 끄고(폰 연결을 켰다면 `punisher phone off`) `punisher import 파일.zip`. 지금 데이터는 옆 폴더(`…bak-날짜`)에 남겨 둬요. 더 새 버전에서 내보낸 파일은 Punisher를 먼저 업데이트해야 가져올 수 있어요.
- 내보낸 파일에는 내가 쓴 답과 붙여 넣은 실습 출력도 들어 있어요. 남에게 줄 때는 확인하세요.
- 처음부터 다시: 앱을 끄고 `punisher.db*` 파일들을 휴지통으로

## 6. 업데이트

버전별로 바뀐 점은 [CHANGELOG.md](CHANGELOG.md)에 있어요.

새 `punisher` 파일을 받아 2번 설치를 다시 하면 됩니다(`sh install.sh`). 데이터는 그대로 유지되고, 필요한 DB 변경은 실행할 때 자동으로 적용돼요.

## 7. 문제 해결

| 증상 | 해결 |
| --- | --- |
| 화면 위에 "Claude Code(claude 명령)를 찾을 수 없습니다" | Claude Code 설치(1번) 후 화면의 `다시 확인`. 특이한 위치에 설치했다면 `PUNISHER_CLAUDE_BIN=/경로/claude punisher` |
| "Claude Code에 로그인되어 있지 않습니다" | 터미널에서 `claude` 실행 → 로그인 → 화면의 `다시 확인` |
| "Claude 사용 한도에 도달했습니다" | 구독 사용량 한도입니다. 몇 시간 뒤 다시 시도. 이미 만든 강의는 계속 읽을 수 있어요 |
| "Codex(codex 명령)를 찾을 수 없습니다" / 로그인 안내 | Codex 설치 후 `codex login`, 화면의 `다시 확인`. 특이한 위치면 `PUNISHER_CODEX_BIN=/경로/codex punisher` |
| "Antigravity CLI(agy 명령)를 찾을 수 없습니다" | Antigravity CLI 설치 후 `agy` 로 로그인, `다시 확인`. 특이한 위치면 `PUNISHER_AGY_BIN=/경로/agy punisher` |
| 다른 AI 로 쓰고 싶음 | 설정 › AI 연결에서 바꾸거나 처음 켤 때 `PUNISHER_AI_PROVIDER=codex punisher` |
| `punisher: command not found` | 새 터미널 창을 열거나 `~/bin/punisher` 로 실행 |
| "확인되지 않은 개발자" 경고로 안 열림 | `xattr -d com.apple.quarantine ~/bin/punisher` 또는 시스템 설정 → 개인정보 보호 및 보안 → "그래도 열기" |
| 강의 생성이 오래 걸림 | 작성과 검수까지 보통 4~7분, 검수에서 고칠 곳이 나오면 더 걸려요. 화면을 닫아도 생성은 계속되고, 다시 열면 이어서 보여요 |
| 강의가 발행되지 않음 | 검수에서 사실 오류가 고쳐지지 않으면 발행하지 않아요. 화면의 이유를 보고 `다시 만들기`를 눌러 주세요 |
| 도식이 원문(코드)으로 보임 | AI가 그린 도식 문법이 틀린 경우입니다. 내용은 그대로 읽을 수 있어요 |

## 8. 폰에서 보기

Mac을 켜 둔 채 밖에서 폰으로 강의를 읽을 수 있어요.

> Linux에서는 `punisher phone on`이 `systemd --user` 서비스로 켜져요. 로그아웃해도 켜 두려면 한 번만 `loginctl enable-linger $USER`, Tailscale이 권한을 거부하면 한 번만 `sudo tailscale set --operator=$USER`를 실행하세요. 아래 설명은 Mac 기준이에요. 무료 앱 **Tailscale**이 내 계정의 기기끼리만 암호화해서 이어 줍니다.
기록은 그대로 Mac에만 있고, 폰은 Mac 화면을 열어 보는 방식이에요.

> **맥북이 잠들면 폰에서 새로 열 수 없어요.** 뚜껑을 닫으면 Mac이 잠듭니다. 폰에서 **한 번 열어 본 강의·질문 해설과 오늘 카드**는 폰에 사본이 남아 읽을 수 있어요(오프라인 읽기). 채점·저장·새 강의는 Mac이 깨어 있어야 해요.
>
> **나가기 전에 매번: 전원을 꽂고, 뚜껑을 열어 둔 채로 두세요.** 화면은 꺼져도 괜찮아요. (아래 4번의 배터리 설정을 켜 두어야 화면이 꺼져도 잠들지 않아요.)
>
> 집에 두고 나왔는데 뚜껑이 닫혔거나 전원이 빠졌다면, 돌아가서 깨우기 전에는 열 방법이 없어요.
>
> Mac이 깨어나면 저절로 다시 연결돼요. 잠시 기다렸다가 폰에서 다시 열어 보세요. 만들던 강의가 실패했으면 `다시 시도`를 누르세요.

### 처음 한 번만 하면 돼요

**1. Tailscale 설치**: Mac은 <https://tailscale.com/download> 에서 받습니다. 폰은 앱 스토어(안드로이드는 Play 스토어)에서 "Tailscale"을 받으세요. Mac에서 시스템 확장이나 VPN을 허용하라고 물으면 허용합니다. 시스템 설정의 개인정보 보호 및 보안에서 허용해야 할 수도 있어요.

**2. Mac과 폰에서 같은 계정으로 로그인**: 구글 계정 등 **똑같은 계정**을 쓰세요. 다른 계정으로 로그인한 기기에서는 열리지 않아요. 폰에서 VPN 구성을 허용할지 물으면 허용합니다. 개인 사용은 무료예요.

**3. Mac 이름 확인**: <https://login.tailscale.com/admin/machines> 를 같은 계정으로 열면 Mac과 폰이 함께 보여요. 여기 보이는 Mac 이름이 폰 주소에 들어가는데, **주소를 안전하게 만드는 과정에서 인터넷에 공개되는 기록에 남아요.** 이름에 실명이 있으면 먼저 `punisher-mac`처럼 바꾸세요. (줄 오른쪽 `...` 메뉴의 Edit machine name)

**4. Mac 설정 두 가지**
- 메뉴 막대의 Tailscale 아이콘 > 설정(Settings)에서 **로그인 시 실행(Run at login)** 을 켭니다. 보이지 않으면 시스템 설정 > 일반 > 로그인 항목에 Tailscale을 넣어 주세요.
- 시스템 설정 > 배터리 > 옵션에서 **전원 어댑터를 쓸 때 디스플레이가 꺼져도 자동으로 잠자지 않게** 하는 항목을 켭니다. (영어 화면: Prevent automatic sleeping on power adapter when the display is off)

**5. 켜기**: Punisher가 켜져 있으면 그 터미널 창에서 `Ctrl+C`로 끄세요. 그런 다음 터미널에 입력합니다.

```sh
punisher phone on
```

처음에는 `https://login.tailscale.com/...` 링크가 나와요. Command(⌘) 키를 누른 채 링크를 클릭하고, 열린 화면에서 켜기(Enable)를 누르세요. 승인하면 터미널이 자동으로 이어져요. 5분 안에 승인하지 못했거나 중간에 멈췄다면 같은 명령을 다시 실행하면 됩니다.

**6. 폰에서 열기**: 폰의 Tailscale 앱에서 연결을 켜고, 터미널에 나온 QR을 찍거나 `https://...ts.net` 주소를 여세요.
- 아이폰: Safari의 공유 버튼 > **홈 화면에 추가**
- 안드로이드: Chrome의 메뉴(⋮) > **홈 화면에 추가**

다 됐으면 터미널 창은 닫아도 돼요. 이제 Mac에 로그인하면 Punisher가 알아서 켜져요. Mac에서 보고 싶을 때는 터미널에 `punisher`를 입력하거나 브라우저로 <http://127.0.0.1:4710> 을 열면 됩니다.

### 처음 켠 뒤 확인해 주세요

폰에서 확인:
- [ ] 폰의 **와이파이를 끄고**(LTE/5G) 주소가 열린다
- [ ] 강의를 하나 시작하면 글이 실시간으로 써진다
- [ ] 용어를 누르면 설명이 나오고, 문장을 선택해 `이 부분 설명`을 쓸 수 있다
- [ ] 홈 화면 아이콘으로 열면 주소창 없이 앱처럼 뜬다
- [ ] 폰의 Tailscale을 끄면 주소가 **열리지 않는다**
- [ ] Tailscale이 없는 다른 사람의 폰에서는 주소가 **열리지 않는다**
- [ ] 다른 Tailscale 계정의 기기가 있다면, 그 기기에서는 `forbidden`만 나온다 (거부 응답 403)

Mac과 함께 확인:
- [ ] Mac의 브라우저로 같은 `https://...ts.net` 주소를 열어도 화면이 나온다 (정상 응답 200)
- [ ] 전원을 꽂고 뚜껑을 열어 둔 채 화면이 꺼질 때까지 기다리고, 꺼진 뒤 10분이 지나도 폰에서 열린다
- [ ] Mac을 재시동하고 로그인만 하면 폰에서 다시 열린다
- [ ] `punisher phone off` 뒤에는 폰에서 열리지 않는다 (확인했으면 `punisher phone on`으로 다시 켜세요)

안 되는 항목이 있으면 아래 '안 열릴 때'를 보세요.

### 끄기

```sh
punisher phone off
```

자동 시작과 폰 연결이 모두 꺼지고, 전처럼 터미널에서 `punisher`로 실행하는 방식으로 돌아가요. 기록은 지워지지 않습니다.

### 안 열릴 때

| 증상 | 확인할 것 |
| --- | --- |
| 폰에서 페이지가 안 열림 | 폰의 Tailscale이 켜져 있는지 → Mac이 깨어 있는지(뚜껑, 전원) → Mac의 Tailscale이 로그인 상태인지 |
| Mac을 재시동한 뒤 안 열림 | Mac에 로그인해야 Punisher가 다시 켜져요 |
| 몇 달 뒤 갑자기 안 열림 | Tailscale 로그인이 만료됐을 수 있어요. Mac의 메뉴 막대 Tailscale 아이콘에서 다시 로그인하세요 |
| `forbidden`이라는 글자만 나옴 | Mac과 폰이 같은 계정인지 확인하고, `punisher phone on`을 다시 실행해 새로 나온 주소로 여세요. Mac 이름을 바꿨거나 계정을 바꿨다면 꼭 다시 실행해야 해요 |
| 강의가 만들어지지 않음 | Mac 터미널에서 `claude`를 실행해 로그인되어 있는지 확인하세요 |
| 터미널에서는 `claude`가 되는데 폰에서 강의가 안 만들어짐 | `punisher phone on`을 다시 실행하세요. Claude Code를 나중에 설치했거나 위치를 바꿨을 때 필요해요 |
| Punisher를 업데이트한 뒤 | 새 버전으로 바꾸려면 `punisher phone on`을 다시 실행하세요 |
| 그 밖의 문제 | 터미널에 `open ~/Library/Logs/Punisher/punisher.log`를 입력하면 기록이 열려요. 그 내용을 Punisher를 준 사람에게 보여 주세요 |

폰 연결을 켜면 `4710` 포트를 항상 씁니다. 다른 프로그램이 쓰고 있으면 Mac을 재시동한 뒤 바로 다시 실행해 주세요.

### 지켜 주세요

- Punisher에는 로그인 화면이 없어요. 주소가 열리는 사람은 내 기록을 보고 **내 Claude 사용량으로 강의를 만들 수 있어요.** 그래서 내 Tailscale 계정으로 들어온 요청만 받아요.
- Mac의 Tailscale 계정을 다른 사람과 함께 쓰지 마세요. 인터넷 전체에 공개하는 기능인 Tailscale Funnel이나 공유기 설정(포트 포워딩)도 쓰지 마세요. Funnel이 켜져 있으면 `phone on`이 멈추고, 들어오는 요청도 앱이 거부해요.
- 켜 두는 동안에는 1시간마다 다음 추천 강의를 미리 만들어서 Claude 사용량이 조금 쓰여요. 원하지 않으면 `PUNISHER_SCHEDULER_ENABLED=false punisher phone on`으로 켜세요.

## 개발자용

```sh
make build     # web 빌드(→ internal/webui/dist) + macOS 유니버설 바이너리 dist/punisher
make test      # go vet + go test + 프론트 typecheck/lint
cd web && npm run dev   # 프론트 개발 서버(:5173), /v1 은 punisher --no-browser(:4710) 로 프록시
```

- Go 1.24+, Node 20+ 필요. SQLite 는 순수 Go 드라이버(modernc.org/sqlite)라 CGO 없이 교차 컴파일됩니다.
- 구조: `cmd/punisher`(진입점) · `internal/store`(SQLite) · `internal/claudecli`(로컬 claude 실행기) · `internal/services`(강의 생성·질문) · `web/`(Vite + React + React Router).
- 원본 출처는 [NOTICE](NOTICE), witems 변경을 옮겨 오는 방법은 [docs/SYNC.md](docs/SYNC.md) 를 보세요.
- 로고·배너 PNG 는 `scripts/generate-brand-assets.sh`(rsvg-convert)로 `assets/*.svg` 에서 만듭니다.

## forgen 패밀리

Punisher 는 Claude 를 내 방식에 맞추는 [forgen](https://github.com/forgen-team/forgen) 의 형제 프로젝트입니다.
같은 개발자가 만든 Handoff(내 노트북에서 일을 맡기고 휴대폰으로 확인하는 워크플로 에이전트)와 같은 불꽃 로고·오렌지 색을 씁니다.

## 라이선스

MIT © 2026 jang-ujin · [forgen](https://github.com/forgen-team/forgen) 패밀리의 형제 프로젝트입니다.
