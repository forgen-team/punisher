import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "@forgen-team/ui";
import { selectedProviderStatus, useAppStatus } from "@/src/shared/api/status";
import { isWitems } from "@/src/shared/config/runtime";
import { useT } from "@/src/shared/i18n";
import { isRateLimitedCode, messages, statusText, witemsStatusText } from "./i18n";

/** 고른 AI 를 지금 쓸 수 없는가(배너가 보이는가) — 틀(Page banner)이 빈 자리를 만들지 않게 먼저 본다. */
export function useClaudeStatusIssue(): boolean {
  const data = useAppStatus().data;
  return Boolean(data && selectedProviderStatus(data)?.code);
}

/**
 * 고른 AI(설정 › AI 연결: claude·codex·agy 로컬 CLI)를 쓸 수 없을 때만 보이는 안내 배너.
 * 앱 틀이 생성을 쓰는 화면에만, 머리 바로 아래 한 자리(Page banner)에 둔다(policy/20 §6).
 * 미설치 · 미로그인 · 사용 한도 각각 고치는 방법을 화면 언어로 보여 준다.
 * 이미 만든 강의·지도·기록은 AI 없이도 계속 볼 수 있다.
 * Witems 앱 모드는 셸의 AI 연결(claude-gateway)이라 로컬 CLI 단계 없이 한 줄로 알린다.
 */
export function ClaudeStatusBanner() {
  const t = useT(messages);
  const status = useAppStatus();
  const data = status.data;
  const code = data ? selectedProviderStatus(data)?.code : undefined;
  if (!data || !code) { return null; }
  const text = isWitems() ? witemsStatusText(t, code) : statusText(t, code, data.message, data.steps);
  const blocking = !isRateLimitedCode(code);

  return (
    <section
      role="status"
      className={
        blocking
          ? "border border-line bg-warning-subtle px-4 py-3 text-sm text-fg"
          : "border border-line bg-canvas-subtle px-4 py-3 text-sm text-fg"
      }
    >
      <div className="flex items-start gap-2">
        <AlertTriangle className="mt-1 h-4 w-4 shrink-0 text-warning" aria-hidden="true" />
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <p className="font-medium">{text.message}</p>
          {text.steps.length > 0 && (
            <ol className="list-decimal space-y-1 pl-6 text-fg-secondary">
              {text.steps.map((step) => (
                <li key={step}>
                  <StepText text={step} />
                </li>
              ))}
            </ol>
          )}
          {blocking && (
            <p className="text-xs text-fg-muted">{t.readOnlyNote}</p>
          )}
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => void status.refetch()}
          disabled={status.isFetching}
          aria-label={t.recheck}
        >
          <RefreshCw className={status.isFetching ? "h-4 w-4 animate-spin" : "h-4 w-4"} aria-hidden="true" />
          <span className="hidden sm:inline">{t.recheck}</span>
        </Button>
      </div>
    </section>
  );
}

/** 명령(claude·codex login·agy·설치 명령)·/login 부분을 코드체로. */
const COMMAND = /(curl -fsSL \S+ \| bash|npm install -g @openai\/codex|brew install codex|\bcodex login(?: status)?|\bclaude\b(?! Code)|\bagy\b|\/login)/g;

function StepText({ text }: { text: string }) {
  const parts = text.split(COMMAND);
  return (
    <>
      {parts.map((part, i) =>
        i % 2 === 1 ? (
          <code key={i} className="bg-canvas-elevated px-1 text-xs text-fg">
            {part}
          </code>
        ) : (
          <span key={i}>{part}</span>
        ),
      )}
    </>
  );
}
