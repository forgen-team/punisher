
import * as React from "react";
import { cn } from "../lib/utils";

export interface SectionProps extends Omit<React.HTMLAttributes<HTMLElement>, "title"> {
  /** 섹션 제목 (h2). 없으면 제목 줄을 그리지 않는다 */
  title?: React.ReactNode;
  /** 제목 오른쪽 보조 동작 1개 (예: "전체 보기" ghost 버튼) */
  action?: React.ReactNode;
  /** 제목 아래 한 줄 설명 */
  description?: React.ReactNode;
  /** 제목 태그 수준 (기본 h2) */
  headingLevel?: 2 | 3;
  /** card: 내용을 테두리·라운드 카드로 감싼다(대시보드형 묶음). 기본은 상자 없음 */
  variant?: "plain" | "card";
}

/**
 * Section — 제목 + 선택적 동작. 기본은 상자 없음, variant="card" 면 테두리 카드로 감싼다.
 * 카드 안에 카드를 넣지 말 것: 카드 안의 구획은 List 구분선으로 만든다.
 */
export const Section = React.forwardRef<HTMLElement, SectionProps>(function Section(
  { title, action, description, headingLevel = 2, variant = "plain", className, children, ...props },
  ref
) {
  const headingId = React.useId();
  const Heading = headingLevel === 3 ? "h3" : "h2";
  return (
    <section
      ref={ref}
      aria-labelledby={title ? headingId : undefined}
      className={className}
      {...props}
    >
      {title ? (
        <div className="mb-1.5 flex min-h-touch items-center justify-between gap-3">
          <div className="min-w-0">
            <Heading id={headingId} className="truncate text-sm font-semibold tracking-tight text-fg">
              {title}
            </Heading>
            {description ? <p className="mt-0.5 break-keep text-xs text-fg-muted">{description}</p> : null}
          </div>
          {action ? <div className="flex shrink-0 items-center">{action}</div> : null}
        </div>
      ) : null}
      {variant === "card" ? (
        <div className="rounded-xl border border-line bg-canvas-elevated px-4 shadow-xs">{children}</div>
      ) : (
        children
      )}
    </section>
  );
});

export interface PageHeaderProps extends Omit<React.HTMLAttributes<HTMLElement>, "title"> {
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  /** 오른쪽 동작. 모바일에서는 1개(나머지는 Menu로) 권장 */
  actions?: React.ReactNode;
  /** 왼쪽 뒤로가기 등 */
  leading?: React.ReactNode;
  /** 제목 수준(기본 h1). 셸이 화면 h1 을 그리는 곳에 놓인 하위 제목(주제 이름 등)은 2 */
  headingLevel?: 1 | 2;
}

/** PageHeader — 화면 제목(h1) + 부제 + 동작. 배경·그림자 없음. */
export function PageHeader({ title, subtitle, actions, leading, headingLevel = 1, className, ...props }: PageHeaderProps) {
  const Heading = headingLevel === 2 ? "h2" : "h1";
  return (
    <header className={cn("flex items-start gap-3 py-4", className)} {...props}>
      {leading ? <div className="-ml-2 flex shrink-0 items-center">{leading}</div> : null}
      <div className="min-w-0 flex-1">
        <Heading className="truncate text-xl font-semibold tracking-tight text-fg md:text-2xl">{title}</Heading>
        {subtitle ? <p className="mt-1 text-sm text-fg-secondary">{subtitle}</p> : null}
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-1">{actions}</div> : null}
    </header>
  );
}
