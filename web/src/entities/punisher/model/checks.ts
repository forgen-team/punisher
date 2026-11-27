import { useEffect, useRef } from "react";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
  PunisherApiError,
  describePunisherError,
  errorCodeMessage,
  punisherApi,
  punisherKeys,
  punisherQueryDefaults,
  type CheckDetail,
  type Confidence,
  type ConceptMastery,
  type DiagnosticChoice,
  type DiagnosticRequest,
} from "@/src/shared/api/punisher";
import { CHECK_ERROR_MESSAGES, type CheckErrorCode, type SelfVerdict } from "../ui/constants";
import { invalidateLearningState } from "./concepts";

/** 채점·문항 생성이 도는 동안 이 간격으로 다시 확인한다. */
const CHECK_POLL_MS = 2500;

/** 확인 결과가 반영되면 숙달·학습 상태(지도·오늘·개념 상세 등)와 복습 요약·기록·지표를 새로 받는다. */
export function invalidateMastery(queryClient: QueryClient): void {
  for (const queryKey of [
    ["punisher", "mastery"],
    ["punisher", "misconceptions"],
    punisherKeys.masteryMetrics(),
    punisherKeys.reviewSummary(),
    punisherKeys.checkHistory(),
  ]) {
    void queryClient.invalidateQueries({ queryKey });
  }
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

/** 사전 진단(예측 질문 · 이미 앎 확인 · 배치 진단) 세션을 연다 — 예측 질문은 낼 것이 없으면 undefined. */
export function useStartDiagnostic() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: DiagnosticRequest) => punisherApi.startDiagnostic(body),
    onSuccess: (detail) => {
      if (detail) { queryClient.setQueryData(punisherKeys.check(detail.session.id), detail); }
    },
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
  return useCheckAction(({ text, confidence }: { text: string; confidence?: Confidence }) =>
    punisherApi.answerCheck(id, text, confidence));
}

/** 예측·배치 문항 — 보기 또는 '모르겠어요'(피드백 없이 기록하고 다음 문항으로). */
export function useCheckChoice(id: string) {
  return useCheckAction((answer: DiagnosticChoice) => punisherApi.answerChoice(id, answer));
}

/** '그냥 알려줘' — 현재 문항의 모범답을 공개하고 문항을 끝낸다. */
export function useCheckReveal(id: string) {
  return useCheckAction<void>(() => punisherApi.revealCheck(id));
}

/** '건너뛰기' — 지금 문항을 답하지 않고 넘긴다(근거 아님, 그 목표 복습은 가까운 날로). */
export function useCheckSkip(id: string) {
  return useCheckAction((itemId: string) => punisherApi.skipCheckItem(id, itemId));
}

/** '내 답이 맞아요' — 그 문항의 마지막 판정 턴을 다른 모델이 다시 채점한다(202). */
export function useCheckDispute(id: string) {
  return useCheckAction(({ turnId, reason }: { turnId: string; reason?: string }) =>
    punisherApi.disputeCheck(id, turnId, reason),
  );
}

/** 자기 채점(복습) — 답을 보낸 뒤 맞음·부분·틀림을 고르면 문항이 끝난다. */
export function useCheckSelfGrade(id: string) {
  return useCheckAction((verdict: SelfVerdict) => punisherApi.selfGradeCheck(id, verdict));
}

/** 실패한 문항 생성·턴 다시 시도. */
export function useCheckRetry(id: string) {
  return useCheckAction<void>(() => punisherApi.retryCheck(id));
}

/** 포기 — 이미 반영된 근거는 남는다. */
export function useCheckAbandon(id: string) {
  return useCheckAction<void>(() => punisherApi.abandonCheck(id));
}

/** '그만하기' 되돌리기 — 같은 문항에서 다시 진행. */
export function useCheckResume(id: string) {
  return useCheckAction<void>(() => punisherApi.resumeCheck(id));
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
 * 확인·복습 화면의 오류 문구 — 400 은 서버 code 별 안내, 409 는 code 별 안내(모르는 code 면 서버 문구)
 * (공통 describePunisherError 의 강의 생성 409 문구가 섞이지 않게). 그 밖은 공통 문구.
 */
export function describeCheckError(error: unknown): string {
  if (!(error instanceof PunisherApiError)) { return describePunisherError(error); }
  if (error.status === 409) {
    const known = CHECK_ERROR_MESSAGES[error.code as CheckErrorCode];
    return known ?? errorCodeMessage(error.code) ?? error.message;
  }
  if (error.status === 400) { return CHECK_ERROR_MESSAGES[error.code as CheckErrorCode] ?? CHECK_ERROR_MESSAGES.BAD_REQUEST; }
  return describePunisherError(error);
}
