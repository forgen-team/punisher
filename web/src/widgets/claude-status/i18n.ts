import { defineMessages } from "@/src/shared/i18n";

/**
 * 단독판 로컬 AI CLI(claude·codex·agy) 상태 안내. 서버 /v1/status 의 message·steps 는 한국어라
 * 화면은 고른 AI 의 code 로 이 카탈로그를 쓴다(모르는 code 면 서버 문구). 단계는 줄바꿈으로 나눈다.
 */
export const messages = defineMessages({
  ko: {
    "claude_unavailable": "Claude Code(claude 명령)를 찾을 수 없어요. Claude Code를 설치한 뒤 터미널에서 claude 를 한 번 실행해 로그인하고 다시 시도해 주세요.",
    "claude_not_logged_in": "Claude Code에 로그인되어 있지 않아요. 터미널에서 claude 를 실행해 로그인(/login)한 뒤 다시 시도해 주세요.",
    "claude_rate_limited": "Claude 사용 한도에 도달했어요. 한도가 풀린 뒤 다시 시도해 주세요. 이미 만든 강의는 계속 읽을 수 있어요.",
    "steps_claude_unavailable": [
      "Claude Code를 설치합니다: 터미널에서 curl -fsSL https://claude.ai/install.sh | bash",
      "터미널에서 claude 를 실행하고 안내에 따라 로그인합니다.",
      "이 화면을 새로고침합니다(Punisher를 다시 켤 필요는 없습니다).",
    ].join("\n"),
    "steps_claude_not_logged_in": ["터미널에서 claude 를 실행합니다.", "안내에 따라 로그인(/login)을 마칩니다.", "이 화면을 새로고침합니다."].join("\n"),
    "steps_claude_rate_limited": "",
    "codex_unavailable": "Codex(codex 명령)를 찾을 수 없어요. Codex CLI를 설치한 뒤 터미널에서 codex login 으로 로그인하고 다시 시도해 주세요.",
    "codex_not_logged_in": "Codex에 로그인되어 있지 않아요. 터미널에서 codex login 을 실행해 로그인한 뒤 다시 시도해 주세요.",
    "codex_rate_limited": "Codex 사용 한도에 도달했어요. 한도가 풀린 뒤 다시 시도해 주세요. 이미 만든 강의는 계속 읽을 수 있어요.",
    "steps_codex_unavailable": [
      "Codex CLI를 설치합니다: 터미널에서 npm install -g @openai/codex (또는 brew install codex)",
      "터미널에서 codex login 을 실행해 로그인합니다.",
      "이 화면을 새로고침합니다(Punisher를 다시 켤 필요는 없습니다).",
    ].join("\n"),
    "steps_codex_not_logged_in": [
      "터미널에서 codex login 을 실행합니다.",
      "안내에 따라 로그인을 마칩니다(codex login status 로 확인할 수 있습니다).",
      "이 화면을 새로고침합니다.",
    ].join("\n"),
    "steps_codex_rate_limited": "",
    "agy_unavailable": "Antigravity CLI(agy 명령)를 찾을 수 없어요. Antigravity CLI를 설치한 뒤 터미널에서 agy 를 한 번 실행해 로그인하고 다시 시도해 주세요.",
    "agy_not_logged_in": "Antigravity에 로그인되어 있지 않아요. 터미널에서 agy 를 실행해 로그인한 뒤 다시 시도해 주세요.",
    "agy_rate_limited": "Antigravity(Gemini) 사용 한도에 도달했어요. 한도가 풀린 뒤 다시 시도해 주세요. 이미 만든 강의는 계속 읽을 수 있어요.",
    "steps_agy_unavailable": [
      "Antigravity CLI(agy)를 설치합니다: antigravity.google 의 안내를 따릅니다.",
      "터미널에서 agy 를 한 번 실행해 로그인합니다.",
      "이 화면을 새로고침합니다(Punisher를 다시 켤 필요는 없습니다).",
    ].join("\n"),
    "steps_agy_not_logged_in": ["터미널에서 agy 를 실행합니다.", "안내에 따라 로그인을 마칩니다.", "이 화면을 새로고침합니다."].join("\n"),
    "steps_agy_rate_limited": "",
    readOnlyNote: "이미 만든 강의 · 지도 · 기록은 그대로 볼 수 있어요. 새 강의 만들기와 질문만 잠시 안 돼요.",
    recheck: "다시 확인",
    witemsUnavailable: "Witems AI 연결에 닿지 못했어요. 잠시 뒤 다시 확인해 주세요.",
    witemsNotLoggedIn: "Witems 서버의 Claude 로그인이 풀렸어요. 서버에서 다시 로그인해야 해요.",
  },
  en: {
    "claude_unavailable": "Can't find Claude Code (the claude command). Install Claude Code, run claude once in a terminal to sign in, then try again.",
    "claude_not_logged_in": "You're not signed in to Claude Code. Run claude in a terminal, sign in (/login), then try again.",
    "claude_rate_limited": "You've hit the Claude usage limit. Try again after it resets. Lectures you already have stay readable.",
    "steps_claude_unavailable": [
      "Install Claude Code: in a terminal, run curl -fsSL https://claude.ai/install.sh | bash",
      "Run claude in a terminal and follow the prompts to sign in.",
      "Refresh this page (no need to restart Punisher).",
    ].join("\n"),
    "steps_claude_not_logged_in": ["Run claude in a terminal.", "Finish signing in (/login) as prompted.", "Refresh this page."].join("\n"),
    "steps_claude_rate_limited": "",
    "codex_unavailable": "Can't find Codex (the codex command). Install the Codex CLI, sign in with codex login in a terminal, then try again.",
    "codex_not_logged_in": "You're not signed in to Codex. Run codex login in a terminal, sign in, then try again.",
    "codex_rate_limited": "You've hit the Codex usage limit. Try again after it resets. Lectures you already have stay readable.",
    "steps_codex_unavailable": [
      "Install the Codex CLI: in a terminal, run npm install -g @openai/codex (or brew install codex)",
      "Run codex login in a terminal to sign in.",
      "Refresh this page (no need to restart Punisher).",
    ].join("\n"),
    "steps_codex_not_logged_in": [
      "Run codex login in a terminal.",
      "Finish signing in as prompted (codex login status shows whether you're signed in).",
      "Refresh this page.",
    ].join("\n"),
    "steps_codex_rate_limited": "",
    "agy_unavailable": "Can't find the Antigravity CLI (the agy command). Install the Antigravity CLI, run agy once in a terminal and sign in, then try again.",
    "agy_not_logged_in": "You're not signed in to Antigravity. Run agy once in a terminal and sign in, then try again.",
    "agy_rate_limited": "You've hit the Antigravity (Gemini) usage limit. Try again after it resets. Lectures you already have stay readable.",
    "steps_agy_unavailable": [
      "Install the Antigravity CLI (agy): follow the instructions at antigravity.google.",
      "Run agy once in a terminal and sign in.",
      "Refresh this page (no need to restart Punisher).",
    ].join("\n"),
    "steps_agy_not_logged_in": ["Run agy in a terminal.", "Finish signing in as prompted.", "Refresh this page."].join("\n"),
    "steps_agy_rate_limited": "",
    readOnlyNote: "Lectures, the map and history stay available. Only new lectures and questions are paused.",
    recheck: "Check again",
    witemsUnavailable: "Couldn't reach the Witems AI connection. Check again in a moment.",
    witemsNotLoggedIn: "Claude on the Witems server is signed out. It needs to be signed in again on the server.",
  },
});

/** 설정 › AI 연결 화면 문구(제공자 이름은 shared/api/status AI_PROVIDER_NAMES). */
export const panelMessages = defineMessages({
  ko: {
    legend: "강의 생성·검수·질문 답·채점에 쓸 AI",
    connected: (name: string) => `연결됨 — 강의 생성·질문에 내 ${name} 계정을 사용해요.`,
    selected: "사용 중",
    installed: "설치됨",
    notInstalled: "설치 안 됨",
    loggedIn: "로그인됨",
    loggedOut: "로그인 필요",
    loginUnknown: "로그인 확인 전",
    statusLoadFailed: "상태를 불러오지 못했어요.",
    activeJobs: "진행 중 생성",
    activeCount: (active: number, queued: number) => `${active}개${queued > 0 ? ` · 대기 ${queued}개` : ""}`,
    dataDir: "데이터 위치",
    version: "Punisher 버전",
    witemsConnected: "Witems AI 연결을 써요(셸을 따름).",
  },
  en: {
    legend: "AI used to write and review lectures, answer questions and grade checks",
    connected: (name: string) => `Connected — lectures and questions use your ${name} account.`,
    selected: "In use",
    installed: "Installed",
    notInstalled: "Not installed",
    loggedIn: "Signed in",
    loggedOut: "Sign-in needed",
    loginUnknown: "Sign-in not checked yet",
    statusLoadFailed: "Couldn't load the status.",
    activeJobs: "Generating now",
    activeCount: (active: number, queued: number) => `${active}${queued > 0 ? ` · ${queued} queued` : ""}`,
    dataDir: "Data folder",
    version: "Punisher version",
    witemsConnected: "Uses the Witems AI connection (follows the shell).",
  },
});

type Provider = "claude" | "codex" | "agy";
type StatusCode = `${Provider}_${"unavailable" | "not_logged_in" | "rate_limited"}`;

const STATUS_CODE = /^(claude|codex|agy)_(unavailable|not_logged_in|rate_limited)$/;

function isStatusCode(code: string | undefined): code is StatusCode {
  return code !== undefined && STATUS_CODE.test(code);
}

/** 사용 한도 code(생성이 잠시 멈출 뿐 막히지는 않는다). */
export function isRateLimitedCode(code: string | undefined): boolean {
  return code?.endsWith("_rate_limited") ?? false;
}

/** 상태 code 의 안내 문구와 고치는 단계 — 모르는 code 면 서버가 보낸 문구 그대로. */
export function statusText(
  t: (typeof messages)["ko"],
  code: string | undefined,
  serverMessage?: string,
  serverSteps?: string[],
): { message: string; steps: string[] } {
  if (!isStatusCode(code)) { return { message: serverMessage ?? "", steps: serverSteps ?? [] }; }
  const steps = t[`steps_${code}`];
  return { message: t[code], steps: steps ? steps.split("\n") : [] };
}

/**
 * Witems 앱 모드: AI 는 호스트 claude-gateway 하나라 로컬 CLI 설치·로그인 단계는 해당이 없다.
 * 사용 한도는 단독판과 같은 문구, 그 밖은 연결/로그인 두 가지로 줄인다.
 */
export function witemsStatusText(t: (typeof messages)["ko"], code: string | undefined): { message: string; steps: string[] } {
  if (!code) { return { message: "", steps: [] }; }
  if (isRateLimitedCode(code)) { return { message: t.claude_rate_limited, steps: [] }; }
  if (code.endsWith("_not_logged_in")) { return { message: t.witemsNotLoggedIn, steps: [] }; }
  return { message: t.witemsUnavailable, steps: [] };
}
