import * as React from "react";
import { cn } from "../lib/utils";

export interface ScrollRowProps extends React.HTMLAttributes<HTMLDivElement> {
  /** 안쪽 줄(가로 스크롤)의 클래스 */
  innerClassName?: string;
}

/**
 * ScrollRow — 가로로 넘치는 칩 줄. 더 있는 쪽 끝을 흐리게(페이드) 해 "옆으로 더 있음"을 알린다.
 * 끝까지 밀면 그쪽 페이드가 사라진다. 스크롤바는 숨긴다.
 */
export function ScrollRow({ className, innerClassName, children, ...props }: ScrollRowProps) {
  const ref = React.useRef<HTMLDivElement>(null);
  const [edges, setEdges] = React.useState({ start: false, end: false });
  const update = React.useCallback(() => {
    const el = ref.current;
    if (!el) return;
    const start = el.scrollLeft > 2;
    const end = el.scrollLeft + el.clientWidth < el.scrollWidth - 2;
    setEdges((prev) => (prev.start === start && prev.end === end ? prev : { start, end }));
  }, []);
  React.useEffect(() => {
    update();
    const el = ref.current;
    if (!el || typeof ResizeObserver === "undefined") return undefined;
    const observer = new ResizeObserver(update);
    observer.observe(el);
    return () => observer.disconnect();
  }, [update]);
  return (
    <div className={cn("relative min-w-0", className)} {...props}>
      <div ref={ref} onScroll={update} className={cn("flex gap-2 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden", innerClassName)}>
        {children}
      </div>
      <span
        aria-hidden="true"
        className={cn("pointer-events-none absolute inset-y-0 left-0 w-8 bg-gradient-to-r from-canvas to-transparent transition-opacity duration-fast", edges.start ? "opacity-100" : "opacity-0")}
      />
      <span
        aria-hidden="true"
        className={cn("pointer-events-none absolute inset-y-0 right-0 w-10 bg-gradient-to-l from-canvas to-transparent transition-opacity duration-fast", edges.end ? "opacity-100" : "opacity-0")}
      />
    </div>
  );
}
