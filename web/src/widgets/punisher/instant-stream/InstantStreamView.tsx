
import { useEffect, useState } from "react";
import { useRouter } from "@/src/shared/lib/router";
import { useQueryClient } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { Button, Spinner, cn } from "@/src/shared/ui/kit";
import { invalidateLearningState } from "@/src/entities/punisher";
import { punisherApi } from "@/src/shared/api/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { PretestPanel, usePretest } from "@/src/features/punisher/pretest";
import { LectureLayout, LectureTopBar } from "../lecture-viewer";
import { currentStep, failureLines, stageLabel, type StageState, type StepKey } from "./lib/stageState";
import { useInstantStream } from "./useInstantStream";
import { useRetryInstant } from "./useRetryInstant";
import { useLocale, translate } from "@/src/shared/i18n";
import { messages } from "./i18n";

const STEPS: Array<{ key: StepKey; label: "stepWrite" | "stepReview" | "stepFix" }> = [
  { key: "write", label: "stepWrite" },
  { key: "review", label: "stepReview" },
  { key: "fix", label: "stepFix" },
];

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

/** 작성 → 검수 → (고침) — 지금 칸만 단계 이름(고치는 중 n/2 등)으로 강조한다. 고침 칸은 고쳐 쓰기가 시작돼야 나온다. */
function StageSteps({ state }: { state: StageState }) {
  const locale = useLocale();
  const t = translate(messages, locale);
  const now = currentStep(state.stage);
  const steps = STEPS.filter((s) => s.key !== "fix" || state.attempt > 0);
  return (
    <ol className="flex flex-wrap items-center gap-2 text-sm">
      {steps.map((s, index) => (
        <li key={s.key} aria-current={s.key === now ? "step" : undefined} className="flex items-center gap-2">
          {index > 0 && <span aria-hidden="true" className="text-fg-muted">→</span>}
          <span
            className={cn(
              "rounded-full px-2.5 py-1",
              s.key === now ? "bg-accent-subtle font-medium text-accent-text" : "text-fg-muted",
            )}
          >
            {s.key === now ? stageLabel(state, locale) : t[s.label]}
          </span>
        </li>
      ))}
    </ol>
  );
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
        <Button size="sm" className="self-start" loading={retry.isPending} onClick={retry.run}>
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
  const queryClient = useQueryClient();
  const state = useInstantStream(jobId);
  const pretest = usePretest({ jobId });
  const { stage, lectureId } = state;

  useEffect(() => {
    if (stage !== "published" || !lectureId) { return; }
    void invalidateLearningState(queryClient);
    router.replace(lectureHref(lectureId));
  }, [stage, lectureId, router, queryClient]);

  const busy = stage !== "published" && stage !== "failed";
  const elapsed = useElapsedLabel(jobId, busy);

  return (
    <div className="flex w-full flex-col">
      <LectureTopBar title={translate(messages, locale).title} />
      <LectureLayout sections={[]} totalMinutes={0} activeAnchor="">
        <div aria-busy={busy} className="flex flex-col gap-4 pt-4">
          {stage === "failed" ? (
            <FailurePanel state={state} jobId={jobId} />
          ) : (
            <div className="flex min-h-touch flex-wrap items-center gap-3 border-y border-line py-2">
              {/* 단계가 바뀔 때만 읽히도록, 매초 바뀌는 경과 시간은 알림 영역 밖에 둔다. */}
              <div role="status" aria-live="polite" className="flex flex-wrap items-center gap-3">
                {busy && <Spinner size="sm" label={null} />}
                <StageSteps state={state} />
                {stage === "published" && <span className="text-sm font-medium text-success">{stageLabel(state, locale)}</span>}
              </div>
              {elapsed && <span className="ml-auto text-xs text-fg-muted tabular-nums">{elapsed}</span>}
            </div>
          )}
          <PretestPanel detail={pretest.data} />
        </div>
      </LectureLayout>
    </div>
  );
}
