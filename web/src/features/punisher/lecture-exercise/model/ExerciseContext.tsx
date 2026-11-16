import { createContext, useContext, useMemo, type ReactNode } from "react";
import { useDeleteExercise, useLearningSettings, useLectureExercises, useSubmitExercise } from "@/src/entities/punisher";
import type { ExerciseAttempt, ExerciseInput, LabOS } from "@/src/shared/api/punisher";

interface ExerciseContextValue {
  attempts: ExerciseAttempt[];
  labOs?: LabOS;
  submit: (input: ExerciseInput) => Promise<ExerciseAttempt>;
  remove: (attemptId: string) => void;
  /** anchor being submitted */
  submitting?: string;
  submitError: unknown;
  submitErrorAnchor?: string;
}

const ExerciseContext = createContext<ExerciseContextValue | null>(null);

/** 강의 화면의 연습·실습 상태 — 스트리밍 미리보기처럼 없으면 블록은 읽기 전용으로 그린다. */
export function ExerciseProvider({ lectureId, children }: { lectureId: string; children: ReactNode }) {
  const list = useLectureExercises(lectureId);
  const settings = useLearningSettings();
  const submit = useSubmitExercise(lectureId);
  const remove = useDeleteExercise(lectureId);
  const value = useMemo<ExerciseContextValue>(
    () => ({
      attempts: list.data?.attempts ?? [],
      labOs: settings.data?.settings.labOs,
      submit: (input) => submit.mutateAsync(input),
      remove: (id) => remove.mutate(id),
      submitting: submit.isPending ? submit.variables?.anchor : undefined,
      submitError: submit.error,
      submitErrorAnchor: submit.error ? submit.variables?.anchor : undefined,
    }),
    [list.data, settings.data, submit, remove],
  );
  return <ExerciseContext.Provider value={value}>{children}</ExerciseContext.Provider>;
}

export function useExerciseContext(): ExerciseContextValue | null {
  return useContext(ExerciseContext);
}

/** The latest attempt of a block (same anchor and content). */
export function findAttempt(attempts: ExerciseAttempt[], anchor: string, hash: string): ExerciseAttempt | undefined {
  return attempts.find((a) => a.anchor === anchor && a.contentHash === hash);
}
