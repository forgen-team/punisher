
import { Link } from "@/src/shared/lib/router";
import { useEffect, useMemo, useState } from "react";
import { useRouter } from "@/src/shared/lib/router";
import { useQueryClient } from "@tanstack/react-query";
import { EmptyState, Spinner, buttonVariants } from "@/src/shared/ui/kit";
import { invalidateLearningState, parsePartialMarkup } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { LectureBody, LectureHeader, LectureTopBar, SectionToc, TocSheet } from "../lecture-viewer";
import { useInstantStream, type StreamPhase } from "./useInstantStream";

const PHASE_LABELS: Record<StreamPhase, string> = {
  connecting: "생성 작업에 연결하는 중",
  generating: "강의를 실시간으로 작성하는 중",
  validating: "구조와 출처를 검증하는 중",
  done: "완료, 저장된 강의로 이동합니다",
  error: "생성 실패",
};

/** 즉석 강의 스트리밍 — 강의 뷰어와 같은 배치(상단 바·목차·읽기 열), 완료되면 저장된 강의로 교체. */
export function InstantStreamView({ jobId }: { jobId: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { phase, text, lectureId, errorMessage } = useInstantStream(jobId);
  const content = useMemo(() => parsePartialMarkup(text), [text]);
  const [tocOpen, setTocOpen] = useState(false);

  useEffect(() => {
    if (phase !== "done" || !lectureId) { return; }
    void invalidateLearningState(queryClient);
    router.replace(lectureHref(lectureId));
  }, [phase, lectureId, router, queryClient]);

  const busy = phase === "connecting" || phase === "generating" || phase === "validating";
  const hasToc = content.sections.length > 0;

  return (
    <div className="flex w-full flex-col">
      <LectureTopBar title={content.title || "즉석 강의"} onOpenToc={hasToc ? () => setTocOpen(true) : undefined} />
      <div className="flex gap-8">
        <aside className="sticky top-[calc(var(--app-header-offset)+4rem)] hidden max-h-[calc(100dvh-6rem)] w-56 shrink-0 self-start overflow-y-auto pb-10 pt-4 lg:block">
          {hasToc && <SectionToc sections={content.sections} totalMinutes={content.estimatedMinutes} />}
        </aside>
        <article className="mx-auto w-full min-w-0 max-w-2xl pb-16" aria-busy={busy}>
          <p role="status" aria-live="polite" className="flex min-h-touch items-center gap-2 border-b border-line text-sm text-fg-secondary">
            {busy && <Spinner size="sm" label={null} />}
            즉석 강의 · {PHASE_LABELS[phase]}
          </p>
          {phase === "error" ? (
            <EmptyState
              title="강의를 생성하지 못했습니다"
              description={errorMessage}
              action={
                <Link href="/" className={buttonVariants({ variant: "secondary", size: "sm" })}>
                  강의 화면으로 돌아가기
                </Link>
              }
            />
          ) : (
            <>
              <LectureHeader content={content} fallbackTitle="강의 준비 중…" />
              <div className="mt-6">
                <LectureBody content={content} />
              </div>
            </>
          )}
        </article>
      </div>
      {hasToc && (
        <TocSheet open={tocOpen} onClose={() => setTocOpen(false)} sections={content.sections} totalMinutes={content.estimatedMinutes} />
      )}
    </div>
  );
}
