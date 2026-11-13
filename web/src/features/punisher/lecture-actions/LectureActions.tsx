
import { forwardRef, type ReactNode } from "react";
import { Button, Section } from "@/src/shared/ui/kit";
import { USER_STATE_LABELS, useCompleteLecture, useMarkKnown } from "@/src/entities/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { FeedbackBar } from "./FeedbackBar";

/**
 * 완료 / 이미 앎 상태 — 상단 오버플로 메뉴와 본문 끝 마무리 구역이 같은 상태를 공유한다.
 * 완료는 사용자가 누를 때만 기록된다 (design §3.4).
 */
export function useLectureActions(
  lectureId: string,
  conceptId: string,
  completed: boolean,
  /** '이미 앎'을 누르면 묻는 흐름("짧게 확인할까요?") — 있으면 그 상태를 쓴다. */
  knownFlow?: { known: ReturnType<typeof useMarkKnown>; ask: (conceptId: string) => void },
) {
  const complete = useCompleteLecture(lectureId);
  const ownKnown = useMarkKnown();
  const known = knownFlow?.known ?? ownKnown;
  return {
    complete,
    known,
    done: completed || complete.isSuccess,
    knownDone: known.isSuccess,
    markComplete: () => complete.mutate(),
    markKnown: () => (knownFlow ? knownFlow.ask(conceptId) : known.mutate(conceptId)),
  };
}

export type LectureActionsState = ReturnType<typeof useLectureActions>;

/**
 * 완료·이미 앎 결과 한 줄. 상단(메뉴 실행 결과)과 본문 끝에 함께 보이므로
 * 낭독 중복을 피하려 한 곳만 live 로 둔다.
 */
export function LectureActionStatus({ actions, live = true }: { actions: LectureActionsState; live?: boolean }) {
  const { complete, known } = actions;
  return (
    <div aria-live={live ? "polite" : undefined} className="text-sm">
      <InlineError error={complete.error ?? known.error} />
      {complete.data && (
        <p className="text-success">
          목표 {complete.data.objectivesCompleted?.length ?? 0}개를 완료했습니다. 개념 상태:{" "}
          {USER_STATE_LABELS[complete.data.conceptState] ?? complete.data.conceptState}
        </p>
      )}
      {known.isSuccess && <p className="text-fg-secondary">이 개념을 이미 앎으로 표시했습니다.</p>}
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

/** 본문 끝 마무리: 완료(주 버튼 1개) · 확인하기 · 이미 앎 · 피드백. 상단 메뉴의 "피드백"이 여기로 스크롤한다. */
export const LectureWrapup = forwardRef<HTMLElement, LectureWrapupProps>(function LectureWrapup(
  { lectureId, actions, getAnchor, checkAction, checkStatus },
  ref,
) {
  const { complete, known, done, knownDone } = actions;
  return (
    <Section ref={ref} id="lecture-wrapup" tabIndex={-1} title="강의 마무리" className="border-t border-line pt-4 outline-none">
      <div className="flex flex-col gap-4">
        <div className="flex flex-wrap items-center gap-2">
          <Button onClick={actions.markComplete} disabled={done} loading={complete.isPending}>
            {done ? "완료함" : "완료"}
          </Button>
          {checkAction}
          <Button variant="secondary" onClick={actions.markKnown} disabled={knownDone} loading={known.isPending}>
            {knownDone ? "이미 앎으로 표시됨" : "이미 앎"}
          </Button>
        </div>
        <LectureActionStatus actions={actions} live={false} />
        {checkStatus}
        <FeedbackBar lectureId={lectureId} getAnchor={getAnchor} />
      </div>
    </Section>
  );
});
