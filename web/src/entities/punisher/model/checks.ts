
import { useEffect, useRef } from "react";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
  CHECK_MAX_ANSWER_CHARS,
  CHECK_MAX_DISPUTE_CHARS,
  PunisherApiError,
  describePunisherError,
  punisherApi,
  punisherKeys,
  punisherQueryDefaults,
  type CheckDetail,
  type ConceptMastery,
} from "@/src/shared/api/punisher";
import { invalidateLearningState } from "./concepts";

/** 채점·문항 생성이 도는 동안 이 간격으로 다시 확인한다. */
const CHECK_POLL_MS = 2500;

/** 확인 결과가 반영되면 숙달·학습 상태(지도·오늘·개념 상세 등)를 새로 받는다. */
function invalidateMastery(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: ["punisher", "mastery"] });
  void invalidateLearningState(queryClient);
}

/** 동작 응답을 세션 캐시에 넣고, 끝난 동작이면(pending 아님) 관련 쿼리를 무효화한다. */
function applyDetail(queryClient: QueryClient, detail: CheckDetail): void {
  queryClient.setQueryData(punisherKeys.check(detail.session.id), detail);
  if (!detail.pending) { invalidateMastery(queryClient); }
}

/** 확인 세션 — pending 인 동안만 폴링하고, pending 이 끝나면 숙달·학습 상태를 무효화한다. */
export function useCheckSession(id: string) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: punisherKeys.check(id),
    queryFn: () => punisherApi.getCheck(id),
    enabled: Boolean(id),
    ...punisherQueryDefaults,
    refetchInterval: (q) => (q.state.data?.pending ? CHECK_POLL_MS : false),
  });
  const pending = query.data?.pending;
  const wasPending = useRef(false);
  useEffect(() => {
    if (pending === undefined) { return; }
    if (wasPending.current && !pending) { invalidateMastery(queryClient); }
    wasPending.current = pending;
  }, [pending, queryClient]);
  return query;
}

/** '확인하기' — 강의의 확인 세션을 만든다(열린 세션이 있으면 서버가 그것을 돌려준다). */
export function useStartCheck() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (lectureId: string) => punisherApi.startCheck(lectureId),
    onSuccess: (detail) => queryClient.setQueryData(punisherKeys.check(detail.session.id), detail),
  });
}

function useCheckAction<T>(run: (variables: T) => Promise<CheckDetail>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: run,
    onSuccess: (detail) => applyDetail(queryClient, detail),
  });
}

/** 현재 문항에 답하기(202 — 튜터 응답은 폴링으로 받는다). */
export function useCheckAnswer(id: string) {
  return useCheckAction((text: string) => punisherApi.answerCheck(id, text));
}

/** '그냥 알려줘' — 현재 문항의 모범답을 공개하고 문항을 끝낸다. */
export function useCheckReveal(id: string) {
  return useCheckAction<void>(() => punisherApi.revealCheck(id));
}

/** '내 답이 맞아요' — 그 문항의 마지막 판정 턴을 다른 모델이 다시 채점한다(202). */
export function useCheckDispute(id: string) {
  return useCheckAction(({ turnId, reason }: { turnId: string; reason?: string }) =>
    punisherApi.disputeCheck(id, turnId, reason),
  );
}

/** 실패한 문항 생성·턴 다시 시도. */
export function useCheckRetry(id: string) {
  return useCheckAction<void>(() => punisherApi.retryCheck(id));
}

/** 포기 — 이미 반영된 근거는 남는다. */
export function useCheckAbandon(id: string) {
  return useCheckAction<void>(() => punisherApi.abandonCheck(id));
}

/** 개념의 목표별 숙달(레벨·다음 복습일·채점 주체). */
export function useMastery(conceptId: string | undefined) {
  return useQuery({
    queryKey: punisherKeys.mastery(conceptId ?? ""),
    queryFn: () => punisherApi.getMastery(conceptId ?? ""),
    enabled: Boolean(conceptId),
    ...punisherQueryDefaults,
  });
}

/** '다시 배우기' — 그 목표를 다시 배울 수 있는 상태로 돌린다(개념은 진행 중으로). */
export function useRelearn() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (objectiveId: string) => punisherApi.relearnObjective(objectiveId),
    onSuccess: (mastery: ConceptMastery) => {
      queryClient.setQueryData(punisherKeys.mastery(mastery.conceptId), mastery);
      invalidateMastery(queryClient);
    },
  });
}

/**
 * 확인 동작 오류 문구 — 409(처리 중·지금 할 수 없는 동작)는 서버의 한국어 안내를 그대로,
 * 400(입력 오류)은 영문 서버 문구 대신 원인별 안내로 바꾼다. 그 밖은 공통 문구.
 */
export function describeCheckError(error: unknown): string {
  if (!(error instanceof PunisherApiError)) { return describePunisherError(error); }
  if (error.status === 409) { return error.message; }
  if (error.status === 400) {
    if (error.message.includes("objectives")) { return "이 강의에는 확인할 학습 목표가 없어요."; }
    if (error.message.includes("reason")) { return `사유는 ${CHECK_MAX_DISPUTE_CHARS}자까지 쓸 수 있어요.`; }
    return `답은 1~${CHECK_MAX_ANSWER_CHARS}자로 적어 주세요.`;
  }
  return describePunisherError(error);
}
