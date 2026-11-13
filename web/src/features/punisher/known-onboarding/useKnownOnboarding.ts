
import { useCallback, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { invalidateLearningState, useConcepts } from "@/src/entities/punisher";
import { punisherApi } from "@/src/shared/api/punisher";

const DISMISS_KEY = "punisher.map.knownOnboarding.dismissed";
/** 완료·앎 개념이 이 수 미만이면 "거의 아무것도 표시하지 않은" 첫 사용자로 본다. */
const FEW_LEARNED = 3;
const CONCURRENCY = 3;

function readDismissed(): boolean {
  try {
    return window.localStorage.getItem(DISMISS_KEY) === "1";
  } catch (err) {
    console.warn("punisher: localStorage read failed", err);
    return false;
  }
}

function writeDismissed(): void {
  try {
    window.localStorage.setItem(DISMISS_KEY, "1");
  } catch (err) {
    console.warn("punisher: localStorage write failed", err);
  }
}

/** 첫 진입 안내 표시 여부 + 닫기(한 번 닫으면 localStorage에 기억). */
export function useKnownOnboardingPrompt() {
  const concepts = useConcepts();
  const [dismissed, setDismissed] = useState(true);

  useEffect(() => setDismissed(readDismissed()), []);

  const learnedCount = useMemo(
    () => (concepts.data ?? []).filter((c) => c.userState === "completed" || c.userState === "known").length,
    [concepts.data],
  );

  const dismiss = useCallback(() => {
    writeDismissed();
    setDismissed(true);
  }, []);

  return { show: Boolean(concepts.data) && !dismissed && learnedCount < FEW_LEARNED, dismiss };
}

export interface MarkKnownProgress {
  done: number;
  total: number;
  failedIds: string[];
}

/**
 * 선택한 개념들을 POST /v1/concepts/{id}/known 로 일괄 저장(동시 3개) 후 지도·추천을 한 번에 갱신.
 * queueVerify: 진단 대기열에 넣는다(일괄 '이미 앎' 기본값). 배치 진단 후보 확정은 이미 확인했으므로 넣지 않는다.
 */
export function useBatchMarkKnown({ queueVerify = true }: { queueVerify?: boolean } = {}) {
  const queryClient = useQueryClient();
  const [progress, setProgress] = useState<MarkKnownProgress | null>(null);
  const [running, setRunning] = useState(false);

  const run = useCallback(
    async (ids: string[]): Promise<MarkKnownProgress> => {
      const failedIds: string[] = [];
      let done = 0;
      setRunning(true);
      setProgress({ done: 0, total: ids.length, failedIds: [] });
      const queue = [...ids];
      async function worker() {
        for (let id = queue.shift(); id !== undefined; id = queue.shift()) {
          try {
            await punisherApi.markKnown(id, { queueVerify });
          } catch (err) {
            console.warn("punisher: mark known failed", id, err);
            failedIds.push(id);
          }
          done += 1;
          setProgress({ done, total: ids.length, failedIds: [...failedIds] });
        }
      }
      await Promise.all(Array.from({ length: Math.min(CONCURRENCY, ids.length) }, () => worker()));
      await invalidateLearningState(queryClient);
      setRunning(false);
      return { done, total: ids.length, failedIds };
    },
    [queryClient, queueVerify],
  );

  return { run, running, progress };
}

export const KNOWN_ONBOARDING_DISMISS_KEY = DISMISS_KEY;
export { writeDismissed as rememberKnownOnboardingDismissed };
