
import { useEffect, useRef, useState } from "react";
import { instantStreamUrl, punisherApi } from "@/src/shared/api/punisher";
import { INITIAL_STAGE, reduceStage, type StageState, type StreamEventName } from "./lib/stageState";
import { translate } from "@/src/shared/i18n";
import { messages } from "./i18n";

const MAX_RECONNECTS = 3;
const RECONNECT_DELAY_MS = 2000;
const EVENT_NAMES: StreamEventName[] = ["stage", "status", "delta", "done", "message"];

function parseData(event: Event): Record<string, unknown> | null {
  if (!(event instanceof MessageEvent) || typeof event.data !== "string") { return null; }
  try {
    const value: unknown = JSON.parse(event.data);
    return typeof value === "object" && value !== null ? (value as Record<string, unknown>) : null;
  } catch {
    return null;
  }
}

/**
 * GET /v1/lectures/instant/{jobId}/stream 의 생성 단계만 받는다(초안 본문은 보이지 않는다, v2.2 Q2).
 * 서버는 새 구독자에게 지난 이벤트를 다시 보내므로 (재)연결마다 상태를 처음부터 다시 쌓는다.
 */
export function useInstantStream(jobId: string): StageState {
  const [state, setState] = useState<StageState>(INITIAL_STAGE);
  const finished = useRef(false);

  useEffect(() => {
    finished.current = false;
    let source: EventSource | null = null;
    let attempts = 0;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;

    let current = INITIAL_STAGE;
    const commit = (next: StageState) => {
      if (next === current) { return; }
      current = next;
      setState(next);
    };
    const apply = (event: StreamEventName, data: Record<string, unknown> | null) => {
      commit(reduceStage(current, event, data));
      if (current.stage === "published" || current.stage === "failed") {
        finished.current = true;
        source?.close();
      }
    };

    const recoverFromJob = async () => {
      try {
        const job = await punisherApi.getJob(jobId);
        if (job.status === "succeeded" && job.lectureId) { return apply("done", { lectureId: job.lectureId }); }
        if (job.status === "failed" || job.status === "skipped") {
          return apply("error", { message: job.error || translate(messages).generateFailed });
        }
      } catch (err) {
        console.warn("punisher: job status check failed", err);
      }
      if (attempts >= MAX_RECONNECTS) {
        apply("error", { message: translate(messages).disconnected });
        return;
      }
      retryTimer = setTimeout(connect, RECONNECT_DELAY_MS);
    };

    function connect() {
      attempts += 1;
      commit(INITIAL_STAGE);
      const es = new EventSource(instantStreamUrl(jobId));
      source = es;
      for (const name of EVENT_NAMES) {
        es.addEventListener(name, (event) => apply(name, parseData(event)));
      }
      es.addEventListener("error", (event) => {
        const data = parseData(event);
        if (data) {
          apply("error", data);
          return;
        }
        // 전송 오류(내용 없음): 닫고 잡 상태로 복구한다.
        es.close();
        if (!finished.current) { void recoverFromJob(); }
      });
    }

    connect();
    return () => {
      finished.current = true;
      if (retryTimer) { clearTimeout(retryTimer); }
      source?.close();
    };
  }, [jobId]);

  return state;
}
