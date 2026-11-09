
import { Link } from "@/src/shared/lib/router";
import { useState } from "react";
import { Target } from "lucide-react";
import { EmptyState, ErrorState, List, ListSkeleton, Section, buttonVariants } from "@/src/shared/ui/kit";
import { groupByParent, useToday } from "@/src/entities/punisher";
import { InstantQueryBox, ResolutionChoices, useLaunchInstant } from "@/src/features/punisher/instant-start";
import { DismissedNotice, useResumeDismiss } from "@/src/features/punisher/resume-dismiss";
import { AddStudyRequestForm } from "@/src/features/punisher/study-request-add";
import { describePunisherError, type InstantResponse, type Recommendation } from "@/src/shared/api/punisher";
import { DeeperGroup, InlineError } from "@/src/shared/ui/punisher";
import { RecommendationRow, RecommendationSheet, ResumeRow, StudyRequestList } from "./ui";

function resumeKey(item: Recommendation): string {
  return item.lectureId ?? item.conceptId;
}

/** 강의(오늘): 궁금한 것 입력 한 줄 → 이어보기 → 추천 목록 → 학습 요청 (design §3.5). */
export function TodayWidget() {
  const today = useToday();
  const { launch, isPending, error } = useLaunchInstant();
  const [pickResult, setPickResult] = useState<InstantResponse | null>(null);
  const [detail, setDetail] = useState<Recommendation | null>(null);
  const resumeDismiss = useResumeDismiss();

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

  function renderResume(item: Recommendation) {
    return (
      <ResumeRow
        key={resumeKey(item)}
        item={item}
        dismissing={resumeDismiss.pending}
        onDismiss={(it) => resumeDismiss.dismiss({ lectureId: it.lectureId ?? "", label: it.conceptName || it.title })}
      />
    );
  }

  const items = today.data?.items ?? [];
  const resume = items.filter((item) => item.kind === "resume");
  const recommendations = items.filter((item) => item.kind !== "resume");
  const ready = !today.isLoading && !today.error;
  const showResume = ready && (resume.length > 0 || resumeDismiss.dismissed || resumeDismiss.error);

  const resumeSection = showResume ? (
    <Section variant="card" title="이어보기">
      <List>
        {groupByParent(resume, resumeKey, (it) => it.parentLectureId).map(({ item, children }) => (
          <DeeperGroup key={resumeKey(item)} parent={renderResume(item)}>
            {children.map((child) => renderResume(child))}
          </DeeperGroup>
        ))}
      </List>
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
          description={today.data.message ?? "새 목표를 선택하면 다음 경로를 추천합니다."}
          action={
            <Link href="/map?panel=goals" className={buttonVariants({ variant: "secondary", size: "sm" })}>
              목표 선택
            </Link>
          }
        />
      ) : (
        <EmptyState compact title="오늘 추천이 없습니다" />
      )}
    </Section>
  );

  // 한 단: 입력 → 이어보기 → 오늘의 추천 → 학습 요청. 2단(≥600px, 폴드 펼침): 왼쪽 입력·오늘의 추천 / 오른쪽 이어보기·학습 요청.
  // 열 묶음은 한 단에서 display:contents 라 order 로 위 순서를 지킨다(접기/펴기 때 같은 DOM 을 재배치만 → 상태 유지).
  return (
    <div className="flex w-full flex-col gap-6 fold:grid fold:grid-cols-2 fold:items-start fold:gap-4 md:gap-6">
      <div className="contents fold:flex fold:min-w-0 fold:flex-col fold:gap-4 md:gap-6">
        <div className="order-1 fold:order-none">
          <InstantQueryBox />
        </div>
        <div aria-live="polite" className="order-2 empty:hidden fold:order-none">
          <InlineError error={error} />
          {pickResult && <ResolutionChoices response={pickResult} disabled={isPending} onPick={(id) => start(id)} />}
        </div>
        <div className="order-4 fold:order-none">{recommendSection}</div>
      </div>
      <div className="contents fold:flex fold:min-w-0 fold:flex-col fold:gap-4 md:gap-6">
        {resumeSection && <div className="order-3 fold:order-none">{resumeSection}</div>}
        <div className="order-5 fold:order-none">
          <Section variant="card" title="학습 요청">
            <AddStudyRequestForm />
            <StudyRequestList starting={isPending} onStart={(id, query) => start(id, query)} />
          </Section>
        </div>
      </div>

      <RecommendationSheet item={detail} starting={isPending} onClose={() => setDetail(null)} onStart={(id) => start(id)} />
    </div>
  );
}
