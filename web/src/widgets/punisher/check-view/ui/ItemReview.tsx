
import { Link } from "@/src/shared/lib/router";
import type { ReactNode } from "react";
import { Badge, Button } from "@/src/shared/ui/kit";
import { OUTCOME_LABELS } from "@/src/entities/punisher";
import type { CheckItem } from "@/src/shared/api/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { OUTCOME_TONE } from "./constants";

interface ItemReviewProps {
  item: CheckItem;
  /** 판정 턴의 놓친 기준 */
  missing: string[];
  /** '다시 볼 곳' 링크(앵커 없으면 null) */
  anchorHref: string | null;
  anchorLabel: string;
  /** '내 답이 맞아요'(없으면 버튼을 두지 않는다) */
  onDispute?: () => void;
  disputing?: boolean;
}

function Part({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <p className="text-xs font-semibold text-fg-secondary">{title}</p>
      {children}
    </div>
  );
}

/** 끝난 문항: 판정(+ 이의) · 놓친 기준 · 모범답 · 기준 · 다시 볼 곳. */
export function ItemReview({ item, missing, anchorHref, anchorLabel, onDispute, disputing }: ItemReviewProps) {
  return (
    <div className="flex flex-col gap-3 rounded-lg border border-line p-3 text-sm">
      {item.outcome && (
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={OUTCOME_TONE[item.outcome]}>{OUTCOME_LABELS[item.outcome]}</Badge>
          {onDispute && (
            <Button size="sm" variant="ghost" loading={disputing} onClick={onDispute}>
              내 답이 맞아요
            </Button>
          )}
        </div>
      )}
      {missing.length > 0 && (
        <Part title="놓친 기준">
          <ul className="list-disc space-y-0.5 pl-5 text-fg">
            {missing.map((m, index) => (
              <li key={index}>{m}</li>
            ))}
          </ul>
        </Part>
      )}
      {item.answer && (
        <Part title="모범답">
          <Markdown>{item.answer}</Markdown>
        </Part>
      )}
      {item.must && item.must.length > 0 && (
        <Part title="기준">
          <ul className="list-disc space-y-0.5 pl-5 text-fg">
            {item.must.map((m, index) => (
              <li key={index}>{m}</li>
            ))}
          </ul>
        </Part>
      )}
      {anchorHref && (
        <Part title="다시 볼 곳">
          <Link href={anchorHref} className="text-accent-text underline underline-offset-2">
            {anchorLabel}
          </Link>
        </Part>
      )}
    </div>
  );
}
