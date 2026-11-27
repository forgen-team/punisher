import { ErrorState, List, ListRow, Section } from "@forgen-team/ui";
import { INQUIRY_STATUS_LABELS_BY_LOCALE, InquiryStatusIcon, inquiryRowTitle } from "@/src/entities/punisher";
import { inquiryHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type Inquiry } from "@/src/shared/api/punisher";
import { formatDate, useLocale, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

const DATE: Intl.DateTimeFormatOptions = { month: "short", day: "numeric" };

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
  const locale = useLocale();
  const t = useT(messages);
  if (error) {
    return (
      <Section variant="card" title={t.inquiries}>
        <ErrorState title={describePunisherError(error)} onRetry={onRetry} />
      </Section>
    );
  }
  if (!inquiries || inquiries.length === 0) { return null; }
  return (
    <Section variant="card" title={t.inquiries} description={t.inquiriesDescription}>
      <List>
        {inquiries.map((q) => (
          <ListRow
            key={q.id}
            href={inquiryHref(q.id)}
            leading={<InquiryStatusIcon status={q.status} />}
            title={inquiryRowTitle(q)}
            meta={`${INQUIRY_STATUS_LABELS_BY_LOCALE[locale][q.status] ?? q.status} · ${formatDate(new Date(q.createdAt), locale, DATE)}`}
          />
        ))}
      </List>
    </Section>
  );
}
