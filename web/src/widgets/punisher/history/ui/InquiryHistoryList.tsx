import { Link } from "@/src/shared/lib/router";
import { EmptyState, ErrorState, List, ListRow, ListSkeleton } from "@/src/shared/ui/kit";
import { INQUIRY_STATUS_LABELS, InquiryStatusIcon, inquiryRowTitle, useInquiries } from "@/src/entities/punisher";
import { inquiryHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type Inquiry } from "@/src/shared/api/punisher";
import { historyDay } from "../lib/historyDate";


function metaOf(q: Inquiry): string {
  return [INQUIRY_STATUS_LABELS[q.status] ?? q.status, q.readAt ? "다 봄" : "", historyDay(q.createdAt)]
    .filter(Boolean)
    .join(" · ");
}

/** '이 질문에 답하기'로 물은 것 전체(다 본 것 포함, 최신순). */
export function InquiryHistoryList() {
  const inquiries = useInquiries();
  const items = inquiries.data ?? [];
  if (inquiries.isLoading) { return <ListSkeleton rows={5} />; }
  if (inquiries.error) {
    return <ErrorState message={describePunisherError(inquiries.error)} onRetry={() => inquiries.refetch()} />;
  }
  if (items.length === 0) {
    return <EmptyState title="아직 물어본 질문이 없어요" description="학습 홈의 입력 상자에서 '이 질문에 답하기'로 물어볼 수 있어요." />;
  }
  // 넓은 화면(xl)은 두 열 — 짧은 행이 화면 끝까지 늘어나지 않게.
  return (
    <List className="xl:grid xl:grid-cols-2 xl:gap-x-8 xl:[&>li:nth-last-child(2):nth-child(odd)]:border-b-0">
      {items.map((q) => (
        <ListRow
          key={q.id}
          href={inquiryHref(q.id)}
          linkAs={Link}
          leading={<InquiryStatusIcon status={q.status} />}
          title={inquiryRowTitle(q)}
          meta={metaOf(q)}
        />
      ))}
    </List>
  );
}
