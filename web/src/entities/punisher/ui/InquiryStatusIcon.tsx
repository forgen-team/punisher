import { AlertTriangle, MessageCircleQuestion } from "lucide-react";
import { ProgressCircle } from "@forgen-team/ui";
import type { Inquiry, InquiryStatus } from "@/src/shared/api/punisher";

/** 질문 해설 목록 행의 앞 아이콘 — 쓰는 중 스피너, 실패 경고, 완료 질문 아이콘(홈·기록·지도 공통). */
export function InquiryStatusIcon({ status }: { status: InquiryStatus }) {
  if (status === "pending") { return <ProgressCircle size="24" tone="inherit" aria-hidden />; }
  if (status === "failed") { return <AlertTriangle aria-hidden="true" className="text-warning" />; }
  return <MessageCircleQuestion aria-hidden="true" />;
}

/** 목록 행 제목: 해설 제목(짧고 요점) — 아직 없으면 질문 원문. */
export function inquiryRowTitle(q: Pick<Inquiry, "question" | "content">): string {
  return q.content?.title || q.question;
}
