
import { Link } from "@/src/shared/lib/router";
import { useState } from "react";
import { Target } from "lucide-react";
import { EmptyState, ErrorState, HeightReserve, List, ListSkeleton, Section, buttonVariants, cn } from "@/src/shared/ui/kit";
import { useInquiries, useToday } from "@/src/entities/punisher";
import { InstantQueryBox, ResolutionChoices, useLaunchInstant } from "@/src/features/punisher/instant-start";
import { DismissedNotice, ResumeList, useResumeDismiss } from "@/src/features/punisher/resume-dismiss";
import { describePunisherError, type InstantResponse, type Recommendation } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { AddStudyRequestForm } from "@/src/features/punisher/study-request-add";
import { InquiryList, NEW_CHAPTER_PATH, NextChapterCard, RecommendationRow, RecommendationSheet, StudyRequestList } from "./ui";

/** 강의(오늘): 궁금한 것 입력 한 줄 → 이어보기 → 추천 목록 → 학습 요청(단독판: 직접 입력. witems 는 이 자리가 RSS '학습 요청' — docs/SYNC.md). */
export function TodayWidget() {
  const today = useToday();
  const { launch, isPending, error } = useLaunchInstant();
  const [pickResult, setPickResult] = useState<InstantResponse | null>(null);
  const [detail, setDetail] = useState<Recommendation | null>(null);
  const resumeDismiss = useResumeDismiss();
  const [resumeExpanded, setResumeExpanded] = useState(false);
  const inquiries = useInquiries({ open: true });

  async function start(conceptId: string, query?: string) {
    setPickResult(null);
    try {
      const result = await launch({ conceptId, query });
      setPickResult(result);
      if (result) { setDetail(null); }
    } catch {
      // 오류는 <InlineError error={error}> 가 표시한다.
      setDetail(null);
    }
  }

  const items = today.data?.items ?? [];
  const resume = items.filter((item) => item.kind === "resume");
  const recommendations = items.filter((item) => item.kind !== "resume");
  const ready = !today.isLoading && !today.error;
  const showResume = ready && (resume.length > 0 || resumeDismiss.dismissed || resumeDismiss.error);

  const resumeSection = showResume ? (
    <Section variant="card" title="이어보기">
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
    <Section variant="card" title="오늘의 추천" description="하루 30분, 기반 지식을 한 단계씩" aria-busy="true">
      <ListSkeleton rows={5} />
    </Section>
  ) : today.error ? (
    <ErrorState message={describePunisherError(today.error)} onRetry={() => today.refetch()} retrying={today.isRefetching} />
  ) : (
    <Section variant="card" title="오늘의 추천" description="하루 30분, 기반 지식을 한 단계씩">
      {recommendations.length > 0 ? (
        <List>
          {recommendations.map((item) => (
            <RecommendationRow key={`${item.kind}-${item.conceptId}`} item={item} onOpen={setDetail} />
          ))}
        </List>
      ) : today.data?.exhausted ? (
        <EmptyState
          compact
          icon={<Target />}
          title="추천할 다음 강의가 없습니다"
          description="지금 지도에서 배울 강의를 다 마쳤어요. 다음 챕터를 골라 새 개념을 열어 보세요."
          action={
            <Link href={NEW_CHAPTER_PATH} className={buttonVariants({ variant: "secondary", size: "sm" })}>
              다음 챕터 고르기
            </Link>
          }
        />
      ) : (
        <EmptyState compact title="오늘 추천이 없습니다" />
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
      <InstantQueryBox />
      <div aria-live="polite" className="empty:hidden">
        <InlineError error={error} />
        {pickResult && <ResolutionChoices response={pickResult} disabled={isPending} onPick={(id) => start(id)} />}
      </div>
      <HeightReserve id="learn-resume" ready={ready} skeleton={null} remember={!resumeExpanded && !resumeDismiss.dismissed} className="[&:empty:not([style])]:hidden">{resumeSection}</HeightReserve>
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
            <Section variant="card" title="학습 요청">
              <AddStudyRequestForm />
              <StudyRequestList starting={isPending} onStart={(id, query) => start(id, query)} />
            </Section>
          </div>
        </div>
      </div>

      <RecommendationSheet item={detail} starting={isPending} onClose={() => setDetail(null)} onStart={(id) => start(id)} />
    </div>
  );
}
