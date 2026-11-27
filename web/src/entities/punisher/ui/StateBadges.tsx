import { AlertTriangle } from "lucide-react";
import { Badge } from "@forgen-team/ui";
import type { Lecture, UserConceptState } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { USER_STATE_LABELS_BY_LOCALE } from "./constants";

type BadgeTone = "neutral" | "accent" | "success" | "warning" | "danger";

/** 앱 배지 색 이름 → ui Badge tone */
const BADGE_TONE = { neutral: "neutral", accent: "brand", success: "positive", warning: "warning", danger: "critical" } as const;

const STATE_TONE: Record<UserConceptState, BadgeTone> = {
  completed: "success",
  known: "accent",
  "in_progress": "warning",
  "not_started": "neutral",
};

export function ConceptStateBadge({ state }: { state: UserConceptState }) {
  const locale = useLocale();
  return <Badge variant="weak" tone={BADGE_TONE[STATE_TONE[state] ?? "neutral"]}>{USER_STATE_LABELS_BY_LOCALE[locale][state] ?? state}</Badge>;
}

/**
 * 강의 상태 배지 1개(행당 최대 1개 규칙): 검수 미완료가 즉석 표시보다 우선.
 * 아무 것도 해당하지 않으면 null.
 */
export function LectureReviewBadges({ lecture }: { lecture: Pick<Lecture, "kind" | "status" | "flagged" | "flagReason" | "reviewState"> }) {
  const t = useT(messages);
  // v2.2 reviewState: 검수 전은 작은 배지, 문제 발견(flagged)은 본문 위 배너가 대신한다.
  if (lecture.reviewState === "unreviewed") { return <Badge variant="weak" tone="neutral">{t.reviewUnreviewed}</Badge>; }
  if (lecture.reviewState === "flagged") { return null; }
  const unreviewed = !lecture.reviewState && (lecture.flagged || lecture.status === "needs_revision");
  if (unreviewed) {
    return (
      <Badge variant="weak" tone="warning" title={lecture.flagReason || t.reviewNotPassed}>
        <AlertTriangle className="size-3" aria-hidden="true" />
        {t.reviewIncomplete}
      </Badge>
    );
  }
  if (lecture.kind === "instant") { return <Badge variant="weak" tone="neutral">{t.instantLecture}</Badge>; }
  return null;
}
