
import { useEffect, useRef, useState } from "react";
import { instantStreamUrl, punisherApi } from "@/src/shared/api/punisher";

export type StreamPhase = "connecting" | "generating" | "validating" | "done" | "error";

export interface InstantStreamState {
  phase: StreamPhase;
  text: string;
  lectureId?: string;
  errorMessage?: string;
}

const MAX_RECONNECTS = 3;
const RECONNECT_DELAY_MS = 2000;

function parseData<T>(event: Event): T | null {
  if (!(event instanceof MessageEvent) || typeof event.data !== "string") { return null; }
  try {
    return JSON.parse(event.data) as T;
  } catch {
    return null;
  }
}

/**
 * Consumes GET /v1/lectures/instant/{jobId}/stream (SSE: status, delta, done, error).
 * The server replays all events to new subscribers, so each (re)connect resets the text.
 */
export function useInstantStream(jobId: string): InstantStreamState {
  const [state, setState] = useState<InstantStreamState>({ phase: "connecting", text: "" });
  const finished = useRef(false);

  useEffect(() => {
    finished.current = false;
    let source: EventSource | null = null;
    let attempts = 0;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;

    const finish = (next: Partial<InstantStreamState>) => {
      finished.current = true;
      source?.close();
      setState((prev) => ({ ...prev, ...next }));
    };

    const recoverFromJob = async () => {
      try {
        const job = await punisherApi.getJob(jobId);
        if (job.status === "succeeded" && job.lectureId) { return finish({ phase: "done", lectureId: job.lectureId }); }
        if (job.status === "failed" || job.status === "skipped") {
          return finish({ phase: "error", errorMessage: job.error || "강의 생성에 실패했습니다." });
        }
      } catch (err) {
        console.warn("punisher: job status check failed", err);
      }
      if (attempts >= MAX_RECONNECTS) {
        finish({ phase: "error", errorMessage: "스트림 연결이 끊겼습니다. 잠시 후 새로고침해 주세요." });
        return;
      }
      retryTimer = setTimeout(connect, RECONNECT_DELAY_MS);
    };

    function connect() {
      attempts += 1;
      setState((prev) => ({ ...prev, phase: "connecting", text: "" }));
      const es = new EventSource(instantStreamUrl(jobId));
      source = es;
      es.addEventListener("status", (event) => {
        const data = parseData<{ stage?: string }>(event);
        const phase: StreamPhase = data?.stage === "validating" ? "validating" : "generating";
        setState((prev) => (prev.phase === phase ? prev : { ...prev, phase }));
      });
      es.addEventListener("delta", (event) => {
        const data = parseData<{ text?: string }>(event);
        if (data?.text) {setState((prev) => ({ ...prev, phase: "generating", text: prev.text + data.text }));}
      });
      es.addEventListener("done", (event) => {
        const data = parseData<{ lectureId?: string }>(event);
        if (data?.lectureId) {finish({ phase: "done", lectureId: data.lectureId });}
        else {finish({ phase: "error", errorMessage: "강의가 저장되지 않았습니다." });}
      });
      es.addEventListener("error", (event) => {
        const data = parseData<{ code?: string; message?: string }>(event);
        if (data) {
          finish({ phase: "error", errorMessage: data.message || data.code || "강의 생성에 실패했습니다." });
          return;
        }
        // Transport error (no payload): close and recover via job status.
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
