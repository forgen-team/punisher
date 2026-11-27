import { ListOrdered } from "lucide-react";
import { Button, Sheet } from "@forgen-team/ui";
import type { LectureSection } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { SectionToc } from "../../lecture-viewer";
import { messages } from "../i18n";

/**
 * 보조 열이 없는 폭의 본문 위 고정 바 — 강의 화면 바와 같은 모양, 이 화면에는 [목차] 하나.
 * 섹션이 아직 없으면(쓰는 중) 눌리지 않게 두되 자리는 지켜, 해설이 생길 때 본문이 밀리지 않는다.
 */
export function InquiryTocBar({ disabled, onOpen }: { disabled: boolean; onOpen: () => void }) {
  const t = useT(messages);
  return (
    <div
      data-lecture-readbar=""
      className="sticky top-[var(--app-header-offset,0px)] z-nav mb-4 flex gap-2 border-b border-line bg-canvas py-2"
    >
      <Button size="sm" variant="secondary" disabled={disabled} onClick={onOpen}>
        <ListOrdered aria-hidden="true" />
        {t.toc}
      </Button>
    </div>
  );
}

interface InquiryTocSheetProps {
  open: boolean;
  /** 닫기 — after 가 있으면 닫힌 뒤(스크롤 잠금이 풀린 뒤) 부른다 */
  onClose: (after?: () => void) => void;
  sections: LectureSection[];
  totalMinutes: number;
}

/** 목차 시트(주소 ?panel=toc). 항목을 고르면 시트를 닫고 닫힌 뒤 그 섹션으로 간다. */
export function InquiryTocSheet({ open, onClose, sections, totalMinutes }: InquiryTocSheetProps) {
  const t = useT(messages);
  return (
    <Sheet
      open={open}
      onOpenChange={(o) => { if (!o) { onClose(); } }}
      title={t.toc}
      description={t.tocTotal(totalMinutes)}
      desktop="side"
    >
      <SectionToc sections={sections} totalMinutes={totalMinutes} onNavigate={onClose} hideHeading />
    </Sheet>
  );
}
