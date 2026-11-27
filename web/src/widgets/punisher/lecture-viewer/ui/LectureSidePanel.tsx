import type { ReactNode } from "react";
import { ListOrdered, MessagesSquare } from "lucide-react";
import { Button, SegmentedControl, SegmentedControlItem, Sheet } from "@forgen-team/ui";
import { useNotes, useQuestions } from "@/src/entities/punisher";
import { countPassageThreads, passageQuestionIds, useReadingThreads } from "@/src/features/punisher/reading-assist";
import type { LectureSection } from "@/src/shared/api/punisher";
import type { LecturePanel } from "../lib/useLecturePanel";
import { NotesPanel } from "./NotesPanel";
import { QuestionsPanel } from "./QuestionsPanel";
import { SectionToc } from "./SectionToc";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** "질문과 메모 n" 의 n — 구절 대화 + 구절에 묶이지 않은 질문 + 메모. */
export function useAssistCount(lectureId: string): number {
  const threads = useReadingThreads();
  const questions = useQuestions(lectureId);
  const notes = useNotes({ lectureId });
  const inPassages = threads ? passageQuestionIds(threads.passages) : null;
  const loose = (questions.data ?? []).filter((q) => !inPassages || !inPassages.has(q.id)).length;
  return countPassageThreads(threads?.passages ?? []) + loose + (notes.data?.length ?? 0);
}

function assistLabel(label: string, count: number): string {
  return count > 0 ? `${label} ${count}` : label;
}

interface NotesAndQuestionsProps {
  lectureId: string;
  conceptId: string;
  getAnchor: () => string;
  onJump?: (scroll: () => void) => void;
}

/** 질문과 메모 — 이 강의에 남긴 질문(구절 대화 포함)과 메모를 한 곳에. */
export function NotesAndQuestions({ lectureId, conceptId, getAnchor, onJump }: NotesAndQuestionsProps) {
  const t = useT(messages);
  return (
    <div className="flex flex-col gap-6">
      <section aria-label={t.tabQuestions} className="flex flex-col gap-2">
        <h3 className="text-xs font-semibold text-fg-secondary">{t.tabQuestions}</h3>
        <QuestionsPanel lectureId={lectureId} getAnchor={getAnchor} onJump={onJump} />
      </section>
      <section aria-label={t.tabNotes} className="flex flex-col gap-2 border-t border-line pt-4">
        <h3 className="text-xs font-semibold text-fg-secondary">{t.tabNotes}</h3>
        <NotesPanel lectureId={lectureId} conceptId={conceptId} getAnchor={getAnchor} onJump={onJump} />
      </section>
    </div>
  );
}

/** 폰·좁은 폭 본문 고정 바 — [목차] [질문과 메모 n] 두 버튼만(보조 열이 있으면 그리지 않는다). */
export function ReadingBar({ count, onOpen }: { count: number; onOpen: (panel: LecturePanel) => void }) {
  const t = useT(messages);
  return (
    <div
      data-lecture-readbar=""
      className="sticky top-[var(--app-header-offset,0px)] z-nav mb-4 flex gap-2 border-b border-line bg-canvas py-2"
    >
      <Button size="sm" variant="secondary" onClick={() => onOpen("toc")}>
        <ListOrdered aria-hidden="true" />
        {t.toc}
      </Button>
      <Button size="sm" variant="secondary" onClick={() => onOpen("notes")}>
        <MessagesSquare aria-hidden="true" />
        {assistLabel(t.questionsAndNotes, count)}
      </Button>
    </div>
  );
}

interface ReadingAsideProps {
  tab: LecturePanel;
  onTabChange: (tab: LecturePanel) => void;
  count: number;
  toc: ReactNode;
  notes: ReactNode;
}

/** 데스크톱 오른쪽 보조 열 — 탭 [목차] [질문과 메모 n]. 화면 높이 안에서 따로 스크롤한다. */
export function ReadingAside({ tab, onTabChange, count, toc, notes }: ReadingAsideProps) {
  const t = useT(messages);
  return (
    <div
      className="sticky top-[calc(var(--app-header-offset,0px)+1rem)] flex max-h-[calc(100dvh-var(--app-header-offset,0px)-2rem)] flex-col gap-3 overflow-y-auto pb-6"
    >
      <SegmentedControl aria-label={t.readingTools} value={tab} onValueChange={(v) => onTabChange(v as LecturePanel)}>
        <SegmentedControlItem value="toc">{t.toc}</SegmentedControlItem>
        <SegmentedControlItem value="notes">{assistLabel(t.questionsAndNotes, count)}</SegmentedControlItem>
      </SegmentedControl>
      <div>{tab === "toc" ? toc : notes}</div>
    </div>
  );
}

interface PanelSheetsProps {
  panel: LecturePanel | null;
  onClose: (after?: () => void) => void;
  sections: LectureSection[];
  totalMinutes: number;
  lectureId: string;
  conceptId: string;
  getAnchor: () => string;
}

/** 폰·좁은 폭 하단 시트(주소 ?panel=toc|notes). 항목·위치로 가면 시트를 닫고 닫힌 뒤 스크롤한다. */
export function PanelSheets({ panel, onClose, sections, totalMinutes, lectureId, conceptId, getAnchor }: PanelSheetsProps) {
  const t = useT(messages);
  return (
    <>
      <Sheet
        open={panel === "toc"}
        onOpenChange={(o) => { if (!o) { onClose(); } }}
        title={t.toc}
        description={t.totalMinutes(totalMinutes)}
        desktop="side"
      >
        <SectionToc sections={sections} totalMinutes={totalMinutes} onNavigate={onClose} hideHeading />
      </Sheet>
      <Sheet
        open={panel === "notes"}
        onOpenChange={(o) => { if (!o) { onClose(); } }}
        title={t.questionsAndNotes}
        desktop="side"
      >
        <NotesAndQuestions lectureId={lectureId} conceptId={conceptId} getAnchor={getAnchor} onJump={onClose} />
      </Sheet>
    </>
  );
}

/** 생성 중(즉석 강의) 보조 열 자리 — 완성된 강의로 바뀔 때 본문이 옆으로 밀리지 않도록 같은 폭을 미리 잡는다. */
export function LectureAssistPlaceholder({ children }: { children?: ReactNode }) {
  const t = useT(messages);
  return (
    <p className="break-keep rounded-xl border border-line bg-canvas-elevated p-4 text-sm text-fg-muted shadow-xs">
      {children ?? t.assistPlaceholder}
    </p>
  );
}
