import { Link } from "@/src/shared/lib/router";
import { AlertTriangle } from "lucide-react";
import { Button, buttonVariants } from "@/src/shared/ui/kit";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useStartRegenerate } from "./RegenerateSheet";

/**
 * 검수에서 문제가 발견된 강의(reviewState=flagged) — 본문 위 배너: 지적 요약 + '다시 만들기'.
 * 이미 다시 만든 버전(newerId)이 있으면 또 만들지 않고 그 강의로 보낸다.
 */
export function FlaggedBanner({ lectureId, reason, newerId }: { lectureId: string; reason?: string; newerId?: string }) {
  const regenerate = useStartRegenerate(lectureId);
  return (
    <div role="status" className="mt-4 flex flex-col gap-2 rounded-md bg-warning-subtle px-3 py-2">
      <p className="flex items-center gap-1.5 text-sm font-semibold text-warning">
        <AlertTriangle className="size-4 shrink-0" aria-hidden="true" />
        검수에서 문제가 발견된 강의
      </p>
      {reason && <p className="line-clamp-3 break-words text-sm text-fg">{reason}</p>}
      <div className="flex flex-wrap items-center gap-2">
        {newerId ? (
          <Link href={lectureHref(newerId)} className={buttonVariants({ variant: "secondary", size: "sm" })}>
            다시 만든 강의 보기
          </Link>
        ) : (
          <Button size="sm" variant="secondary" loading={regenerate.isPending} onClick={regenerate.start}>
            다시 만들기
          </Button>
        )}
        {regenerate.error && <p role="alert" className="text-sm text-danger">{describePunisherError(regenerate.error)}</p>}
      </div>
    </div>
  );
}
