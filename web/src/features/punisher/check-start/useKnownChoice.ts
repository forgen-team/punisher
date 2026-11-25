
import { useRouter } from "@/src/shared/lib/router";
import { useCallback, useRef, useState } from "react";
import { useMarkKnown, useStartDiagnostic, useUndoKnown } from "@/src/entities/punisher";
import type { UserConceptState } from "@/src/shared/api/punisher";
import { notifyUndo } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { checkHref } from "./useLaunchCheck";
import { messages } from "./i18n";

/**
 * 개별 '이미 앎' — 누르면 "짧게 확인할까요?"를 묻는다. 어느 쪽이든 '이미 앎'은 누른 대로 저장하고,
 * '확인하기'는 이어서 목표마다 1문항 확인(known_verify)을 열어 확인 화면으로 간다.
 * 저장이 실패하면 선택 창을 닫지 않고 그 오류(knownError)를 보인다.
 * 확인 없이 저장('그냥 이미 앎')하면 되돌리기 알림을 띄운다 — 누르기 전 상태(ask 의 previous)로 돌아간다.
 */
export function useKnownChoice() {
  const router = useRouter();
  const t = useT(messages);
  const known = useMarkKnown();
  const undo = useUndoKnown();
  const start = useStartDiagnostic();
  const [conceptId, setConceptId] = useState<string | null>(null);
  const previous = useRef<UserConceptState>("not_started");
  const resetStart = start.reset;
  const resetKnown = known.reset;
  const close = useCallback(() => setConceptId(null), []);
  const ask = useCallback((id: string, before: UserConceptState = "not_started") => {
    resetStart();
    resetKnown();
    previous.current = before === "known" ? "not_started" : before;
    setConceptId(id);
  }, [resetStart, resetKnown]);

  function skip() {
    if (!conceptId) { return; }
    const id = conceptId;
    const before = previous.current;
    known.mutate(id, {
      onSuccess: () => {
        setConceptId(null);
        notifyUndo(t.knownToast, async () => {
          await undo.mutateAsync({ conceptId: id, previous: before });
          known.reset();
        });
      },
    });
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
    knownError: known.error,
  };
}

export type KnownChoice = ReturnType<typeof useKnownChoice>;
