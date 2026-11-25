
import { forwardRef, useEffect, useRef, type ReactNode } from "react";
import { ArrowRight } from "lucide-react";
import { Button, Section, buttonVariants, notifyUndo } from "@/src/shared/ui/kit";
import { USER_STATE_LABELS, useCompleteLecture, useMarkKnown, useUncompleteLecture, useUndoKnown } from "@/src/entities/punisher";
import type { UserConceptState } from "@/src/shared/api/punisher";
import { Link } from "@/src/shared/lib/router";
import { InlineError } from "@/src/shared/ui/punisher";
import { FeedbackBar } from "./FeedbackBar";
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
 * 완료 / 이미 앎 상태 — 상단 오버플로 메뉴와 본문 끝 마무리 구역이 같은 상태를 공유한다.
 * 완료는 사용자가 누를 때만 기록된다 (design §3.4). 둘 다 되돌릴 수 있어 확인 창 대신 실행 뒤
 * "되돌리기" 알림을 띄운다(UI-PAGE-LAYOUT §5).
 */
export function useLectureActions(
  lectureId: string,
  conceptId: string,
  completed: boolean,
  /** '이미 앎'을 누르면 묻는 흐름("짧게 확인할까요?") — 있으면 그 상태를 쓴다. */
  knownFlow?: { known: ReturnType<typeof useMarkKnown>; ask: (conceptId: string, before?: UserConceptState) => void },
  /** 누르기 전 개념 상태 — 이미 앎 되돌리기가 이 상태로 돌아간다 */
  conceptState: UserConceptState = "not_started",
) {
  const t = useT(messages);
  const complete = useCompleteLecture(lectureId);
  const uncomplete = useUncompleteLecture(lectureId);
  const ownKnown = useMarkKnown();
  const undoKnown = useUndoKnown();
  const known = knownFlow?.known ?? ownKnown;
  const previous = useRef(conceptState);
  if (!known.isPending && !known.isSuccess) { previous.current = conceptState; }

  useOnSuccess(complete.isSuccess, () => {
    notifyUndo(t.completedToast, async () => {
      await uncomplete.mutateAsync();
      complete.reset();
    });
  });
  // 묻는 흐름(knownFlow)은 그쪽(useKnownChoice)이 되돌리기 알림을 띄운다.
  useOnSuccess(!knownFlow && known.isSuccess, () => {
    const id = conceptId;
    const before = previous.current === "known" ? "not_started" : previous.current;
    notifyUndo(t.knownToast, async () => {
      await undoKnown.mutateAsync({ conceptId: id, previous: before });
      known.reset();
    });
  });

  return {
    complete,
    known,
    done: completed || complete.isSuccess,
    knownDone: known.isSuccess,
    markComplete: () => complete.mutate(),
    markKnown: () => (knownFlow ? knownFlow.ask(conceptId, conceptState) : known.mutate(conceptId)),
  };
}

export type LectureActionsState = ReturnType<typeof useLectureActions>;

/**
 * 완료·이미 앎 결과 한 줄. 상단(메뉴 실행 결과)과 본문 끝에 함께 보이므로
 * 낭독 중복을 피하려 한 곳만 live 로 둔다.
 */
export function LectureActionStatus({ actions, live = true }: { actions: LectureActionsState; live?: boolean }) {
  const t = useT(messages);
  const { complete, known } = actions;
  return (
    <div aria-live={live ? "polite" : undefined} className="text-sm">
      <InlineError error={complete.error ?? known.error} />
      {complete.data && (
        <p className="text-success">
          {t.objectivesCompleted(complete.data.objectivesCompleted?.length ?? 0)}{" "}
          {USER_STATE_LABELS[complete.data.conceptState] ?? complete.data.conceptState}
        </p>
      )}
      {known.isSuccess && <p className="text-fg-secondary">{t.markedKnown}</p>}
    </div>
  );
}

interface LectureWrapupProps {
  lectureId: string;
  actions: LectureActionsState;
  getAnchor: () => string;
  /** '완료' 옆에 둘 동작(위젯이 '확인하기'를 넣는다) */
  checkAction?: ReactNode;
  /** checkAction 의 결과 한 줄 */
  checkStatus?: ReactNode;
}

/**
 * 본문 끝 마무리: 완료(주 버튼 1개) · 확인하기 · 이미 앎 · 피드백. 상단 메뉴의 "피드백"이 여기로 스크롤한다.
 * 완료한 뒤에는 다음 행동(확인하기가 주 버튼이 되고, 오늘의 다음 강의로 가는 링크)을 보인다.
 */
export const LectureWrapup = forwardRef<HTMLElement, LectureWrapupProps>(function LectureWrapup(
  { lectureId, actions, getAnchor, checkAction, checkStatus },
  ref,
) {
  const t = useT(messages);
  const { complete, known, done, knownDone } = actions;
  return (
    <Section ref={ref} id="lecture-wrapup" tabIndex={-1} title={t.wrapup} className="border-t border-line pt-4 outline-none">
      <div className="flex flex-col gap-4">
        <div className="flex flex-wrap items-center gap-2">
          <Button variant={done ? "secondary" : "primary"} onClick={actions.markComplete} disabled={done} loading={complete.isPending}>
            {done ? t.completed : t.complete}
          </Button>
          {checkAction}
          <Button variant="secondary" onClick={actions.markKnown} disabled={knownDone} loading={known.isPending}>
            {knownDone ? t.knownDone : t.known}
          </Button>
        </div>
        <LectureActionStatus actions={actions} live={false} />
        {done && (
          <Link href="/" className={buttonVariants({ variant: "ghost", size: "sm", className: "w-fit -ml-2" })}>
            {t.nextLecture}
            <ArrowRight className="size-4" aria-hidden="true" />
          </Link>
        )}
        {checkStatus}
        <FeedbackBar lectureId={lectureId} getAnchor={getAnchor} />
      </div>
    </Section>
  );
});
