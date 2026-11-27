import { useState } from "react";
import { BookOpen } from "lucide-react";
import { Link } from "@/src/shared/lib/router";
import { Button, EmptyState, ListRow, Menu, useToast } from "@forgen-team/ui";
import { groupByParent, recommendationReason, useConceptDismiss } from "@/src/entities/punisher";
import { ResumeRow, resumeKey, useResumeDismiss } from "@/src/features/punisher/resume-dismiss";
import { describePunisherError, type NextChapterPrompt, type Recommendation } from "@/src/shared/api/punisher";
import { DeeperGroup } from "@/src/shared/ui/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { BUNDLE_LIMIT, NEW_CHAPTER_PATH, chapterDaysLeft, mapNodeHref, recommendationHref } from "../lib/recommendations";
import { BoxPad, Bundle, BundleList, MoreToggle } from "./TodayBox";

/** 공부할 것 › 이어보기 — 읽던 강의가 있을 때만. "더 깊게" 본 강의는 원 강의 아래로 묶는다. 3묶음 넘으면 "n개 더". */
export function ResumeBundle({ items }: { items: Recommendation[] }) {
  const t = useT(messages);
  const resume = useResumeDismiss();
  const [expanded, setExpanded] = useState(false);
  const groups = groupByParent(items, resumeKey, (it) => it.parentLectureId);
  if (groups.length === 0) { return null; }
  const visible = expanded ? groups : groups.slice(0, BUNDLE_LIMIT);
  const hidden = groups.slice(BUNDLE_LIMIT).reduce((sum, group) => sum + 1 + group.children.length, 0);
  const row = (item: Recommendation) => <ResumeRow key={resumeKey(item)} item={item} dismissing={resume.pending} onDismiss={resume.dismiss} />;
  return (
    <Bundle name={t.resume} info={String(items.length)}>
      <BundleList>
        {visible.map(({ item, children }) => (
          <DeeperGroup key={resumeKey(item)} parent={row(item)}>{children.map((child) => row(child))}</DeeperGroup>
        ))}
      </BundleList>
      <MoreToggle hidden={hidden} expanded={expanded} onToggle={() => setExpanded((v) => !v)} />
    </Bundle>
  );
}

function NewChapterLink() {
  const t = useT(messages);
  return (
    <Link href={NEW_CHAPTER_PATH} className="text-accent-text underline-offset-2 hover:underline">
      {t.newChapter}
    </Link>
  );
}

/** 추천 행 = 강의 열기(D13). meta "왜 오늘 · 약 n분", ⋯ [필요 없음] [지도에서 보기]. */
function RecommendationRow({ item, onDismiss, dismissing }: { item: Recommendation; onDismiss: (item: Recommendation) => void; dismissing: boolean }) {
  const t = useT(messages);
  const title = item.conceptName || item.title;
  const reason = recommendationReason(item);
  const minutes = item.estimatedMinutes > 0 ? t.aboutMinutes(item.estimatedMinutes) : "";
  return (
    <ListRow
      href={recommendationHref(item)}
      leading={<BookOpen aria-hidden />}
      title={title}
      meta={[reason, minutes].filter(Boolean).join(" · ") || undefined}
      trailing={
        <Menu
          label={t.recommendationActions(title)}
          items={[
            { key: "dismiss", label: t.dismiss, disabled: dismissing, onSelect: () => onDismiss(item) },
            { key: "map", label: t.viewOnMap, href: mapNodeHref(item.conceptId) },
          ]}
        />
      }
    />
  );
}

interface RecommendationBundleProps {
  items: Recommendation[];
  exhausted: boolean;
  nextChapter: NextChapterPrompt | undefined;
  /** 머리 "강의 약 n분"(0 이면 생략) */
  lectureMinutes: number;
  /** 머리 "하루 n분 목표"(0 이면 생략) */
  dailyMinutes: number;
}

/** 공부할 것 › 오늘의 추천 — 강의만. 빈 상태·소진은 [새 챕터 열기], 남은 강의가 적으면 끝 줄. */
export function RecommendationBundle({ items, exhausted, nextChapter, lectureMinutes, dailyMinutes }: RecommendationBundleProps) {
  const t = useT(messages);
  const toast = useToast();
  const dismiss = useConceptDismiss();
  const [expanded, setExpanded] = useState(false);
  // '필요 없음'은 되돌릴 수 있어 확인 없이 빼고 토스트로 되돌리기(정책 23 §3).
  const dismissItem = (item: Recommendation) =>
    dismiss.mutate({ conceptId: item.conceptId, dismissed: true }, {
      onSuccess: () => {
        toast.undo(t.dismissedToast, () => {
          dismiss.mutate({ conceptId: item.conceptId, dismissed: false }, { onError: () => toast.show(t.undoFailed, { tone: "danger" }) });
        }, t.undo);
      },
      onError: (error) => toast.show(describePunisherError(error), { tone: "danger" }),
    });

  const info = [
    items.length > 0 ? String(items.length) : "",
    lectureMinutes > 0 ? t.lectureMinutes(lectureMinutes) : "",
    dailyMinutes > 0 ? t.dailyGoal(dailyMinutes) : "",
  ].filter(Boolean).join(" · ");
  const visible = expanded ? items : items.slice(0, BUNDLE_LIMIT);
  const days = nextChapter ? chapterDaysLeft(nextChapter) : 0;

  return (
    <Bundle name={t.recommendations} info={info}>
      {items.length > 0 ? (
        <>
          <BundleList>
            {visible.map((item) => (
              <RecommendationRow key={`${item.kind}-${item.conceptId}`} item={item} onDismiss={dismissItem} dismissing={dismiss.isPending} />
            ))}
          </BundleList>
          <MoreToggle hidden={items.length - BUNDLE_LIMIT} expanded={expanded} onToggle={() => setExpanded((v) => !v)} />
          {nextChapter && (
            <p className="flex items-center justify-between gap-3 border-t border-line-subtle px-4 py-2 text-xs text-fg-secondary">
              <span>{days > 0 ? t.daysLeft(days) : t.lessThanDay}</span>
              <NewChapterLink />
            </p>
          )}
        </>
      ) : (
        <BoxPad>
          <EmptyState
            title={exhausted ? t.exhausted : t.noRecommendations}
            action={
              <Button asChild variant="secondary" size="sm">
                <Link href={NEW_CHAPTER_PATH}>{t.newChapter}</Link>
              </Button>
            }
          />
        </BoxPad>
      )}
    </Bundle>
  );
}
