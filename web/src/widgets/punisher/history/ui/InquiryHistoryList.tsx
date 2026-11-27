import { EmptyState, ErrorState, List, ListRow, ListSkeleton } from "@forgen-team/ui";
import { INQUIRY_STATUS_LABELS_BY_LOCALE, InquiryStatusIcon, inquiryRowTitle, useInquiries } from "@/src/entities/punisher";
import { inquiryHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type Inquiry } from "@/src/shared/api/punisher";
import { translate, useLocale, useT, type Locale } from "@/src/shared/i18n";
import { historyDay } from "../lib/historyDate";
import { messages } from "../i18n";


function metaOf(q: Inquiry, locale: Locale): string {
  return [INQUIRY_STATUS_LABELS_BY_LOCALE[locale][q.status] ?? q.status, q.readAt ? translate(messages, locale).read : "", historyDay(q.createdAt, locale)]
    .filter(Boolean)
    .join(" · ");
}

/** '이 질문에 답하기'로 물은 것 전체(다 본 것 포함, 최신순). */
export function InquiryHistoryList() {
  const locale = useLocale();
  const t = useT(messages);
  const inquiries = useInquiries();
  const items = inquiries.data ?? [];
  if (inquiries.isLoading) { return <ListSkeleton rows={5} />; }
  if (inquiries.error) {
    return <ErrorState title={describePunisherError(inquiries.error)} onRetry={() => inquiries.refetch()} />;
  }
  if (items.length === 0) {
    return <EmptyState title={t.noInquiries} description={t.noInquiriesDescription} />;
  }
  // 넓은 화면(xl)은 두 열 — 짧은 행이 화면 끝까지 늘어나지 않게.
  return (
    <List className="xl:grid xl:grid-cols-2 xl:gap-x-8 xl:[&>li:nth-last-child(2):nth-child(odd)]:border-b-0">
      {items.map((q) => (
        <ListRow
          key={q.id}
          href={inquiryHref(q.id)}
          leading={<InquiryStatusIcon status={q.status} />}
          title={inquiryRowTitle(q)}
          meta={metaOf(q, locale)}
        />
      ))}
    </List>
  );
}
