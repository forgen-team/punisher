
import { TermPopover, type DeeperOrigin } from "@/src/features/punisher/term-popover";
import type { LectureTerm } from "../lib/lectureTerms";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

interface TermListProps {
  terms: LectureTerm[];
  onBeforeDeeper: () => DeeperOrigin | void;
}

/** 이 강의에 나온 용어 — 본문의 용어 링크와 같은 설명 팝오버를 연다(나온 순서). */
export function TermList({ terms, onBeforeDeeper }: TermListProps) {
  const t = useT(messages);
  if (terms.length === 0) {
    return <p className="py-4 text-sm text-fg-muted">{t.noTerms}</p>;
  }
  return (
    <ul role="list" className="flex flex-wrap gap-x-4">
      {terms.map((term) => (
        <li key={term.conceptId}>
          <TermPopover term={term.term} conceptId={term.conceptId} onBeforeDeeper={onBeforeDeeper} />
        </li>
      ))}
    </ul>
  );
}
