
import { TermPopover, type DeeperOrigin } from "@/src/features/punisher/term-popover";
import type { LectureTerm } from "../lib/lectureTerms";

interface TermListProps {
  terms: LectureTerm[];
  onBeforeDeeper: () => DeeperOrigin | void;
}

/** 이 강의에 나온 용어 — 본문의 용어 링크와 같은 설명 팝오버를 연다(나온 순서). */
export function TermList({ terms, onBeforeDeeper }: TermListProps) {
  if (terms.length === 0) {
    return <p className="py-4 text-sm text-fg-muted">이 강의에는 따로 설명하는 용어가 없어요.</p>;
  }
  return (
    <ul role="list" className="flex flex-wrap gap-x-4">
      {terms.map((t) => (
        <li key={t.conceptId}>
          <TermPopover term={t.term} conceptId={t.conceptId} onBeforeDeeper={onBeforeDeeper} />
        </li>
      ))}
    </ul>
  );
}
