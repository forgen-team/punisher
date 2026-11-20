import { useEffect, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { X } from "lucide-react";
import { Link } from "@/src/shared/lib/router";
import { IconButton, Spinner } from "@/src/shared/ui/kit";
import { invalidateLearningState, useInquiry } from "@/src/entities/punisher";
import { ResolutionChoices, inquiryHref, lectureHref, useLaunchInstant } from "@/src/features/punisher/instant-start";
import { useReadingThreads } from "@/src/features/punisher/reading-assist";
import { InlineError } from "@/src/shared/ui/punisher";
import { punisherKeys, type InstantResponse } from "@/src/shared/api/punisher";
// instant-stream 위젯 index 는 lecture-viewer 를 가져오므로(순환) 훅·단계 표시만 파일로 가져온다.
import { useInstantStream } from "../../instant-stream/useInstantStream";
import { StageSteps } from "../../instant-stream/ui/StageSteps";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

function Shell({ children, busy, onClose }: { children: ReactNode; busy: boolean; onClose: () => void }) {
  const t = useT(messages);
  return (
    <section aria-label={t.deepStatusLabel} aria-busy={busy} className="mb-3 flex min-h-touch flex-wrap items-center gap-3 border-y border-line py-1 text-sm">
      <div role="status" aria-live="polite" className="flex min-w-0 flex-1 flex-wrap items-center gap-3">
        {busy && <Spinner size="sm" label={null} />}
        {children}
      </div>
      <IconButton aria-label={t.deepClose} size="sm" onClick={onClose}>
        <X aria-hidden="true" />
      </IconButton>
    </section>
  );
}

function Ready({ lectureId }: { lectureId: string }) {
  const t = useT(messages);
  return (
    <>
      <span className="font-medium text-success">{t.deepReady}</span>
      <Link href={lectureHref(lectureId)} className="inline-flex min-h-touch items-center text-accent-text underline underline-offset-2">
        {t.deepOpen}
      </Link>
    </>
  );
}

/** 생성 중: 작성 ✓ → 검수 중 → 고침(즉석 강의 화면과 같은 단계 표시). 끝나면 '이어진 강의' 목록을 다시 읽는다. */
function JobStatus({ jobId, lectureId, onClose }: { jobId: string; lectureId: string; onClose: () => void }) {
  const t = useT(messages);
  const queryClient = useQueryClient();
  const state = useInstantStream(jobId);
  const done = state.stage === "published" && state.lectureId;
  useEffect(() => {
    if (!done) { return; }
    void queryClient.invalidateQueries({ queryKey: punisherKeys.lectureLinks(lectureId) });
    void invalidateLearningState(queryClient);
  }, [done, queryClient, lectureId]);
  const busy = state.stage !== "published" && state.stage !== "failed";
  return (
    <Shell busy={busy} onClose={onClose}>
      {state.stage === "failed" ? (
        <span className="text-danger">{t.deepFailed}</span>
      ) : done && state.lectureId ? (
        <Ready lectureId={state.lectureId} />
      ) : (
        <StageSteps state={state} />
      )}
    </Shell>
  );
}

/** 읽던 강의와 같은 개념이라 질문 해설로 대신 만든 경우: 쓰는 중 → 준비됨·열기. */
function InquiryStatus({ inquiryId, onClose }: { inquiryId: string; onClose: () => void }) {
  const t = useT(messages);
  const inquiry = useInquiry(inquiryId);
  const status = inquiry.data?.status;
  const busy = !status || status === "pending";
  return (
    <Shell busy={busy} onClose={onClose}>
      {status === "failed" ? (
        <span className="text-danger">{t.deepFailed}</span>
      ) : busy ? (
        <span>{t.deepInquiryPending}</span>
      ) : (
        <span className="font-medium text-success">{t.deepInquiryReady}</span>
      )}
      <Link href={inquiryHref(inquiryId)} className="inline-flex min-h-touch items-center text-accent-text underline underline-offset-2">
        {t.deepOpen}
      </Link>
    </Shell>
  );
}

/** 개념을 하나로 못 정했거나 이미 배운 개념이면 즉석 강의와 같은 선택 화면 — 고르면 원 강의 아래로 이어 붙여 시작한다. */
function Choices({ response, lectureId, parentAnchor, onClose }: { response: InstantResponse; lectureId: string; parentAnchor: string; onClose: () => void }) {
  const { launch, isPending, error } = useLaunchInstant();
  async function pick(conceptId: string) {
    try {
      await launch({ conceptId, parentLectureId: lectureId, parentAnchor });
    } catch {
      // 선택지를 유지하고 오류만 표시한다.
      return;
    }
  }
  return (
    <Shell busy={false} onClose={onClose}>
      <div className="flex w-full flex-col gap-2 py-1">
        <ResolutionChoices response={response} disabled={isPending} onPick={pick} />
        <InlineError error={error} />
      </div>
    </Shell>
  );
}

/** '이 대화로 심화 강의 만들기' 뒤 강의 화면 위 상태 줄. */
export function DeepLectureStatus({ lectureId }: { lectureId: string }) {
  const t = useT(messages);
  const ctx = useReadingThreads();
  const deep = ctx?.deep;
  const result = deep?.result;
  if (!ctx || !deep || !result) { return null; }
  if (result.status === "needs_selection" || result.status === "already_learned") {
    const root = ctx.threads.find((th) => th.root.id === deep.rootId)?.root;
    return <Choices response={result} lectureId={lectureId} parentAnchor={root?.anchor ?? ""} onClose={deep.clear} />;
  }
  if (result.status === "inquiry" && result.inquiryId) {
    return <InquiryStatus key={result.inquiryId} inquiryId={result.inquiryId} onClose={deep.clear} />;
  }
  if (result.jobId && (result.status === "started" || result.status === "attached")) {
    return <JobStatus key={result.jobId} jobId={result.jobId} lectureId={lectureId} onClose={deep.clear} />;
  }
  if (result.lectureId) {
    return (
      <Shell busy={false} onClose={deep.clear}>
        <Ready lectureId={result.lectureId} />
      </Shell>
    );
  }
  return (
    <Shell busy={false} onClose={deep.clear}>
      <span className="text-danger">{t.deepNotStarted}</span>
    </Shell>
  );
}
