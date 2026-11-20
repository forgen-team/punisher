import { useCallback, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { punisherApi } from "@/src/shared/api/punisher";
import { invalidateLearningState } from "./concepts";

const CONCURRENCY = 3;

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
