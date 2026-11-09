
import { useEffect } from "react";
import { Button, ErrorState, Skeleton } from "@/src/shared/ui/kit";
import { usePromptPreview } from "@/src/entities/punisher";
import { describePunisherError } from "@/src/shared/api/punisher";

function PromptText({ label, text }: { label: string; text: string }) {
  return (
    <div className="flex flex-col gap-1">
      <p className="text-xs font-semibold text-fg-secondary">{label}</p>
      <pre className="max-h-[50vh] overflow-auto whitespace-pre-wrap break-words rounded-md border border-line bg-canvas-subtle p-3 text-xs leading-5 text-fg">
        {text}
      </pre>
    </div>
  );
}

/** 예시 개념으로 실제 조립된 최종 프롬프트(읽기 전용). 생성(LLM 호출)은 하지 않는다. */
export function PreviewView({ blockId, draft }: { blockId: string; draft: string }) {
  const preview = usePromptPreview();
  const { mutate } = preview;

  useEffect(() => {
    mutate({ id: blockId, body: draft });
  }, [blockId, draft, mutate]);

  if (preview.isPending || (!preview.data && !preview.error)) { return <Skeleton lines={8} />; }
  if (preview.error || !preview.data) {
    return <ErrorState compact message={describePunisherError(preview.error)} onRetry={() => mutate({ id: blockId, body: draft })} />;
  }
  const p = preview.data;
  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-fg-muted">
        예시 개념 &lsquo;{p.conceptName}&rsquo;로 조립한 그대로예요(생성은 하지 않아요). 기록될 버전: <code>{p.version}</code>
      </p>
      {p.unknownVars.length > 0 && (
        <p role="alert" className="rounded-md bg-warning-subtle px-3 py-2 text-xs text-warning">
          이 블록이 모르는 변수는 그대로 남아요: {p.unknownVars.map((v) => `{{${v}}}`).join(", ")}
        </p>
      )}
      <PromptText label="시스템 프롬프트" text={p.system} />
      <PromptText label="사용자 프롬프트(개념·목표 등은 코드가 채움)" text={p.user} />
      <Button size="sm" variant="ghost" className="self-start" onClick={() => mutate({ id: blockId, body: draft })}>
        다시 조립
      </Button>
    </div>
  );
}
