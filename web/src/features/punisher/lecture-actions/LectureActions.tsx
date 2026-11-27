import { useEffect, useRef, type ReactNode } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button, useToast } from "@forgen-team/ui";
import { invalidateLearningState, useCompleteLecture, useUncompleteLecture, useUndoKnown } from "@/src/entities/punisher";
import { punisherApi, type UserConceptState } from "@/src/shared/api/punisher";
import { Link } from "@/src/shared/lib/router";
import { InlineError } from "@/src/shared/ui/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/** 뮤테이션이 막 성공했을 때 한 번(같은 결과로 다시 렌더해도 한 번만). */
function useOnSuccess(isSuccess: boolean, run: () => void) {
  const runRef = useRef(run);
  runRef.current = run;
  const was = useRef(isSuccess);
  useEffect(() => {
    if (isSuccess && !was.current) { runRef.current(); }
    was.current = isSuccess;
  }, [isSuccess]);
}

/**
 * '이미 앎'(강의 ⋯) — 짧은 확인을 진단 대기열에 넣어(queueVerify) 오늘 복습의 확인 대기로 이어 간다.
 * 되돌리기(DELETE)는 대기열에서도 빠진다(서버 UndoKnown).
 */
function useMarkKnownQueued() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (conceptId: string) => punisherApi.markKnown(conceptId, { queueVerify: true }),
    onSuccess: () => invalidateLearningState(queryClient),
  });
}

/**
 * 완료 / 이미 앎 — 둘 다 되돌릴 수 있어 확인 창 대신 실행 뒤 "되돌리기" 토스트(정책 22 §3).
 * 완료는 끝 마무리 줄에서만, 이미 앎은 페이지 동작 ⋯ 에서 부른다.
 */
export function useLectureActions(
  lectureId: string,
  conceptId: string,
  completed: boolean,
  /** 누르기 전 개념 상태 — 이미 앎 되돌리기가 이 상태로 돌아간다 */
  conceptState: UserConceptState = "not_started",
) {
  const t = useT(messages);
  const toast = useToast();
  const complete = useCompleteLecture(lectureId);
  const uncomplete = useUncompleteLecture(lectureId);
  const known = useMarkKnownQueued();
  const undoKnown = useUndoKnown();

  useOnSuccess(complete.isSuccess, () => {
    toast.undo(t.completedToast, () => {
      uncomplete.mutateAsync()
        .then(() => complete.reset())
        .catch(() => toast.show(t.undoFailed, { tone: "danger" }));
    });
  });

  function markKnown() {
    if (!conceptId || known.isPending) { return; }
    const id = conceptId;
    const before: UserConceptState = conceptState === "known" ? "not_started" : conceptState;
    known.mutate(id, {
      onSuccess: () => {
        toast.undo(t.knownToast, () => {
          undoKnown.mutateAsync({ conceptId: id, previous: before })
            .then(() => known.reset())
            .catch(() => toast.show(t.undoFailed, { tone: "danger" }));
        });
      },
      onError: () => toast.show(t.knownFailed, { tone: "danger" }),
    });
  }

  return {
    complete,
    known,
    done: completed || complete.isSuccess,
    knownDone: conceptState === "known" || known.isSuccess,
    markComplete: () => complete.mutate(),
    markKnown,
  };
}

export type LectureActionsState = ReturnType<typeof useLectureActions>;

/** 마지막 확인하기 결과 — "확인 결과 4/5 · 다음 복습 12월 2일" */
export interface CheckOutcomeSummary {
  correct: number;
  total: number;
  /** 화면 언어로 적은 다음 복습일(없으면 생략) */
  nextReview?: string;
}

/** [다음 강의: 제목] — 없으면 [오늘로] */
export interface NextLectureLink {
  title: string;
  href: string;
}

interface LectureWrapupProps {
  actions: LectureActionsState;
  /** [확인하기] 버튼(주/보조는 위젯이 상태로 정한다) */
  checkAction: (primary: boolean) => ReactNode;
  /** checkAction 의 결과 한 줄 */
  checkStatus?: ReactNode;
  /** 이 강의의 학습 목표 수 */
  objectiveCount: number;
  /** 확인하기까지 마쳤으면 결과 */
  checked?: CheckOutcomeSummary | null;
  next: NextLectureLink | null;
}

/**
 * 본문 끝 마무리 줄(T-reading) — 상태 3가지:
 * 읽기 전 [완료로 표시](주) [확인하기] / 완료 뒤 "완료했어요 · 학습 목표 n개" [확인하기](주) [다음 강의] /
 * 확인까지 했으면 "확인 결과 a/b · 다음 복습 날짜" [다음 강의](주).
 */
export function LectureWrapup({ actions, checkAction, checkStatus, objectiveCount, checked, next }: LectureWrapupProps) {
  const t = useT(messages);
  const { complete, done } = actions;
  const nextButton = (primary: boolean) => (
    <Button asChild variant={primary ? "primary" : "secondary"}>
      <Link href={next?.href ?? "/"}>{next ? t.nextLecture(next.title) : t.toToday}</Link>
    </Button>
  );
  let status: string | null = null;
  let buttons: ReactNode;
  if (checked) {
    status = t.checkedLine(`${checked.correct}/${checked.total}`, checked.nextReview);
    buttons = nextButton(true);
  } else if (done) {
    status = t.completedLine(objectiveCount);
    buttons = (
      <>
        {checkAction(true)}
        {nextButton(false)}
      </>
    );
  } else {
    buttons = (
      <>
        <Button variant="primary" onClick={actions.markComplete} loading={complete.isPending}>
          {t.markComplete}
        </Button>
        {checkAction(false)}
      </>
    );
  }
  return (
    <section aria-label={t.wrapup} className="flex flex-col gap-3">
      {status && <p className="text-sm text-fg-secondary">{status}</p>}
      <div className="flex flex-wrap items-center gap-2">{buttons}</div>
      <div aria-live="polite" className="empty:hidden">
        <InlineError error={complete.error} />
        {checkStatus}
      </div>
    </section>
  );
}
