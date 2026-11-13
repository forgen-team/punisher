
import { Link } from "@/src/shared/lib/router";
import { useId, useState, type ReactNode } from "react";
import { Layers } from "lucide-react";
import { BottomSheet, List, ListRow, SegmentedTabs, Section } from "@/src/shared/ui/kit";
import { lectureHref } from "@/src/features/punisher/instant-start";
import type { DeeperOrigin } from "@/src/features/punisher/term-popover";
import type { LinkedLecture } from "@/src/shared/api/punisher";
import type { LectureTerm } from "../lib/lectureTerms";
import { NotesPanel } from "./NotesPanel";
import { QuestionsPanel } from "./QuestionsPanel";
import { TermList } from "./TermList";

type AssistTab = "questions" | "notes";

const TABS = [
  { value: "questions", label: "질문" },
  { value: "notes", label: "메모" },
] as const;

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
  const [tab, setTab] = useState<AssistTab>("questions");
  const panelId = useId();

  return (
    <BottomSheet open={open} onClose={onClose} title="질문과 메모" desktop="side">
      <SegmentedTabs
        aria-label="질문 또는 메모"
        items={TABS.map((t) => ({ ...t, panelId }))}
        value={tab}
        onChange={setTab}
        fill
        className="mb-3"
      />
      <div role="tabpanel" id={panelId}>
        {tab === "questions" ? (
          <QuestionsPanel lectureId={lectureId} getAnchor={getAnchor} onJump={onClose} />
        ) : (
          <NotesPanel lectureId={lectureId} conceptId={conceptId} getAnchor={getAnchor} onJump={onClose} />
        )}
      </div>
    </BottomSheet>
  );
}

const PANEL_TABS = [...TABS, { value: "terms", label: "용어" }] as const;
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
  const [tab, setTab] = useState<PanelTab>("questions");
  const panelId = useId();
  return (
    <div className="flex flex-col gap-6">
      <section aria-label="질문 · 메모 · 용어" className="flex flex-col gap-3">
        <SegmentedTabs aria-label="보조 정보" items={PANEL_TABS.map((t) => ({ ...t, panelId }))} value={tab} onChange={setTab} fill />
        <div role="tabpanel" id={panelId}>
          {tab === "questions" && <QuestionsPanel lectureId={lectureId} getAnchor={getAnchor} />}
          {tab === "notes" && <NotesPanel lectureId={lectureId} conceptId={conceptId} getAnchor={getAnchor} />}
          {tab === "terms" && <TermList terms={terms} onBeforeDeeper={onBeforeDeeper} />}
        </div>
      </section>
      {deeper.length > 0 && (
        <Section title="더 깊게 본 강의" headingLevel={3}>
          <List>
            {deeper.map((l) => (
              <ListRow key={l.id} href={lectureHref(l.id)} linkAs={Link} leading={<Layers aria-hidden="true" />} title={l.conceptName || l.title} titleLines={1} />
            ))}
          </List>
        </Section>
      )}
    </div>
  );
}

/** 생성 중(즉석 강의) 보조 열 자리 — 완성된 강의로 바뀔 때 본문이 옆으로 밀리지 않도록 같은 폭을 미리 잡는다. */
export function LectureAssistPlaceholder({ children }: { children?: ReactNode }) {
  return (
    <p className="break-keep rounded-xl border border-line bg-canvas-elevated p-4 text-sm text-fg-muted shadow-xs">
      {children ?? "강의가 완성되면 여기서 질문·메모를 남기고 용어 설명을 볼 수 있어요."}
    </p>
  );
}
