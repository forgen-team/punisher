import { defineMessages } from "@/src/shared/i18n";

/**
 * 단독판 Claude Code(로컬 claude CLI) 상태 안내. 서버 /v1/status 의 message·steps 는 한국어라
 * 화면은 code 로 이 카탈로그를 쓴다(모르는 code 면 서버 문구). 단계는 줄바꿈으로 나눈다.
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
    readOnlyNote: "이미 만든 강의 · 지도 · 기록은 그대로 볼 수 있어요. 새 강의 만들기와 질문만 잠시 안 돼요.",
    recheck: "다시 확인",
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
    readOnlyNote: "Lectures, the map and history stay available. Only new lectures and questions are paused.",
    recheck: "Check again",
  },
});

type StatusCode = "claude_unavailable" | "claude_not_logged_in" | "claude_rate_limited";

function isStatusCode(code: string | undefined): code is StatusCode {
  return code === "claude_unavailable" || code === "claude_not_logged_in" || code === "claude_rate_limited";
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
