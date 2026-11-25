import { Skeleton } from "@/src/shared/ui/kit";
import { isWitems } from "@/src/shared/config/runtime";
import { useT } from "@/src/shared/i18n";
import { LECTURE_TITLE_ID } from "../../lecture-viewer";
import { messages } from "../i18n";

interface InquiryHeaderProps {
  question: string;
  title?: string;
}

/** 해설 머리: 내가 한 질문(원문) → 해설 제목. 제목이 아직 없으면(쓰는 중) 같은 높이의 자리만 잡아 둔다. */
export function InquiryHeader({ question, title }: InquiryHeaderProps) {
  const t = useT(messages);
  return (
    <header className="flex flex-col gap-2 pt-4">
      <p className="text-sm font-semibold text-accent-text">{t.myQuestion}</p>
      <blockquote className="break-keep border-l-2 border-line-strong pl-3 text-base text-fg-secondary">{question}</blockquote>
      {/* Witems: 해설 제목은 셸이 화면 제목으로 그린다(ctx.setTitle). */}
      {isWitems() ? null : title ? (
        <h1 id={LECTURE_TITLE_ID} className="break-keep text-xl font-semibold text-fg md:text-2xl">
          {title}
        </h1>
      ) : (
        <div aria-hidden="true" className="flex h-7 items-center md:h-8">
          <Skeleton className="w-2/3" />
        </div>
      )}
    </header>
  );
}
