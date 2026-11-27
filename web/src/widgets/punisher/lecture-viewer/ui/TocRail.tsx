
import { useEffect, useId, useRef, useState } from "react";
import { ListOrdered } from "lucide-react";
import { IconButton } from "@forgen-team/ui";
import { cn, focusRing } from "@/src/shared/lib/cn";
import { eventWithin } from "@/src/shared/lib/dom-root";
import { SECTION_KIND_LABELS } from "@/src/entities/punisher";
import { scrollToAnchor } from "@/src/features/punisher/lecture-progress";
import type { LectureSection } from "@/src/shared/api/punisher";
import { SectionToc } from "./SectionToc";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

interface TocRailProps {
  sections: LectureSection[];
  totalMinutes: number;
  activeAnchor: string;
}

/**
 * 접은 목차(lg~2xl 미만): 본문 가독 폭을 지키려고 왼쪽 열을 아이콘 + 섹션 점으로 줄인다.
 * 점은 읽은 위치까지 채워지고(진행), 누르면 그 섹션으로 간다. 아이콘을 누르면 목차가 본문 위에 펼쳐진다(폭을 밀지 않음).
 */
export function TocRail({ sections, totalMinutes, activeAnchor }: TocRailProps) {
  const t = useT(messages);
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const panelId = useId();
  const activeIndex = sections.findIndex((s) => s.anchor === activeAnchor);

  useEffect(() => {
    if (!open) { return; }
    const onPointer = (event: PointerEvent) => {
      if (!eventWithin(event, rootRef.current)) { setOpen(false); }
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") { setOpen(false); }
    };
    document.addEventListener("pointerdown", onPointer);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onPointer);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div ref={rootRef} className="relative flex flex-col items-center gap-2">
      <IconButton aria-label={open ? t.tocCollapse : t.tocExpand} aria-expanded={open} aria-controls={panelId} onClick={() => setOpen((v) => !v)} icon={<ListOrdered aria-hidden="true" />} />
      <ol aria-label={t.sectionPositions} className="flex flex-col items-center">
        {sections.map((s, index) => {
          const label = `${index + 1}. ${s.title || SECTION_KIND_LABELS[s.kind] || s.kind}`;
          const current = index === activeIndex;
          let dot = "size-2 bg-line-strong";
          if (current) { dot = "size-2.5 bg-accent-solid"; }
          else if (index < activeIndex) { dot = "size-2 bg-fg-muted"; }
          return (
            <li key={s.anchor || index}>
              <button
                type="button"
                title={label}
                aria-label={label}
                aria-current={current ? "location" : undefined}
                onClick={() => scrollToAnchor(s.anchor)}
                className={cn("flex size-6 items-center justify-center rounded-md hover:bg-canvas-hover", focusRing)}
              >
                <span aria-hidden="true" className={cn("block rounded-full transition-all", dot)} />
              </button>
            </li>
          );
        })}
      </ol>
      {open && (
        <div id={panelId} className="absolute left-full top-0 z-panel ml-2 w-64 rounded-lg border border-line bg-canvas-elevated p-2 shadow-md">
          <SectionToc sections={sections} totalMinutes={totalMinutes} activeAnchor={activeAnchor} onNavigate={(scroll) => { setOpen(false); requestAnimationFrame(scroll); }} />
        </div>
      )}
    </div>
  );
}
