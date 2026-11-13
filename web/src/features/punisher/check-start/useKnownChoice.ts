
import { useRouter } from "@/src/shared/lib/router";
import { useCallback, useState } from "react";
import { useMarkKnown, useStartDiagnostic } from "@/src/entities/punisher";
import { checkHref } from "./useLaunchCheck";

/**
 * 개별 '이미 앎' — 누르면 "짧게 확인할까요?"를 묻는다. 어느 쪽이든 '이미 앎'은 누른 대로 저장하고,
 * '확인하기'는 이어서 목표마다 1문항 확인(known_verify)을 열어 확인 화면으로 간다.
 */
export function useKnownChoice() {
  const router = useRouter();
  const known = useMarkKnown();
  const start = useStartDiagnostic();
  const [conceptId, setConceptId] = useState<string | null>(null);
  const resetStart = start.reset;
  const close = useCallback(() => setConceptId(null), []);
  const ask = useCallback((id: string) => {
    resetStart();
    setConceptId(id);
  }, [resetStart]);

  function skip() {
    if (!conceptId) { return; }
    known.mutate(conceptId);
    setConceptId(null);
  }

  function verify() {
    if (!conceptId) { return; }
    const id = conceptId;
    known.mutate(id, {
      onSuccess: () =>
        start.mutate(
          { kind: "known_verify", conceptId: id },
          { onSuccess: (detail) => { if (detail) { router.push(checkHref(detail.session.id)); } } },
        ),
    });
  }

  return {
    known,
    asking: conceptId !== null,
    ask,
    close,
    skip,
    verify,
    verifying: known.isPending || start.isPending,
    error: start.error,
  };
}

export type KnownChoice = ReturnType<typeof useKnownChoice>;
