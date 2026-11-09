
import { Link } from "@/src/shared/lib/router";
import { useEffect, useMemo, useState } from "react";
import { useRouter } from "@/src/shared/lib/router";
import { useQueryClient } from "@tanstack/react-query";
import { Button, EmptyState, Spinner, buttonVariants } from "@/src/shared/ui/kit";
import { invalidateLearningState, parsePartialMarkup } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { LectureAssistPlaceholder, LectureBody, LectureHeader, LectureLayout, LectureTopBar, TocSheet, useActiveSection } from "../lecture-viewer";
import { useInstantStream, type StreamPhase } from "./useInstantStream";
import { useRetryInstant } from "./useRetryInstant";

const PHASE_LABELS: Record<StreamPhase, string> = {
  connecting: "생성 작업에 연결하는 중",
  generating: "강의를 실시간으로 작성하는 중",
  rewriting: "형식 문제로 다시 쓰는 중이에요",
  validating: "구조와 출처를 검증하는 중",
  done: "완료, 저장된 강의로 이동합니다",
  error: "생성 실패",
};

/** 즉석 강의 스트리밍 — 강의 뷰어와 같은 배치(상단 바·목차·읽기 열), 완료되면 저장된 강의로 교체. */
export function InstantStreamView({ jobId }: { jobId: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { phase, text, lectureId, errorMessage, rewriteReason } = useInstantStream(jobId);
  const retry = useRetryInstant(jobId);
  const content = useMemo(() => parsePartialMarkup(text), [text]);
  const [tocOpen, setTocOpen] = useState(false);

  useEffect(() => {
    if (phase !== "done" || !lectureId) { return; }
    void invalidateLearningState(queryClient);
    router.replace(lectureHref(lectureId));
  }, [phase, lectureId, router, queryClient]);

  const busy = phase === "connecting" || phase === "generating" || phase === "rewriting" || phase === "validating";
  const hasToc = content.sections.length > 0;
  const anchorKey = useMemo(() => content.sections.map((s) => s.anchor).filter(Boolean).join("|"), [content]);
  const activeAnchor = useActiveSection(anchorKey);

  return (
    <div className="flex w-full flex-col">
      <LectureTopBar title={content.title || "즉석 강의"} onOpenToc={hasToc ? () => setTocOpen(true) : undefined} />
      <LectureLayout
        sections={content.sections}
        totalMinutes={content.estimatedMinutes}
        activeAnchor={activeAnchor}
        assist={<LectureAssistPlaceholder />}
      >
        <div aria-busy={busy}>
          <p role="status" aria-live="polite" className="flex min-h-touch items-center gap-2 border-b border-line text-sm text-fg-secondary">
            {busy && <Spinner size="sm" label={null} />}
            즉석 강의 · {PHASE_LABELS[phase]}
          </p>
          {phase === "rewriting" && (
            <p className="mt-3 break-keep rounded-md bg-warning-subtle px-3 py-2 text-sm text-warning">
              처음 쓴 강의가 화면 형식 규칙을 지키지 않아 한 번 더 써요. 잠시만 기다려 주세요.
              {rewriteReason && <span className="mt-1 line-clamp-2 break-all text-xs text-fg-muted">사유: {rewriteReason}</span>}
            </p>
          )}
          {phase === "error" ? (
            <EmptyState
              title="강의를 생성하지 못했습니다"
              description={errorMessage}
              action={
                <div className="flex flex-wrap justify-center gap-2">
                  {retry.available && (
                    <Button size="sm" loading={retry.isPending} onClick={retry.run}>
                      다시 시도
                    </Button>
                  )}
                  <Link href="/" className={buttonVariants({ variant: "secondary", size: "sm" })}>
                    강의 화면으로 돌아가기
                  </Link>
                </div>
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
        </div>
      </LectureLayout>
      {hasToc && (
        <TocSheet open={tocOpen} onClose={() => setTocOpen(false)} sections={content.sections} totalMinutes={content.estimatedMinutes} />
      )}
    </div>
  );
}
