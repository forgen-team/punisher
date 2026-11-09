import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../lib/utils";

/**
 * Badge — 크기 1종. 한 행(ListRow)에 최대 1개.
 * 정보가 아니라 "상태"일 때만 쓴다(예: 새 알림, 영상, 실패). 출처·태그는 meta 텍스트로.
 */
const badgeVariants = cva(
  "inline-flex h-5 shrink-0 items-center gap-1 whitespace-nowrap rounded-md border px-1.5 text-xs font-medium",
  {
    variants: {
      tone: {
        neutral: "border-line bg-canvas-subtle text-fg-secondary",
        accent: "border-transparent bg-accent-subtle text-accent-text",
        success: "border-transparent bg-success-subtle text-success",
        warning: "border-transparent bg-warning-subtle text-warning",
        danger: "border-transparent bg-danger-subtle text-danger",
      },
    },
    defaultVariants: { tone: "neutral" },
  }
);

export type BadgeTone = NonNullable<VariantProps<typeof badgeVariants>["tone"]>;

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  tone?: BadgeTone;
}

export function Badge({ className, tone, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ tone }), className)} {...props} />;
}

export { badgeVariants };
