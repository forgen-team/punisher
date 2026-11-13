
import { useEffect, useRef, useState } from "react";
import { useCheckSession, useStartDiagnostic } from "@/src/entities/punisher";

export type PretestTarget = { lectureId: string } | { jobId: string };

function pretestKey(target: PretestTarget): string {
  return "lectureId" in target ? `lecture:${target.lectureId}` : `job:${target.jobId}`;
}

/**
 * 강의 시작 직후 예측 질문 — 화면을 열 때 한 번 서버에 묻고(낼 것이 없으면 204 → 없음),
 * 받은 세션은 문항 생성(pending) 동안 폴링한다. 시작 실패는 강의를 막지 않도록 조용히 넘긴다.
 */
export function usePretest(target: PretestTarget | null) {
  const start = useStartDiagnostic();
  const [sessionId, setSessionId] = useState("");
  const asked = useRef("");
  const targetRef = useRef(target);
  targetRef.current = target;
  const key = target ? pretestKey(target) : "";
  const { mutate } = start;

  useEffect(() => {
    const current = targetRef.current;
    if (!current || asked.current === key) { return; }
    asked.current = key;
    mutate(
      { kind: "pretest", ...current },
      {
        onSuccess: (detail) => setSessionId(detail?.session.id ?? ""),
        onError: (error) => console.warn("punisher: pretest unavailable", error),
      },
    );
  }, [key, mutate]);

  return useCheckSession(sessionId);
}
