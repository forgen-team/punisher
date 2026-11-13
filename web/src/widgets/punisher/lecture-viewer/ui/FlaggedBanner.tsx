
import { AlertTriangle } from "lucide-react";
import { Button } from "@/src/shared/ui/kit";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useStartRegenerate } from "./RegenerateSheet";

/** 검수에서 문제가 발견된 강의(reviewState=flagged) — 본문 위 배너: 지적 요약 + '다시 만들기'. */
export function FlaggedBanner({ lectureId, reason }: { lectureId: string; reason?: string }) {
  const regenerate = useStartRegenerate(lectureId);
  return (
    <div role="status" className="mt-4 flex flex-col gap-2 rounded-md bg-warning-subtle px-3 py-2">
      <p className="flex items-center gap-1.5 text-sm font-semibold text-warning">
        <AlertTriangle className="size-4 shrink-0" aria-hidden="true" />
        검수에서 문제가 발견된 강의
      </p>
      {reason && <p className="line-clamp-3 break-words text-sm text-fg">{reason}</p>}
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="secondary" loading={regenerate.isPending} onClick={regenerate.start}>
          다시 만들기
        </Button>
        {regenerate.error && <p role="alert" className="text-sm text-danger">{describePunisherError(regenerate.error)}</p>}
      </div>
    </div>
  );
}
