import type { ReactNode } from "react";
import type { LectureContent, LectureSection } from "@/src/shared/api/punisher";
import { SECTION_KIND_LABELS } from "@/src/entities/punisher";
import { anchorElementId } from "@/src/features/punisher/lecture-progress";
import type { DeeperOrigin } from "@/src/features/punisher/term-popover";
import { PassageMarkers } from "@/src/features/punisher/reading-assist";
import { useSummaryRecalled } from "@/src/features/punisher/lecture-exercise";
import type { LectureBlock } from "@/src/shared/api/punisher";
import { groupBlocks, type BlockGroup } from "../lib/blockGroups";
import { layoutAroundSummary } from "../lib/afterSummary";
import { sectionMinutes } from "../lib/sectionMinutes";
import { BlockGroupRenderer } from "./blocks";
import { messages } from "../i18n";
import { useT } from "@/src/shared/i18n";

function anchorProps(anchor: string) {
  return anchor ? { id: anchorElementId(anchor), "data-lecture-anchor": anchor } : {};
}

/** 묶음의 첫 블록 앵커는 감싼 div 에, 접혀 있는 둘째 블록 앵커는 빈 표식으로 남겨 진도 복원이 끊기지 않게 한다. */
function groupAnchors(group: BlockGroup): [string, string] {
  if (group.type === "pause") { return [group.pause.anchor, group.output?.anchor ?? ""]; }
  if (group.type === "exercise") { return [group.exercise.anchor, group.answer?.anchor ?? ""]; }
  return [group.block.anchor, ""];
}

interface LectureBodyProps {
  content: Pick<LectureContent, "sections">;
  onBeforeDeeper?: () => DeeperOrigin | void;
  /** 섹션 제목 아래 안내(예: 질문 해설의 '내 환경 추정'은 추정임을 표시) */
  sectionNote?: (section: LectureSection) => ReactNode;
}

/** 강의 본문 — 16px(--text-md) / 행간 1.75 (모바일·데스크톱 동일, 문단·목록·강조 상자가 물려받는다), 읽기 폭은 부모 열이 제한한다. */
export function LectureBody({ content, onBeforeDeeper, sectionNote }: LectureBodyProps) {
  const t = useT(messages);
  return (
    // 글자 크기는 설정 > 읽기(이 기기)의 배율(--reading-scale, LectureLayout 이 건다).
    <div className="flex flex-col gap-10 text-[length:calc(1rem*var(--reading-scale,1))] leading-[1.75] text-fg">
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
              {sectionMinutes(section) > 0 && ` · ${t.aboutMinutes(sectionMinutes(section))}`}
            </p>
            <h2 id={`h-${section.anchor || index}`} className="text-xl font-semibold text-fg">
              {section.title || SECTION_KIND_LABELS[section.kind] || section.kind}
            </h2>
          </header>
          <PassageMarkers anchor={section.anchor} />
          {sectionNote?.(section)}
          <SectionBlocks section={section} onBeforeDeeper={onBeforeDeeper} />
        </section>
      ))}
    </div>
  );
}

function Blocks({ blocks, onBeforeDeeper }: { blocks: LectureBlock[]; onBeforeDeeper?: LectureBodyProps["onBeforeDeeper"] }) {
  return (
    <>
      {groupBlocks(blocks).map((group, gi) => {
        const [first, second] = groupAnchors(group);
        return (
          <div key={first || `g${gi}`} {...anchorProps(first)} className="scroll-mt-28">
            <BlockGroupRenderer group={group} onBeforeDeeper={onBeforeDeeper} />
            {second && <span {...anchorProps(second)} className="block scroll-mt-28" />}
            <PassageMarkers anchor={first} />
          </div>
        );
      })}
    </>
  );
}

/** judgment_next 는 요약을 떠올린 뒤에 다음 개념(지식 지도 링크)이 보인다(lib/afterSummary). */
function SectionBlocks({ section, onBeforeDeeper }: { section: LectureSection; onBeforeDeeper?: LectureBodyProps["onBeforeDeeper"] }) {
  const layout = section.kind === "judgment_next" ? layoutAroundSummary(section.blocks) : null;
  if (!layout) { return <Blocks blocks={section.blocks} onBeforeDeeper={onBeforeDeeper} />; }
  return (
    <>
      <Blocks blocks={[...layout.before, layout.summary]} onBeforeDeeper={onBeforeDeeper} />
      <AfterSummary summary={layout.summary}>
        <Blocks blocks={layout.after} onBeforeDeeper={onBeforeDeeper} />
      </AfterSummary>
    </>
  );
}

function AfterSummary({ summary, children }: { summary: LectureBlock; children: ReactNode }) {
  const recalled = useSummaryRecalled(summary.anchor, summary.text ?? "");
  return recalled ? <>{children}</> : null;
}
