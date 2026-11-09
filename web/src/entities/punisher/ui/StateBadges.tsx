import { AlertTriangle } from "lucide-react";
import { Badge, type BadgeTone } from "@/src/shared/ui/kit";
import type { Lecture, UserConceptState } from "@/src/shared/api/punisher";
import { USER_STATE_LABELS } from "./constants";

const STATE_TONE: Record<UserConceptState, BadgeTone> = {
  completed: "success",
  known: "accent",
  "in_progress": "warning",
  "not_started": "neutral",
};

export function ConceptStateBadge({ state }: { state: UserConceptState }) {
  return <Badge tone={STATE_TONE[state] ?? "neutral"}>{USER_STATE_LABELS[state] ?? state}</Badge>;
}

/**
 * 강의 상태 배지 1개(행당 최대 1개 규칙): 검수 미완료가 즉석 표시보다 우선.
 * 아무 것도 해당하지 않으면 null.
 */
export function LectureReviewBadges({ lecture }: { lecture: Pick<Lecture, "kind" | "status" | "flagged" | "flagReason"> }) {
  const unreviewed = lecture.flagged || lecture.status === "needs_revision";
  if (unreviewed) {
    return (
      <Badge tone="warning" title={lecture.flagReason || "검수 패스를 통과하지 못했습니다"}>
        <AlertTriangle className="size-3" aria-hidden="true" />
        검수 미완료
      </Badge>
    );
  }
  if (lecture.kind === "instant") { return <Badge tone="neutral">즉석 강의</Badge>; }
  return null;
}
