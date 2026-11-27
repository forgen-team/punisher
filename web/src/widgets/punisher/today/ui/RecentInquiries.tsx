import { ListRow } from "@forgen-team/ui";
import { Link } from "@/src/shared/lib/router";
import { InquiryStatusIcon, inquiryRowTitle, useInquiries } from "@/src/entities/punisher";
import { inquiryHref } from "@/src/features/punisher/instant-start";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { BundleHead, BundleList } from "./TodayBox";

export const INQUIRY_HISTORY_PATH = "/history?tab=notes&kind=inquiry";

/**
 * 묻기·새 주제 › 최근 질문 — 목록은 기록 › 질문·메모로 옮기고 여기는 머리 한 줄 "최근 질문 n · 모두 보기".
 * 지금 쓰는 중인 해설이 있으면 그 행 1개만 둔다(보낸 뒤 돌아왔을 때 진행을 볼 곳). 다 본 질문만 있거나 없으면 그리지 않는다.
 */
export function RecentInquiries() {
  const t = useT(messages);
  const inquiries = useInquiries({ open: true });
  const list = inquiries.data ?? [];
  // 불러오기 실패면 숫자 없이 머리 한 줄만 둔다(기록으로 가는 길은 남긴다).
  if (list.length === 0 && !inquiries.error) { return null; }
  const pending = list.find((q) => q.status === "pending");
  return (
    <div role="group" aria-label={t.recentInquiries}>
      <BundleHead
        name={t.recentInquiries}
        info={list.length > 0 ? String(list.length) : undefined}
        more={
          <Link href={INQUIRY_HISTORY_PATH} aria-label={t.allInquiries} className="text-accent-text underline-offset-2 hover:underline">
            {t.viewAll}
          </Link>
        }
      />
      {pending && (
        <BundleList>
          <ListRow href={inquiryHref(pending.id)} leading={<InquiryStatusIcon status={pending.status} />} title={inquiryRowTitle(pending)} meta={t.inquiryWriting} />
        </BundleList>
      )}
    </div>
  );
}
