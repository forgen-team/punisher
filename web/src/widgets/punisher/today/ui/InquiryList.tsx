import { Link } from "@/src/shared/lib/router";
import { ErrorState, List, ListRow, Section } from "@/src/shared/ui/kit";
import { INQUIRY_STATUS_LABELS, InquiryStatusIcon, inquiryRowTitle } from "@/src/entities/punisher";
import { inquiryHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type Inquiry } from "@/src/shared/api/punisher";

const dateFormat = new Intl.DateTimeFormat("ko-KR", { month: "short", day: "numeric" });

interface InquiryListProps {
  inquiries: Inquiry[] | undefined;
  error: unknown;
  onRetry: () => void;
}

/**
 * 학습 홈 '질문 해설' — '이 질문에 답하기'로 물은 것 중 '다 봤어요' 하지 않은 것(최신순). 다 본 질문은 여기서 빠지고
 * 지도(관련 개념)·기록에 남는다. 질문이 없는 사람이 대부분이라 불러오는 동안·비었을 때는 그리지 않는다
 * (스켈레톤이 떴다 사라지며 아래가 밀리지 않게). 행 제목은 해설 제목(없으면 질문).
 */
export function InquiryList({ inquiries, error, onRetry }: InquiryListProps) {
  if (error) {
    return (
      <Section variant="card" title="질문 해설">
        <ErrorState compact message={describePunisherError(error)} onRetry={onRetry} />
      </Section>
    );
  }
  if (!inquiries || inquiries.length === 0) { return null; }
  return (
    <Section variant="card" title="질문 해설" description="다 본 질문은 지도의 관련 개념과 기록에 남아요">
      <List>
        {inquiries.map((q) => (
          <ListRow
            key={q.id}
            href={inquiryHref(q.id)}
            linkAs={Link}
            leading={<InquiryStatusIcon status={q.status} />}
            title={inquiryRowTitle(q)}
            meta={`${INQUIRY_STATUS_LABELS[q.status] ?? q.status} · ${dateFormat.format(new Date(q.createdAt))}`}
          />
        ))}
      </List>
    </Section>
  );
}
