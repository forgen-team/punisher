import type { ReactNode } from "react";
import type { LectureContent, LectureSection } from "@/src/shared/api/punisher";
import { SECTION_KIND_LABELS } from "@/src/entities/punisher";
import { anchorElementId } from "@/src/features/punisher/lecture-progress";
import type { DeeperOrigin } from "@/src/features/punisher/term-popover";
import { BlockRenderer } from "./blocks";

function anchorProps(anchor: string) {
  return anchor ? { id: anchorElementId(anchor), "data-lecture-anchor": anchor } : {};
}

interface LectureBodyProps {
  content: Pick<LectureContent, "sections">;
  onBeforeDeeper?: () => DeeperOrigin | void;
  /** 섹션 제목 아래 안내(예: 질문 해설의 '내 환경 추정'은 추정임을 표시) */
  sectionNote?: (section: LectureSection) => ReactNode;
}

/** 강의 본문 — 16px / 행간 1.7 (모바일·데스크톱 동일), 읽기 폭은 부모 열이 제한한다. */
export function LectureBody({ content, onBeforeDeeper, sectionNote }: LectureBodyProps) {
  return (
    <div className="flex flex-col gap-10 text-base leading-[1.7] text-fg">
      {content.sections.map((section, index) => (
        <section
          key={section.anchor || index}
          {...anchorProps(section.anchor)}
          aria-labelledby={`h-${section.anchor || index}`}
          className="scroll-mt-28"
        >
          <header className="mb-3 border-b border-line pb-2">
            <p className="text-xs font-medium text-fg-muted">
              {index + 1}. {SECTION_KIND_LABELS[section.kind] ?? section.kind}
              {section.minutes > 0 && ` · 약 ${section.minutes}분`}
            </p>
            <h2 id={`h-${section.anchor || index}`} className="text-xl font-semibold text-fg">
              {section.title || SECTION_KIND_LABELS[section.kind] || section.kind}
            </h2>
          </header>
          {sectionNote?.(section)}
          {section.blocks.map((block, bi) => (
            <div key={block.anchor || bi} {...anchorProps(block.anchor)} className="scroll-mt-28">
              <BlockRenderer block={block} onBeforeDeeper={onBeforeDeeper} />
            </div>
          ))}
        </section>
      ))}
    </div>
  );
}
