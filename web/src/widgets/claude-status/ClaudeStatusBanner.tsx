import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "@/src/shared/ui/kit";
import { selectedProviderStatus, useAppStatus } from "@/src/shared/api/status";
import { useT } from "@/src/shared/i18n";
import { isRateLimitedCode, messages, statusText } from "./i18n";

/**
 * 고른 AI(설정 › AI 연결: claude·codex·agy 로컬 CLI)를 쓸 수 없을 때만 보이는 안내 배너.
 * 미설치 · 미로그인 · 사용 한도 각각 고치는 방법을 화면 언어로 보여 준다.
 * 이미 만든 강의·지도·기록은 AI 없이도 계속 볼 수 있다.
 */
export function ClaudeStatusBanner() {
  const t = useT(messages);
  const status = useAppStatus();
  const data = status.data;
  const code = data ? selectedProviderStatus(data)?.code : undefined;
  if (!data || !code) { return null; }
  const text = statusText(t, code, data.message, data.steps);
  const blocking = !isRateLimitedCode(code);

  return (
    <section
      role="status"
      className={
        blocking
          ? "rounded-lg border border-line bg-warning-subtle px-4 py-3 text-sm text-fg"
          : "rounded-lg border border-line bg-canvas-subtle px-4 py-3 text-sm text-fg"
      }
    >
      <div className="flex items-start gap-2">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-warning" aria-hidden="true" />
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <p className="font-medium">{text.message}</p>
          {text.steps.length > 0 && (
            <ol className="list-decimal space-y-1 pl-5 text-fg-secondary">
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
          <code key={i} className="rounded-sm bg-canvas-elevated px-1 py-0.5 text-xs text-fg">
            {part}
          </code>
        ) : (
          <span key={i}>{part}</span>
        ),
      )}
    </>
  );
}
