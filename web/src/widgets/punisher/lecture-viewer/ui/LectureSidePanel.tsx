
import { useEffect, useState, type ReactNode } from "react";
import { Layers } from "lucide-react";
import { List, ListRow, SegmentedControl, SegmentedControlItem, Section, Sheet } from "@forgen-team/ui";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { useReadingThreads } from "@/src/features/punisher/reading-assist";
import type { DeeperOrigin } from "@/src/features/punisher/term-popover";
import type { LinkedLecture } from "@/src/shared/api/punisher";
import type { LectureTerm } from "../lib/lectureTerms";
import { NotesPanel } from "./NotesPanel";
import { QuestionsPanel } from "./QuestionsPanel";
import { TermList } from "./TermList";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

type AssistTab = "questions" | "notes";
type T = typeof messages.ko;

function assistTabs(t: T) {
  return [
    { value: "questions", label: t.tabQuestions },
    { value: "notes", label: t.tabNotes },
  ] as const;
}

interface LectureAssistSheetProps {
  open: boolean;
  onClose: () => void;
  lectureId: string;
  conceptId: string;
  /** 시트를 열 때의 읽던 위치 — 질문·메모의 앵커로 쓴다 */
  getAnchor: () => string;
}

/**
 * 질문·메모 시트: 모바일 하단 시트, 데스크톱 오른쪽 패널.
 * 상단 바 버튼으로 열어 하단 탭 바와 겹치는 플로팅 버튼을 두지 않는다.
 */
export function LectureAssistSheet({ open, onClose, lectureId, conceptId, getAnchor }: LectureAssistSheetProps) {
  const t = useT(messages);
  const [tab, setTab] = useState<AssistTab>("questions");

  return (
    <Sheet open={open} onOpenChange={(o) => { if (!o) { onClose(); } }} title={t.questionsAndNotes} desktop="side">
      <div className="mb-3">
        <SegmentedControl aria-label={t.assistTabs} value={tab} onValueChange={(v) => setTab(v as AssistTab)}>
          {assistTabs(t).map((item) => (
            <SegmentedControlItem key={item.value} value={item.value}>{item.label}</SegmentedControlItem>
          ))}
        </SegmentedControl>
      </div>
      <div>
        {tab === "questions" ? (
          <QuestionsPanel lectureId={lectureId} getAnchor={getAnchor} onJump={onClose} />
        ) : (
          <NotesPanel lectureId={lectureId} conceptId={conceptId} getAnchor={getAnchor} onJump={onClose} />
        )}
      </div>
    </Sheet>
  );
}

function panelTabs(t: T) {
  return [...assistTabs(t), { value: "terms", label: t.tabTerms }] as const;
}
type PanelTab = AssistTab | "terms";

interface LectureAssistPanelProps extends Omit<LectureAssistSheetProps, "open" | "onClose"> {
  terms: LectureTerm[];
  onBeforeDeeper: () => DeeperOrigin | void;
  /** 이 강의에서 '더 깊게' 연 강의(있을 때만 아래에 짧게) */
  deeper: LinkedLecture[];
}

/**
 * 넓은 화면(xl ≥1280)용 보조 열 — 읽으면서 바로 쓰는 것만: 질문 · 메모 · 이 강의의 용어(본문 팝오버와 같은 설명),
 * 그리고 '더 깊게' 본 강의가 있으면 그 목록. 시트를 열 필요 없이 본문 옆에 늘 펼쳐 둔다.
 */
export function LectureAssistPanel({ lectureId, conceptId, getAnchor, terms, onBeforeDeeper, deeper }: LectureAssistPanelProps) {
  const t = useT(messages);
  const [tab, setTab] = useState<PanelTab>("questions");
  // 본문에서 새 스레드를 열 때만(쉽게·깊게·질문·대화 n) '질문' 탭으로 돌아가 그 스레드를 펼친다 —
  // 답이 오거나 닫는 등 focus 가 바뀔 때마다 탭을 빼앗지 않는다.
  const openSeq = useReadingThreads()?.openSeq ?? 0;
  useEffect(() => {
    if (openSeq > 0) { setTab("questions"); }
  }, [openSeq]);
  return (
    <div className="flex flex-col gap-6">
      <section aria-label={t.panelSection} className="flex flex-col gap-3">
        <SegmentedControl aria-label={t.panelTabs} value={tab} onValueChange={(v) => setTab(v as PanelTab)}>
          {panelTabs(t).map((item) => (
            <SegmentedControlItem key={item.value} value={item.value}>{item.label}</SegmentedControlItem>
          ))}
        </SegmentedControl>
        <div>
          {tab === "questions" && <QuestionsPanel lectureId={lectureId} getAnchor={getAnchor} />}
          {tab === "notes" && <NotesPanel lectureId={lectureId} conceptId={conceptId} getAnchor={getAnchor} />}
          {tab === "terms" && <TermList terms={terms} onBeforeDeeper={onBeforeDeeper} />}
        </div>
      </section>
      {deeper.length > 0 && (
        <Section title={t.deeperShort} headingLevel={3}>
          <List>
            {deeper.map((l) => (
              <ListRow key={l.id} href={lectureHref(l.id)} leading={<Layers aria-hidden="true" />} title={l.conceptName || l.title} />
            ))}
          </List>
        </Section>
      )}
    </div>
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
