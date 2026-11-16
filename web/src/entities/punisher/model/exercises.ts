import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  punisherApi,
  punisherKeys,
  punisherQueryDefaults,
  type ExerciseAttempt,
  type ExerciseInput,
  type LectureExercises,
} from "@/src/shared/api/punisher";

/** 평을 기다리는 동안 다시 읽는 간격과 한도(그 뒤는 '평을 만들지 못했어요'). */
export const EXERCISE_POLL_MS = 2000;
export const EXERCISE_POLL_LIMIT_MS = 30_000;

function stillGrading(data: LectureExercises | undefined, now: number): boolean {
  return Boolean(
    data?.attempts.some((a) => a.status === "pending" && now - new Date(a.createdAt).getTime() < EXERCISE_POLL_LIMIT_MS),
  );
}

/** 강의의 내 연습·실습 시도와 '고칠 곳' — 평을 기다리는 시도가 있으면 잠깐씩 다시 읽는다. */
export function useLectureExercises(lectureId: string, enabled = true) {
  return useQuery({
    queryKey: punisherKeys.exercises(lectureId),
    queryFn: () => punisherApi.listExercises(lectureId),
    enabled,
    ...punisherQueryDefaults,
    refetchInterval: (query) => (stillGrading(query.state.data, Date.now()) ? EXERCISE_POLL_MS : false),
  });
}

function upsert(data: LectureExercises | undefined, attempt: ExerciseAttempt): LectureExercises {
  const rest = (data?.attempts ?? []).filter((a) => !(a.anchor === attempt.anchor && a.contentHash === attempt.contentHash));
  return { attempts: [...rest, attempt], fixes: data?.fixes ?? [] };
}

export function useSubmitExercise(lectureId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ExerciseInput) => punisherApi.submitExercise(lectureId, input),
    onSuccess: (attempt) => {
      queryClient.setQueryData<LectureExercises>(punisherKeys.exercises(lectureId), (prev) => upsert(prev, attempt));
      void queryClient.invalidateQueries({ queryKey: punisherKeys.exercises(lectureId) });
    },
  });
}

export function useDeleteExercise(lectureId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (attemptId: string) => punisherApi.deleteExercise(lectureId, attemptId),
    onSettled: () => queryClient.invalidateQueries({ queryKey: punisherKeys.exercises(lectureId) }),
  });
}
