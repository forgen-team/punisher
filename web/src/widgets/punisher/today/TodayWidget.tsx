
import { Link, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { canGoBack } from "@/src/shared/lib/in-app-history";
import { useState } from "react";
import { Target } from "lucide-react";
import { EmptyState, ErrorState, HeightReserve, List, ListSkeleton, Section, buttonVariants, cn } from "@/src/shared/ui/kit";
import { useInquiries, useLearningSettings, useReviewSummary, useToday } from "@/src/entities/punisher";
import { InstantQueryBox, ResolutionChoices, useLaunchInstant } from "@/src/features/punisher/instant-start";
import { DismissedNotice, ResumeList, useResumeDismiss } from "@/src/features/punisher/resume-dismiss";
import { describePunisherError, type InstantResponse, type Recommendation } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { InlineError } from "@/src/shared/ui/punisher";
import { AddStudyRequestForm } from "@/src/features/punisher/study-request-add";
import { FirstSetup } from "@/src/features/punisher/first-setup";
import { messages } from "./i18n";
import { todayRows, type TodayRow } from "./lib/recheck";
import { reviewCardState } from "./lib/reviewCard";
import { sessionTimeLabel } from "./lib/sessionTime";
import { ExplanationRequestList, InquiryList, NEW_CHAPTER_PATH, NextChapterCard, RECOMMENDATION_PARAM, RecheckRow, RecommendationRow, RecommendationSheet, ReviewCard, StudyRequestList, StudyToday, SynthesisRow, UncheckedRow } from "./ui";
import { isWitems } from "@/src/shared/config/runtime";

/** 추천 목록의 한 줄 — 확인하기 · 다시 확인 · 추천(상세 시트 주소로 가는 링크). */
function renderRow(row: TodayRow) {
  switch (row.type) {
    case "synthesis":
      return <SynthesisRow key="synthesis" />;
    case "unchecked":
      return <UncheckedRow key={`unchecked-${row.card.conceptId}`} card={row.card} />;
    case "recheck":
      return <RecheckRow key={`recheck-${row.card.conceptId}`} card={row.card} />;
    default:
      return <RecommendationRow key={`${row.item.kind}-${row.item.conceptId}`} item={row.item} />;
  }
}

/** 강의(오늘): 복습 카드 → 궁금한 것 입력 한 줄 → 이어보기 → 추천 목록 → 학습 요청(단독판: 직접 입력. witems 는 이 자리가 RSS '학습 요청' — docs/SYNC.md). */
export function TodayWidget() {
  const locale = useLocale();
  const t = useT(messages);
  const today = useToday();
  const { launch, isPending, error } = useLaunchInstant();
  const [pickResult, setPickResult] = useState<InstantResponse | null>(null);
  // 추천 상세 시트는 주소(?rec=<개념>)에 남는다 — 뒤로 가면 닫히고, 새 탭으로 열어도 같은 시트.
  const router = useRouter();
  const params = useSearchParams();
  const detailId = params.get(RECOMMENDATION_PARAM);
  // 닫기: 행에서 열었으면(앱 안 기록이 있으면) 뒤로 — 기록에 같은 화면이 두 번 쌓이지 않게. 직접 들어왔으면 시트 없는 주소로 교체.
  const closeDetail = () => {
    if (!detailId) { return; }
    if (canGoBack()) { router.back(); } else { router.replace("/", { guard: false }); }
  };
  const settings = useLearningSettings();
  const resumeDismiss = useResumeDismiss();
  const [resumeExpanded, setResumeExpanded] = useState(false);
  const inquiries = useInquiries({ open: true });
  const reviewToday = today.data?.review;
  const reviewSummary = useReviewSummary(Boolean(reviewToday));

  async function start(conceptId: string, query?: string) {
    setPickResult(null);
    try {
      const result = await launch({ conceptId, query });
      setPickResult(result);
      if (result) { closeDetail(); }
    } catch {
      // 오류는 <InlineError error={error}> 가 표시한다.
      closeDetail();
    }
  }

  const items = today.data?.items ?? [];
  const resume = items.filter((item) => item.kind === "resume");
  const recommendations = items.filter((item) => item.kind !== "resume");
  const rows = todayRows(recommendations, today.data?.recheck, today.data?.unchecked, Boolean(today.data?.synthesis?.due));
  const detail: Recommendation | null = (detailId && recommendations.find((item) => item.conceptId === detailId)) || null;
  const ready = !today.isLoading && !today.error;
  // 복습 카드는 열들 위 전체 폭 띠(맨 앞) — 3단에서도 열 배치를 바꾸지 않고, '추천 소진' 안내와 다른 칸에 둔다.
  // 완료 여부(오늘 세션)를 받기 전에는 그리지 않는다(숨김 ↔ 접힌 완료가 깜빡이지 않게).
  const reviewState = reviewCardState(reviewToday, reviewSummary.data?.today);
  // 세션 시간 합계(서버 계산 — 강의는 다음에 볼 강의, 이어보기가 있으면 그 남은 시간)는 '오늘의 추천' 밖 한 줄에만 보이고
  // 복습 카드의 합계 줄은 뺀다. 추천 목록의 설명 자리에 두면 추천 항목들의 합으로 읽혀 헷갈린다.
  const sessionTime = sessionTimeLabel(today.data?.session, locale);
  const reviewReady = ready && (!reviewToday || !reviewSummary.isLoading);
  const showResume = ready && (resume.length > 0 || resumeDismiss.dismissed || resumeDismiss.error);

  const resumeSection = showResume ? (
    <Section variant="card" title={t.resume}>
      <ResumeList items={resume} columns dismissing={resumeDismiss.pending} onDismiss={resumeDismiss.dismiss} expanded={resumeExpanded} onExpandedChange={setResumeExpanded} />
      <div aria-live="polite">
        <DismissedNotice
          dismissed={resumeDismiss.dismissed}
          error={resumeDismiss.error}
          pending={resumeDismiss.pending}
          onUndo={resumeDismiss.undo}
          onClose={resumeDismiss.clear}
        />
      </div>
    </Section>
  ) : null;

  const recommendSection = today.isLoading ? (
    // 실제 화면(섹션 제목 + 카드 안 추천 5줄)과 같은 높이로 자리를 잡아 아래 '학습 요청'이 밀리지 않게 한다.
    <Section variant="card" title={t.recommendations} aria-busy="true">
      <ListSkeleton rows={5} />
    </Section>
  ) : today.error ? (
    <ErrorState message={describePunisherError(today.error)} onRetry={() => today.refetch()} retrying={today.isRefetching} />
  ) : (
    <Section variant="card" title={t.recommendations}>
      {rows.length > 0 ? (
        <List>
          {rows.map((row) => renderRow(row))}
        </List>
      ) : today.data?.exhausted ? (
        <EmptyState
          compact
          icon={<Target />}
          title={t.exhaustedTitle}
          description={t.exhaustedDescription}
          action={
            <Link href={NEW_CHAPTER_PATH} className={buttonVariants({ variant: "secondary", size: "sm" })}>
              {t.pickNextChapter}
            </Link>
          }
        />
      ) : (
        <EmptyState compact title={t.noRecommendations} />
      )}
    </Section>
  );

  // 입력·선택 결과는 열들 위에 한 줄(전체 폭)로 둔다 — 2단에서 왼쪽 열만 입력 상자만큼 내려가
  // '오늘의 추천'과 오른쪽 '이어보기/학습 요청' 제목 높이가 어긋나던 문제(2026-11-09).
  // 이어보기는 몇 개든 열들 위 전체 폭 띠로 둔다(폭이 넓으면 항목을 2~3열로 펼쳐 세로로만 길어지지 않게).
  // 한 단: 입력 → 이어보기 → 질문 해설 → (다음 챕터) → 오늘의 추천 → 학습 요청.
  // 2단(fold): 왼쪽 (다음 챕터)·오늘의 추천 / 오른쪽 질문 해설·학습 요청.
  // 3단(xl): 오늘의 추천 | 질문 해설·다음 챕터 | 학습 요청.
  // 열 묶음은 좁은 폭에서 display:contents 라 order 로 위 순서를 지킨다(폭이 바뀌어도 같은 DOM 을 재배치만 → 상태 유지).
  // 다음 챕터 카드는 2단·3단에서 놓이는 열이 달라 두 자리에 두고 폭에 따라 하나만 보인다(상태 없는 카드).
  const nextChapter = today.data?.nextChapter;
  const showInquiries = Boolean(inquiries.error) || (inquiries.data?.length ?? 0) > 0;
  const middle = Boolean(nextChapter || showInquiries);
  return (
    <div className="flex w-full flex-col gap-6">
      <HeightReserve id="learn-review" ready={reviewReady} skeleton={null} className="[&:empty:not([style])]:hidden">
        {reviewState.kind !== "hidden" ? <ReviewCard state={reviewState} showTotal={!sessionTime} /> : null}
      </HeightReserve>
      {/* 오늘 목표·지난 12주·복습 예측·자꾸 막히는 목표(W5a) */}
      <StudyToday />
      <InstantQueryBox />
      <div aria-live="polite" className="empty:hidden">
        <InlineError error={error} />
        {pickResult && <ResolutionChoices response={pickResult} disabled={isPending} onPick={(id) => start(id)} />}
      </div>
      <HeightReserve id="learn-resume" ready={ready} skeleton={null} remember={!resumeExpanded && !resumeDismiss.dismissed} className="[&:empty:not([style])]:hidden">{resumeSection}</HeightReserve>
      {sessionTime && (
        // 합계가 무엇의 합인지 라벨로 보이고, 하루 학습량 설정으로 이어 둔다(P1).
        <p className="-my-3 flex flex-wrap items-center gap-x-2 text-sm tabular-nums text-fg-secondary">
          <span>{sessionTime}</span>
          {settings.data?.settings.dailyMinutes ? (
            <Link href="/settings" title={t.dailyGoalLink} className="text-fg-muted underline-offset-2 hover:text-fg hover:underline">
              {t.dailyGoal(settings.data.settings.dailyMinutes)}
            </Link>
          ) : null}
        </p>
      )}
      <div className={cn("flex flex-col gap-6 fold:grid fold:grid-cols-2 fold:items-start fold:gap-4 md:gap-6", middle && "xl:grid-cols-3")}>
        <div className="contents fold:flex fold:min-w-0 fold:flex-col fold:gap-4 md:gap-6">
          {nextChapter && (
            <div className="order-2 fold:order-none xl:hidden">
              <NextChapterCard prompt={nextChapter} />
            </div>
          )}
          <div className="order-3 fold:order-none">{recommendSection}</div>
        </div>
        <div className="contents fold:flex fold:min-w-0 fold:flex-col fold:gap-4 md:gap-6 xl:contents">
          {middle && (
            <div className="contents xl:flex xl:min-w-0 xl:flex-col xl:gap-6">
              {showInquiries && (
                <div className="order-1 fold:order-none">
                  <InquiryList inquiries={inquiries.data} error={inquiries.error} onRetry={() => void inquiries.refetch()} />
                </div>
              )}
              {nextChapter && (
                <div className="hidden xl:block">
                  <NextChapterCard prompt={nextChapter} />
                </div>
              )}
            </div>
          )}
          <div className="order-4 min-w-0 fold:order-none">
            {isWitems() ? (
              // Witems 안: RSS '이 글 깊게 읽기'로 만든 글 해설(학습 요청 자리를 대신 — witems 와 같은 구성).
              <Section variant="card" title={t.explanations}>
                <ExplanationRequestList />
              </Section>
            ) : (
              <Section variant="card" title={t.studyRequests}>
                <AddStudyRequestForm />
                <StudyRequestList starting={isPending} onStart={(id, query) => start(id, query)} />
              </Section>
            )}
          </div>
        </div>
      </div>

      <RecommendationSheet item={detail} starting={isPending} onClose={closeDetail} onStart={(id) => start(id)} />
      <FirstSetup />
    </div>
  );
}
