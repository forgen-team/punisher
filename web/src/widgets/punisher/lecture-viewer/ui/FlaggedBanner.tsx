import { Link } from "@/src/shared/lib/router";
import { AlertTriangle } from "lucide-react";
import { Button } from "@forgen-team/ui";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, translateReasonCodes } from "@/src/shared/api/punisher";
import { useStartRegenerate } from "./RegenerateSheet";
import { useLocale, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/**
 * 검수에서 문제가 발견된 강의(reviewState=flagged) — 본문 위 배너: 지적 요약 + '다시 만들기'.
 * 이미 다시 만든 버전(newerId)이 있으면 또 만들지 않고 그 강의로 보낸다.
 */
export function FlaggedBanner({ lectureId, reason, reasonCodes, newerId }: {
  lectureId: string;
  reason?: string;
  /** reason 의 사유 code — 모두 아는 code 면 화면 언어로 옮겨 보인다 */
  reasonCodes?: string[];
  newerId?: string;
}) {
  const t = useT(messages);
  const translated = translateReasonCodes(reasonCodes, useLocale());
  const shown = translated.length > 0 ? translated.join(" · ") : reason;
  const regenerate = useStartRegenerate(lectureId);
  return (
    <div role="status" className="mt-4 flex flex-col gap-2 rounded-md bg-warning-subtle px-3 py-2">
      <p className="flex items-center gap-1.5 text-sm font-semibold text-warning">
        <AlertTriangle className="size-4 shrink-0" aria-hidden="true" />
        {t.flaggedTitle}
      </p>
      {shown && <p className="line-clamp-3 break-words text-sm text-fg">{shown}</p>}
      <div className="flex flex-wrap items-center gap-2">
        {newerId ? (
          <Button asChild variant="secondary" size="sm">
            <Link href={lectureHref(newerId)}>
              {t.viewRegenerated}
            </Link>
          </Button>
        ) : (
          <Button size="sm" variant="secondary" loading={regenerate.isPending} onClick={regenerate.start}>
            {t.regenerate}
          </Button>
        )}
        {regenerate.error && <p role="alert" className="text-sm text-danger">{describePunisherError(regenerate.error)}</p>}
      </div>
    </div>
  );
}
