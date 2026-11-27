import type { ReactNode } from "react";
import { Button, List, ListSkeleton } from "@forgen-team/ui";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { cn } from "@/src/shared/lib/cn";

interface TodayBoxProps {
  title: string;
  /** 머리의 합계(0·빈 값이면 숨김) */
  count?: number;
  children: ReactNode;
}

/**
 * 오늘의 종류 상자(T-home) — 선 한 겹, 머리 = 이름 + 합계. 안의 묶음·입력은 가는 선으로만 나눈다(카드 안 카드 없음).
 */
export function TodayBox({ title, count, children }: TodayBoxProps) {
  return (
    <section aria-label={title} className="min-w-0 border border-line bg-canvas-elevated">
      <div className="flex items-baseline gap-2 border-b border-line px-4 py-3">
        <h2 className="text-base font-semibold text-fg">{title}</h2>
        {count ? <span className="text-sm tabular-nums text-fg-muted">{count}</span> : null}
      </div>
      <div className="divide-y divide-line-subtle">{children}</div>
    </section>
  );
}

interface BundleProps {
  name: string;
  /** 개수·상태 한 줄 */
  info?: ReactNode;
  /** 오른쪽 끝 링크 1개 */
  more?: ReactNode;
  children?: ReactNode;
}

/** 묶음 머리 한 줄(32px: 이름 굵게 · 개수·상태 · 오른쪽 끝 링크). */
export function BundleHead({ name, info, more }: Omit<BundleProps, "children">) {
  return (
    <div className="flex min-h-8 items-center gap-2 bg-canvas-subtle px-4 text-xs text-fg-secondary">
      <span className="font-semibold text-fg">{name}</span>
      {info ? <span className="min-w-0 truncate tabular-nums">{info}</span> : null}
      {more ? <span className="ml-auto shrink-0 whitespace-nowrap">{more}</span> : null}
    </div>
  );
}

/** 묶음 = 머리 한 줄 + 행. */
export function Bundle({ name, info, more, children }: BundleProps) {
  return (
    <div role="group" aria-label={name}>
      <BundleHead name={name} info={info} more={more} />
      {children}
    </div>
  );
}

/** 묶음 안 행 목록 — 상자 선과 겹치지 않게 목록 위·마지막 행 아래 선을 뺀다. */
export function BundleList({ children, className }: { children: ReactNode; className?: string }) {
  return <List className={cn("border-t border-line-subtle [&>li:last-child]:border-b-0", className)}>{children}</List>;
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
