
import { useEffect, useState } from "react";
import { useRouter } from "@/src/shared/lib/router";
import { useQueryClient } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { Button } from "@forgen-team/ui";
import { useScreenTitle } from "@/src/shared/lib/screen-chrome";
import { invalidateLearningState } from "@/src/entities/punisher";
import { punisherApi } from "@/src/shared/api/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { PretestPanel, usePretest } from "@/src/features/punisher/pretest";
import { failureLines, stageLabel, type StageState } from "./lib/stageState";
import { StageSteps } from "./ui/StageSteps";
import { useInstantStream } from "./useInstantStream";
import { useRetryInstant } from "./useRetryInstant";
import { useLocale, translate } from "@/src/shared/i18n";
import { messages } from "./i18n";

/** 작업이 시작된 뒤 지난 시간. 시작 시각을 못 받으면 화면을 연 때를 기준으로 한다. */
function useElapsedLabel(jobId: string, running: boolean): string {
  const t = translate(messages, useLocale());
  const [seconds, setSeconds] = useState(0);
  const [startedAt, setStartedAt] = useState<number | null>(null);
  useEffect(() => {
    let live = true;
    punisherApi.getJob(jobId).then(
      (job) => { if (live && job.startedAt) { setStartedAt(Date.parse(job.startedAt)); } },
      (error: unknown) => console.warn("punisher: job start time unavailable", error),
    );
    return () => { live = false; };
  }, [jobId]);
  useEffect(() => {
    if (!running) { return; }
    const from = startedAt ?? Date.now();
    const tick = () => setSeconds(Math.max(0, Math.floor((Date.now() - from) / 1000)));
    tick();
    const timer = window.setInterval(tick, 1000);
    return () => window.clearInterval(timer);
  }, [running, startedAt]);
  if (!running || seconds < 5) { return ""; }
  return seconds < 60 ? t.elapsedSeconds(seconds) : t.elapsedMinutes(Math.floor(seconds / 60), seconds % 60);
}

function FailurePanel({ state, jobId }: { state: StageState; jobId: string }) {
  const t = translate(messages, useLocale());
  const retry = useRetryInstant(jobId);
  const lines = failureLines(state);
  return (
    <div role="alert" className="flex flex-col gap-2 rounded-md bg-danger-subtle px-3 py-3">
      <p className="flex items-center gap-1.5 text-sm font-semibold text-danger">
        <AlertTriangle className="size-4 shrink-0" aria-hidden="true" />
        {t.failedTitle}
      </p>
      {lines.length > 0 && (
        <ul className="flex flex-col gap-0.5 text-sm text-fg">
          {lines.map((line, index) => (
            <li key={index} className="line-clamp-1 break-all">{line}</li>
          ))}
        </ul>
      )}
      {retry.available && (
        <Button variant="primary" size="sm" className="self-start" loading={retry.isPending} onClick={retry.run}>
          {t.retry}
        </Button>
      )}
    </div>
  );
}

/** 즉석 강의 생성 — 초안 본문 대신 단계(작성 → 검수 → 고침)만 보이고, 기다리는 동안 예측 질문을 둔다. 발행되면 강의로 이동. */
export function InstantStreamView({ jobId }: { jobId: string }) {
  const router = useRouter();
  const locale = useLocale();
  const t = translate(messages, locale);
  const queryClient = useQueryClient();
  const state = useInstantStream(jobId);
  const pretest = usePretest({ jobId });
  const { stage, lectureId } = state;
  useScreenTitle(t.title);

  useEffect(() => {
    if (stage !== "published" || !lectureId) { return; }
    void invalidateLearningState(queryClient);
    router.replace(lectureHref(lectureId));
  }, [stage, lectureId, router, queryClient]);

  const busy = stage !== "published" && stage !== "failed";
  const elapsed = useElapsedLabel(jobId, busy);

  return (
    <div aria-busy={busy} className="flex w-full flex-col gap-4 pt-4">
      {stage === "failed" ? (
        <FailurePanel state={state} jobId={jobId} />
      ) : (
        <div className="flex flex-col gap-3">
          <p className="text-xl font-semibold text-fg">{t.headline}</p>
          {/* 단계가 바뀔 때만 읽히도록, 매초 바뀌는 경과 시간은 알림 영역 밖에 둔다. */}
          <div role="status" aria-live="polite">
            <StageSteps state={state} />
            {stage === "published" && <span className="sr-only">{stageLabel(state, locale)}</span>}
          </div>
          {elapsed && <span className="text-xs text-fg-muted tabular-nums">{elapsed}</span>}
        </div>
      )}
      <PretestPanel detail={pretest.data} />
    </div>
  );
}
