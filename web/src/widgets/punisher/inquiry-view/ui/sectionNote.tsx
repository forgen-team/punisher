import { AlertTriangle, BookCheck } from "lucide-react";
import type { InquiryFollowup, LectureSection } from "@/src/shared/api/punisher";
import { translate, type Locale } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** 후속 문답의 '내가 물은 것' — 문서 흐름(질문 → 해설)으로 읽히게 섹션 제목 아래에 둔다. */
export function FollowupQuestion({ followup }: { followup: Pick<InquiryFollowup, "question" | "quote"> }) {
  return (
    <div className="mb-3 flex flex-col gap-1.5">
      {followup.quote && <blockquote className="break-keep border-l-2 border-line pl-3 text-sm text-fg-muted">{followup.quote}</blockquote>}
      <p className="whitespace-pre-wrap break-keep border-l-2 border-line-strong pl-3 text-base text-fg-secondary">{followup.question}</p>
    </div>
  );
}

/** 근거와 해석을 나눠 보이게: 먼저 답은 추정 포함임을, 원인 사슬은 일반 원리(근거 문서), '내 환경 추정'은 추정임을 표시한다. */
export function inquirySectionNote(section: LectureSection, followups: ReadonlyMap<string, InquiryFollowup>, locale: Locale) {
  const t = translate(messages, locale);
  const followup = followups.get(section.anchor);
  if (followup) { return <FollowupQuestion followup={followup} />; }
  if (section.kind === "summary") {
    return (
      <p className="mb-3 flex items-center gap-1.5 break-keep text-xs text-fg-muted">
        <AlertTriangle className="size-3.5 shrink-0" aria-hidden="true" />
        {t.summaryNote}
      </p>
    );
  }
  if (section.kind === "causes") {
    return (
      <p className="mb-3 flex items-center gap-1.5 text-xs text-fg-muted">
        <BookCheck className="size-3.5 shrink-0" aria-hidden="true" />
        {t.causesNote}
      </p>
    );
  }
  if (section.kind === "assumptions") {
    return (
      <p role="note" className="mb-3 flex gap-2 break-keep rounded-md bg-warning-subtle px-3 py-2 text-sm text-warning">
        <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
        {t.assumptionsNote}
      </p>
    );
  }
  return null;
}
