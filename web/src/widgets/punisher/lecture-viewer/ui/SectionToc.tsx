
import { Sheet } from "@forgen-team/ui";
import { cn, focusRing } from "@/src/shared/lib/cn";
import type { LectureSection } from "@/src/shared/api/punisher";
import { SECTION_KIND_LABELS } from "@/src/entities/punisher";
import { scrollToAnchor } from "@/src/features/punisher/lecture-progress";
import { sectionMinutes } from "../lib/sectionMinutes";
import { messages } from "../i18n";
import { useT } from "@/src/shared/i18n";

interface SectionTocProps {
  sections: LectureSection[];
  totalMinutes: number;
  /** 항목 선택 후 (시트 닫기 등) */
  onNavigate?: () => void;
  /** 시트 안에서는 제목 줄을 생략 */
  hideHeading?: boolean;
  /** 지금 읽는 섹션(강조) */
  activeAnchor?: string;
}

export function SectionToc({ sections, totalMinutes, onNavigate, hideHeading = false, activeAnchor }: SectionTocProps) {
  const t = useT(messages);
  return (
    <nav aria-label={t.tocNav} className="flex flex-col gap-1 text-sm">
      {!hideHeading && (
        <p className="px-2 py-1 font-semibold text-fg">
          {t.toc} <span className="font-normal text-fg-muted">· {t.aboutMinutes(totalMinutes)}</span>
        </p>
      )}
      <ol className="flex flex-col">
        {sections.map((s, index) => (
          <li key={s.anchor || index}>
            <a
              // ?at= — 새 탭으로 열어도 그 자리로 간다(#해시는 Witems 셸이 앱에 넘기지 않는다). 보통 클릭은 이 화면 안에서 스크롤.
              href={`?at=${encodeURIComponent(s.anchor)}`}
              aria-current={activeAnchor && s.anchor === activeAnchor ? "location" : undefined}
              onClick={(event) => {
                if (event.metaKey || event.ctrlKey || event.shiftKey || event.button !== 0) { return; }
                onNavigate?.();
                // 시트가 닫히며 스크롤 잠금이 풀린 다음 프레임에 이동한다.
                event.preventDefault();
                requestAnimationFrame(() => scrollToAnchor(s.anchor));
              }}
              className={cn(
                // 두 줄 제목: 번호 칸을 따로 둬 둘째 줄이 제목 첫 글자에 맞춰지고(내어쓰기), 위아래 여백으로 항목 사이를 띄운다.
                "flex min-h-touch items-start justify-between gap-2 rounded-md px-2 py-2.5 leading-snug text-fg-secondary hover:bg-canvas-hover hover:text-fg md:min-h-9 md:py-2",
                "aria-[current=location]:bg-canvas-hover aria-[current=location]:font-medium aria-[current=location]:text-fg",
                focusRing,
              )}
            >
              <span className="flex min-w-0 gap-1.5">
                <span className="w-4 shrink-0 text-right tabular-nums">{index + 1}.</span>
                <span className="min-w-0 break-keep">{s.title || SECTION_KIND_LABELS[s.kind] || s.kind}</span>
              </span>
              {sectionMinutes(s) > 0 && <span className="shrink-0 pt-px text-xs text-fg-muted">{t.minutes(sectionMinutes(s))}</span>}
            </a>
          </li>
        ))}
      </ol>
    </nav>
  );
}

interface TocSheetProps {
  open: boolean;
  onClose: () => void;
  sections: LectureSection[];
  totalMinutes: number;
}

/** 모바일·태블릿 목차 시트 (lg 이상은 왼쪽 고정 목차). */
export function TocSheet({ open, onClose, sections, totalMinutes }: TocSheetProps) {
  const t = useT(messages);
  return (
    <Sheet open={open} onOpenChange={(o) => { if (!o) { onClose(); } }} title={t.toc} description={t.totalMinutes(totalMinutes)}>
      <SectionToc sections={sections} totalMinutes={totalMinutes} onNavigate={onClose} hideHeading />
    </Sheet>
  );
}
