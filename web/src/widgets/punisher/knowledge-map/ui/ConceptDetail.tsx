import { useId } from "react";
import { X } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { Button, ErrorState, IconButton, ListSkeleton, Switch, useToast } from "@forgen-team/ui";
import {
  areaLabel,
  startLectureHref,
  useConceptDetail,
  useConceptLectures,
  useConceptDismiss,
  useCreateStudyRequest,
  useInquiries,
  useMarkKnown,
  useMyQuestions,
  useNotes,
  useUndoKnown,
} from "@/src/entities/punisher";
import { describePunisherError, type ConceptDetail as ConceptDetailData, type GraphNode, type UserConceptState } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { Link, useRouter } from "@/src/shared/lib/router";
import { InlineError } from "@/src/shared/ui/punisher";
import { messages } from "../i18n";
import { mapDisplayState, stateLabel } from "../lib/displayState";

const GRAPH_QUERY_KEY = ["punisher", "graph"] as const;

interface ConceptDetailProps {
  conceptId: string;
  /** 지도에서 이 개념의 노드(요청·추천 표시에 쓴다) — 지도에 없는 개념이면 없다 */
  node?: GraphNode;
  /** panel: 데스크톱 옆 패널(제목·닫기 ×) / screen: 폰 전체 화면(제목은 화면 머리) */
  variant: "panel" | "screen";
  onClose: () => void;
  onSelect: (id: string) => void;
}

/** "질문·메모 n개 보기" — 본문은 기록 › 질문·메모에서만 그린다(찾는 곳 1곳). */
function NotesLink({ conceptId, name }: { conceptId: string; name: string }) {
  const t = useT(messages);
  const notes = useNotes({ conceptId });
  const questions = useMyQuestions(conceptId);
  const inquiries = useInquiries({ concept: conceptId });
  const count = (notes.data?.length ?? 0) + (questions.data?.pages.flatMap((p) => p.items).length ?? 0) + (inquiries.data?.length ?? 0);
  return (
    <Link href={`/history?tab=notes&q=${encodeURIComponent(name)}`} className="text-sm text-accent-text underline underline-offset-2">
      {t.notesLink(count, Boolean(questions.hasNextPage))}
    </Link>
  );
}

/** 상태를 정하는 값 — 지도 노드가 있으면 그것, 없으면 개념 상세. 뺀 개념(필요 없음)은 방금 다시 읽은 상세 쪽을 따른다. */
function stateFrom(c: ConceptDetailData, node: GraphNode | undefined) {
  const base = node ?? { state: c.userState, locked: c.locked, frontier: false };
  return { ...base, dismissed: c.dismissed ?? node?.dismissed };
}

function DetailBody({ c, node, onSelect }: { c: ConceptDetailData; node?: GraphNode; onSelect: (id: string) => void }) {
  const t = useT(messages);
  const router = useRouter();
  const toast = useToast();
  const queryClient = useQueryClient();
  const lectures = useConceptLectures(c.id);
  const mark = useMarkKnown();
  const undo = useUndoKnown();
  const later = useCreateStudyRequest();
  const dismiss = useConceptDismiss();
  const state = mapDisplayState(stateFrom(c, node));
  const completed = c.userState === "completed";
  const known = c.userState === "known";
  const checked = mark.isPending ? true : undo.isPending ? false : known;
  const hasLecture = (lectures.data?.length ?? 0) > 0;
  const previous: UserConceptState = c.userState === "in_progress" ? "in_progress" : "not_started";

  function toggleKnown(next: boolean) {
    if (next) {
      mark.mutate(c.id, {
        onSuccess: () => {
          toast.undo(t.knownToast, () => {
            undo.mutateAsync({ conceptId: c.id, previous }).catch(() => toast.show(t.undoFailed, { tone: "danger" }));
          });
        },
      });
      return;
    }
    undo.mutate(
      { conceptId: c.id, previous: "not_started" },
      {
        onSuccess: () => {
          toast.undo(t.unknownToast, () => {
            mark.mutateAsync(c.id).catch(() => toast.show(t.undoFailed, { tone: "danger" }));
          });
        },
      },
    );
  }

  function studyLater() {
    later.mutate(c.name, {
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: GRAPH_QUERY_KEY });
        toast.show(t.studyLaterToast, { action: { label: t.viewInToday, onAction: () => router.push("/") } });
      },
      onError: () => toast.show(t.studyLaterFailed, { tone: "danger" }),
    });
  }

  // 빼는 동작은 오늘에만 있다(재설계 D7) — 지도에는 뺀 개념을 되돌리는 버튼만.
  function restore() {
    dismiss.mutate(
      { conceptId: c.id, dismissed: false },
      {
        onSuccess: () => toast.show(t.restoreToast),
        onError: () => toast.show(t.restoreFailed, { tone: "danger" }),
      },
    );
  }

  let lectureAction;
  if (lectures.isLoading) {
    lectureAction = <Button variant="primary" className="w-full" disabled loading>{t.openLecture}</Button>;
  } else if (!hasLecture && c.locked) {
    lectureAction = <Button variant="secondary" className="w-full" disabled>{t.lockedLecture}</Button>;
  } else {
    lectureAction = (
      <Button asChild variant="primary" className="w-full">
        <Link href={startLectureHref(c.id)}>{hasLecture ? t.openLecture : t.startLecture}</Link>
      </Button>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-fg-secondary">{`${areaLabel(c.area)} · ${stateLabel(t, state)}`}</p>
      <div className="flex flex-col gap-2">
        {lectureAction}
        {state === "dismissed" && (
          <Button variant="secondary" className="w-full" loading={dismiss.isPending} onClick={restore}>
            {t.restoreRecommend}
          </Button>
        )}
        {state !== "requested" && !completed && (
          <Button variant="secondary" className="w-full" loading={later.isPending} disabled={later.isSuccess} onClick={studyLater}>
            {t.studyLater}
          </Button>
        )}
      </div>
      <div className="border-t border-line-subtle pt-4">
        <Switch
          label={t.known}
          checked={checked}
          disabled={completed || mark.isPending || undo.isPending}
          onCheckedChange={(next) => toggleKnown(Boolean(next))}
          inputProps={{ "aria-label": t.knownLabel(c.name) }}
        />
        <InlineError error={mark.error ?? undo.error ?? later.error} />
      </div>
      {c.prerequisites.length > 0 && (
        <div className="flex flex-col gap-1 border-t border-line-subtle pt-4">
          <span className="text-xs text-fg-muted">{t.prerequisites}</span>
          {c.prerequisites.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => onSelect(p.id)}
              className="w-fit text-left text-sm text-accent-text underline underline-offset-2"
            >
              {p.name}
            </button>
          ))}
        </div>
      )}
      <div className="border-t border-line-subtle pt-4">
        <NotesLink conceptId={c.id} name={c.name} />
      </div>
    </div>
  );
}

/**
 * 개념 상세(`?node=`): 데스크톱은 지도 오른쪽 옆 패널(닫기 ×), 폰은 전체 화면(제목은 화면 머리가 그린다).
 * 영역 · 상태 → [강의 열기/시작] [나중에 공부] → 이미 앎 스위치 → 선행 개념 → 질문·메모 링크.
 */
export function ConceptDetail({ conceptId, node, variant, onClose, onSelect }: ConceptDetailProps) {
  const titleId = useId();
  const t = useT(messages);
  const detail = useConceptDetail(conceptId);
  const c = detail.data;

  const body = detail.isLoading ? (
    <ListSkeleton rows={3} />
  ) : detail.error ? (
    <ErrorState title={describePunisherError(detail.error)} onRetry={() => void detail.refetch()} />
  ) : c ? (
    <DetailBody key={c.id} c={c} node={node} onSelect={onSelect} />
  ) : null;

  if (variant === "screen") {
    return <section aria-label={t.detailLabel}>{body}</section>;
  }
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-4">
      <header className="flex items-start gap-2">
        <h2 id={titleId} className="min-w-0 flex-1 break-keep text-base font-semibold text-fg">{c?.name ?? node?.name ?? conceptId}</h2>
        <IconButton aria-label={t.closeDetail} onClick={onClose} className="-mr-2 -mt-1" icon={<X aria-hidden="true" />} />
      </header>
      {body}
    </section>
  );
}
