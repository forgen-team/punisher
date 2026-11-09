
import { useId, useState } from "react";
import { BottomSheet, SegmentedTabs } from "@/src/shared/ui/kit";
import { NotesPanel } from "./NotesPanel";
import { QuestionsPanel } from "./QuestionsPanel";

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
    <BottomSheet open={open} onClose={onClose} title="질문과 메모" description="지금 읽는 위치에 연결됩니다" desktop="side">
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
