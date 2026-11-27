import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** 해설 머리: 내가 한 질문(원문). 제목은 틀이 화면 제목으로 그린다. */
export function InquiryHeader({ question }: { question: string }) {
  const t = useT(messages);
  return (
    <header className="flex flex-col gap-2 pt-4">
      <p className="text-sm font-semibold text-accent-text">{t.myQuestion}</p>
      <blockquote className="break-keep border-l-2 border-line-strong pl-3 text-base text-fg-secondary">{question}</blockquote>
    </header>
  );
}
