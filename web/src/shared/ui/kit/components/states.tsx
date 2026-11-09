import * as React from "react";
import { cn } from "../lib/utils";
import { Button } from "./button";

export interface EmptyStateProps extends Omit<React.HTMLAttributes<HTMLDivElement>, "title"> {
  icon?: React.ReactNode;
  title: React.ReactNode;
  description?: React.ReactNode;
  /** 다음 행동 1개 (Button) */
  action?: React.ReactNode;
  /** 리스트 안 등 좁은 곳용 */
  compact?: boolean;
}

/** EmptyState — 데이터 없음. 아이콘 타일 + 가운데 정렬 텍스트. */
export function EmptyState({ icon, title, description, action, compact = false, className, ...props }: EmptyStateProps) {
  return (
    <div
      className={cn("flex flex-col items-center px-4 text-center", compact ? "py-8" : "py-16", className)}
      {...props}
    >
      {icon ? (
        <div
          className="mb-3 flex h-10 w-10 items-center justify-center rounded-lg border border-line bg-canvas-subtle text-fg-muted shadow-xs [&_svg]:h-5 [&_svg]:w-5"
          aria-hidden="true"
        >
          {icon}
        </div>
      ) : null}
      <p className="text-sm font-semibold tracking-tight text-fg">{title}</p>
      {description ? <p className="mt-1 max-w-sm text-sm text-fg-muted">{description}</p> : null}
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  );
}

export interface ErrorStateProps extends React.HTMLAttributes<HTMLDivElement> {
  /** 사용자에게 보여줄 한국어 문장 (내부 에러 코드 금지) */
  message?: React.ReactNode;
  onRetry?: () => void;
  retrying?: boolean;
  compact?: boolean;
}

/** ErrorState — 실패 안내 + 다시 시도. role="alert"로 낭독된다. */
export function ErrorState({
  message = "불러오지 못했어요.", onRetry, retrying = false, compact = false, className, ...props
}: ErrorStateProps) {
  return (
    <div
      role="alert"
      className={cn("flex flex-col items-center px-4 text-center", compact ? "py-8" : "py-16", className)}
      {...props}
    >
      <p className="text-sm font-medium text-fg">{message}</p>
      {onRetry ? (
        <Button variant="secondary" size="sm" className="mt-3" onClick={onRetry} loading={retrying}>
          다시 시도
        </Button>
      ) : null}
    </div>
  );
}

export interface SkeletonProps extends React.HTMLAttributes<HTMLDivElement> {
  /** line: 텍스트 한 줄 높이 · block: 이미지/영역 */
  variant?: "line" | "block";
  /** line 반복 개수 (마지막 줄은 60% 폭) */
  lines?: number;
}

/** Skeleton — 로딩 자리표시. 스크린 리더에는 숨긴다(부모에 aria-busy 권장). */
export function Skeleton({ variant = "line", lines = 1, className, ...props }: SkeletonProps) {
  const base = "animate-pulse bg-canvas-hover motion-reduce:animate-none";
  if (variant === "block") {
    return <div aria-hidden="true" className={cn(base, "h-24 w-full rounded-md", className)} {...props} />;
  }
  return (
    <div aria-hidden="true" className={cn("space-y-2", className)} {...props}>
      {Array.from({ length: lines }, (_, i) => (
        <div key={i} className={cn(base, "h-3.5 rounded-sm", lines > 1 && i === lines - 1 ? "w-3/5" : "w-full")} />
      ))}
    </div>
  );
}

/** ListRow 모양의 스켈레톤 n개 — 목록 로딩용 */
export function ListSkeleton({ rows = 5, className }: { rows?: number; className?: string }) {
  return (
    <div aria-hidden="true" className={className}>
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="space-y-2 border-b border-line py-3 last:border-b-0">
          <Skeleton className="w-4/5" />
          <Skeleton className="w-2/5 [&>div]:h-3" />
        </div>
      ))}
    </div>
  );
}
