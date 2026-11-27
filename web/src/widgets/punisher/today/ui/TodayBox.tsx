import type { ReactNode } from "react";
import { Button, List, ListSkeleton } from "@forgen-team/ui";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { cn } from "@/src/shared/lib/cn";

/**
 * 오늘 상자·묶음은 ui TodaySection·TodayBundle(T-home). 여기는 그 안에 넣는 작은 조각만 둔다.
 * 묶음 안 행 목록 — 상자 선과 겹치지 않게 마지막 행 아래 선을 뺀다(묶음 머리가 위 선을 가진다).
 */
export function BundleList({ children, className }: { children: ReactNode; className?: string }) {
  return <List className={cn("[&>li:last-child]:border-b-0", className)}>{children}</List>;
}

/** 상자 안 여백 칸(입력·숫자 줄·빈 상태). */
export function BoxPad({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("px-4 pb-3 pt-2", className)}>{children}</div>;
}

/** 로딩 — 상자마다 같은 모양(상자 머리는 그대로 두고 안에 행 스켈레톤). */
export function BoxSkeleton({ rows = 3 }: { rows?: number }) {
  return (
    <div aria-busy="true" className="px-4">
      <ListSkeleton rows={rows} />
    </div>
  );
}

/** "n개 더" / "접기" — 묶음 행이 3개를 넘을 때. */
export function MoreToggle({ hidden, expanded, onToggle }: { hidden: number; expanded: boolean; onToggle: () => void }) {
  const t = useT(messages);
  if (hidden <= 0) { return null; }
  return (
    <div className="border-t border-line-subtle px-2 py-1">
      <Button variant="ghost" size="sm" aria-expanded={expanded} onClick={onToggle}>
        {expanded ? t.collapse : t.more(hidden)}
      </Button>
    </div>
  );
}
