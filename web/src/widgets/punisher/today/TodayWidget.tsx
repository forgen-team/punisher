
import { Link } from "@/src/shared/lib/router";
import { useState } from "react";
import { Target } from "lucide-react";
import { EmptyState, ErrorState, List, ListSkeleton, Section, buttonVariants } from "@/src/shared/ui/kit";
import { useToday } from "@/src/entities/punisher";
import { InstantQueryBox, ResolutionChoices, useLaunchInstant } from "@/src/features/punisher/instant-start";
import { DismissedNotice, useResumeDismiss } from "@/src/features/punisher/resume-dismiss";
import { describePunisherError, type InstantResponse, type Recommendation } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { RecommendationRow, RecommendationSheet, ResumeRow, StudyRequestList } from "./ui";

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

  const items = today.data?.items ?? [];
  const resume = items.filter((item) => item.kind === "resume");
  const recommendations = items.filter((item) => item.kind !== "resume");

  return (
    <div className="flex w-full flex-col gap-6">
      <InstantQueryBox />

      <div aria-live="polite" className="empty:hidden">
        <InlineError error={error} />
        {pickResult && <ResolutionChoices response={pickResult} disabled={isPending} onPick={(id) => start(id)} />}
      </div>

      {today.isLoading ? (
        // 실제 화면(섹션 제목 + 카드 안 추천 5줄)과 같은 높이로 자리를 잡아 아래 '학습 요청'이 밀리지 않게 한다.
        <Section variant="card" title="오늘의 추천" description="하루 30분, 기반 지식을 한 단계씩" aria-busy="true">
          <ListSkeleton rows={5} />
        </Section>
      ) : today.error ? (
        <ErrorState message={describePunisherError(today.error)} onRetry={() => today.refetch()} retrying={today.isRefetching} />
      ) : (
        <>
          {(resume.length > 0 || resumeDismiss.dismissed || resumeDismiss.error) && (
            <Section variant="card" title="이어보기">
              <List>
                {resume.map((item) => (
                  <ResumeRow
                    key={`resume-${item.lectureId ?? item.conceptId}`}
                    item={item}
                    dismissing={resumeDismiss.pending}
                    onDismiss={(it) => resumeDismiss.dismiss({ lectureId: it.lectureId ?? "", label: it.conceptName || it.title })}
                  />
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
          )}
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
        </>
      )}

      <Section variant="card" title="학습 요청">
        <StudyRequestList starting={isPending} onStart={(id, query) => start(id, query)} />
      </Section>

      <RecommendationSheet item={detail} starting={isPending} onClose={() => setDetail(null)} onStart={(id) => start(id)} />
    </div>
  );
}
