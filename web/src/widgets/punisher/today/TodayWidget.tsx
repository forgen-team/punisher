import { useState, type ReactNode } from "react";
import { ErrorState, TodayColumns, TodaySection } from "@forgen-team/ui";
import { useActivity, useLearningSettings, useReviewSummary, useToday } from "@/src/entities/punisher";
import { AskBox, ResolutionChoices, useLaunchInstant } from "@/src/features/punisher/instant-start";
import { FirstSetup } from "@/src/features/punisher/first-setup";
import { describePunisherError, type InstantResponse, type Recommendation, type TodayResponse } from "@/src/shared/api/punisher";
import { formatDate, useLocale, useT } from "@/src/shared/i18n";
import { InlineError } from "@/src/shared/ui/punisher";
import { messages } from "./i18n";
import { recheckWithoutLecture, reviewBundleState, reviewTotal } from "./lib/reviewBundle";
import { lectureMinutes } from "./lib/recommendations";
import { BoxPad, BoxSkeleton, RecentInquiries, RecommendationBundle, ResumeBundle, ReviewBundle, StudyTopicsBundle } from "./ui";

const DATE: Intl.DateTimeFormatOptions = { month: "long", day: "numeric", weekday: "long" };

/** 강의가 없는 '다시 확인' 개념은 오늘의 추천 강의 행으로 더한다(이미 추천에 있으면 그대로). */
function withRecheck(recommendations: Recommendation[], today: TodayResponse | undefined, reason: string): Recommendation[] {
  const seen = new Set(recommendations.map((item) => item.conceptId));
  const extra = recheckWithoutLecture(today?.recheck)
    .filter((card) => !seen.has(card.conceptId))
    .map((card): Recommendation => ({
      kind: "goal_path", conceptId: card.conceptId, conceptName: card.conceptName, area: card.area, depth: 0, goalId: card.goalId,
      title: "", reason, practicalSituation: "", estimatedMinutes: 0,
    }));
  return [...extra, ...recommendations];
}

function BoxError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  return (
    <BoxPad>
      <ErrorState title={describePunisherError(error)} onRetry={onRetry} />
    </BoxPad>
  );
}

/**
 * 오늘(T-home) — 종류 상자 3개: 처리할 것(복습) · 공부할 것(이어보기 → 오늘의 추천) · 묻기·새 주제(입력 → 공부할 주제 → 최근 질문).
 * 폰 1열(처리할 것 → 공부할 것 → 묻기·새 주제), 본문 1024 이상 2열 3fr/2fr(왼쪽 처리할 것·공부할 것, 오른쪽 묻기·새 주제).
 * 제목·설정·배너·섹션 탭은 틀이 그린다. 셸 안·단독판 구성이 같다.
 */
export function TodayWidget() {
  const locale = useLocale();
  const t = useT(messages);
  const today = useToday();
  const summary = useReviewSummary();
  const activity = useActivity();
  const settings = useLearningSettings();
  const { launch, isPending, error } = useLaunchInstant();
  const [pickResult, setPickResult] = useState<InstantResponse | null>(null);

  async function start(conceptId: string, query?: string, studyRequestId?: string) {
    setPickResult(null);
    try {
      setPickResult(await launch({ conceptId, query, studyRequestId }));
    } catch {
      // 오류는 <InlineError error={error}> 가 표시한다.
    }
  }

  const data = today.data;
  const items = data?.items ?? [];
  const resume = items.filter((item) => item.kind === "resume");
  const recommendations = withRecheck(items.filter((item) => item.kind !== "resume"), data, t.recheckReason);
  const dailyMinutes = settings.data?.settings.dailyMinutes ?? activity.data?.dailyMinutes ?? 0;

  // 처리할 것 — 숫자는 /today, 오늘 세션은 /reviews/summary, 카드·쉬는 날은 /study/activity. 셋 다 받아야 상태가 정해진다.
  // 상자는 그대로 두고 안만 바꾼다(로딩 → 내용으로 바뀔 때 상자가 다시 그려지지 않게).
  let reviewContent: ReactNode;
  let reviewCount = 0;
  if (today.isLoading || summary.isLoading || activity.isLoading) {
    reviewContent = <BoxSkeleton rows={2} />;
  } else if (today.error || summary.error) {
    reviewContent = <BoxError error={today.error ?? summary.error} onRetry={() => { void today.refetch(); void summary.refetch(); }} />;
  } else {
    const state = reviewBundleState({
      today: data,
      session: summary.data?.today,
      cardsDue: activity.data?.today.cardsDue ?? 0,
      restToday: activity.data?.restToday ?? false,
    });
    reviewCount = state.kind === "due" || state.kind === "rest" ? reviewTotal(state.counts) : 0;
    reviewContent = <ReviewBundle state={state} />;
  }

  let studyContent: ReactNode;
  let studyCount = 0;
  if (today.isLoading) {
    studyContent = <BoxSkeleton rows={4} />;
  } else if (today.error) {
    studyContent = <BoxError error={today.error} onRetry={() => void today.refetch()} />;
  } else {
    studyCount = resume.length + recommendations.length;
    studyContent = (
      <>
        <ResumeBundle items={resume} />
        <RecommendationBundle
          items={recommendations}
          exhausted={Boolean(data?.exhausted)}
          nextChapter={data?.nextChapter}
          lectureMinutes={lectureMinutes(data?.session, recommendations)}
          dailyMinutes={dailyMinutes}
        />
      </>
    );
  }

  const askBox = (
    <TodaySection title={t.ask}>
      <div className="divide-y divide-line-subtle">
      <BoxPad className="pt-3">
        <AskBox />
        <div aria-live="polite" className="mt-2 flex flex-col gap-2 empty:hidden">
          <InlineError error={error} />
          {pickResult && <ResolutionChoices response={pickResult} disabled={isPending} onPick={(id) => void start(id)} />}
        </div>
      </BoxPad>
      <StudyTopicsBundle starting={isPending} onStart={(id, query, requestId) => void start(id, query, requestId)} />
      <RecentInquiries />
      </div>
    </TodaySection>
  );

  return (
    <div className="flex w-full flex-col gap-3">
      <p className="text-xs text-fg-muted">{formatDate(new Date(), locale, DATE)}</p>
      <TodayColumns
        todo={
          <>
            <TodaySection title={t.process} count={reviewCount}><div className="divide-y divide-line-subtle">{reviewContent}</div></TodaySection>
            <TodaySection title={t.study} count={studyCount}><div className="divide-y divide-line-subtle">{studyContent}</div></TodaySection>
          </>
        }
        study={askBox}
      />
      <FirstSetup />
    </div>
  );
}
