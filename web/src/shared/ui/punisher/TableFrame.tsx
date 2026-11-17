
import { useEffect, useRef, useState, type ReactNode } from "react";
import { cn } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/**
 * 본문 표를 감싸는 가로 스크롤 상자. 표가 상자보다 넓을 때만, 아직 끝까지 밀지 않은 쪽 가장자리에 흐림 단서를 둔다
 * (다 들어가면 아무것도 그리지 않는다).
 */
export function TableFrame({ children }: { children: ReactNode }) {
  const t = useT(messages);
  const ref = useRef<HTMLDivElement>(null);
  const [edges, setEdges] = useState({ start: false, end: false });

  useEffect(() => {
    const el = ref.current;
    if (!el) { return undefined; }
    const update = () => {
      const overflow = el.scrollWidth - el.clientWidth > 1;
      const next = { start: overflow && el.scrollLeft > 1, end: overflow && el.scrollLeft < el.scrollWidth - el.clientWidth - 1 };
      setEdges((prev) => (prev.start === next.start && prev.end === next.end ? prev : next));
    };
    update();
    el.addEventListener("scroll", update, { passive: true });
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(update);
    observer?.observe(el);
    if (el.firstElementChild) { observer?.observe(el.firstElementChild); }
    return () => {
      el.removeEventListener("scroll", update);
      observer?.disconnect();
    };
  }, []);

  return (
    <div className="relative my-3">
      <div ref={ref} role="region" aria-label={t.table} tabIndex={0} className="max-w-full overflow-x-auto">
        {children}
      </div>
      <span
        aria-hidden="true"
        className={cn("pointer-events-none absolute inset-y-0 left-0 w-6 bg-gradient-to-r from-canvas to-transparent transition-opacity", edges.start ? "opacity-100" : "opacity-0")}
      />
      <span
        aria-hidden="true"
        className={cn("pointer-events-none absolute inset-y-0 right-0 w-6 bg-gradient-to-l from-canvas to-transparent transition-opacity", edges.end ? "opacity-100" : "opacity-0")}
      />
    </div>
  );
}
