import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "@/src/shared/ui/kit";
import { useAppStatus } from "@/src/shared/api/status";

/**
 * Claude Code(로컬 claude CLI)를 쓸 수 없을 때만 보이는 안내 배너.
 * 미설치 · 미로그인 · 사용 한도 각각 고치는 방법을 한국어로 보여 준다.
 * 이미 만든 강의·지도·기록은 Claude 없이도 계속 볼 수 있다.
 */
export function ClaudeStatusBanner() {
  const status = useAppStatus();
  const data = status.data;
  if (!data || !data.claude.code) { return null; }
  const blocking = data.claude.code !== "claude_rate_limited";

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
          <p className="font-medium">{data.message}</p>
          {data.steps && data.steps.length > 0 && (
            <ol className="list-decimal space-y-1 pl-5 text-fg-secondary">
              {data.steps.map((step) => (
                <li key={step}>
                  <StepText text={step} />
                </li>
              ))}
            </ol>
          )}
          {blocking && (
            <p className="text-xs text-fg-muted">이미 만든 강의 · 지도 · 기록은 그대로 볼 수 있어요. 새 강의 만들기와 질문만 잠시 안 돼요.</p>
          )}
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => void status.refetch()}
          disabled={status.isFetching}
          aria-label="다시 확인"
        >
          <RefreshCw className={status.isFetching ? "h-4 w-4 animate-spin" : "h-4 w-4"} aria-hidden="true" />
          <span className="hidden sm:inline">다시 확인</span>
        </Button>
      </div>
    </section>
  );
}

/** Renders `claude`·URL·명령 부분을 코드체로. */
function StepText({ text }: { text: string }) {
  const parts = text.split(/(curl -fsSL \S+ \| bash|\bclaude\b(?! Code)|\/login)/g);
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
