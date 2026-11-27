import { useState, type ReactNode } from "react";
import { Button, ErrorState, ListRow, ListSkeleton, Menu, ProgressCircle, useToast, type MenuItemDef } from "@forgen-team/ui";
import { useConcepts, useDismissStudyRequest, useRetryStudyRequest, useStudyRequests } from "@/src/entities/punisher";
import { describePunisherError, type StudyRequest } from "@/src/shared/api/punisher";
import { translate, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { BUNDLE_LIMIT } from "../lib/recommendations";
import { BoxPad, Bundle, BundleList, MoreToggle } from "./TodayBox";

/** 요청한 주제 자체(첫 해석 개념) — [강의 시작]은 여기로 간다. */
function topicConcept(req: StudyRequest): string {
  return req.resolvedConceptIds?.[0] || req.startingConceptId || "";
}

/** 주제로 가는 길에서 아직 안 배운 첫 선행 개념 — 주제와 다를 때만 ⋯ "기초부터 시작"으로 둔다. */
function prerequisiteConcept(req: StudyRequest): string {
  const topic = topicConcept(req);
  return req.startingConceptId && req.startingConceptId !== topic ? req.startingConceptId : "";
}

/** 개념 찾기 단계 — 화면에서 '언제 되는지'를 알 수 있게 진행 상태를 나눈다. */
type ResolvePhase = "searching" | "queued" | "failed" | "ready";

function phaseOf(req: StudyRequest): ResolvePhase {
  if (req.status !== "unresolved") {return "ready";}
  if (req.resolving) {return "searching";}
  return req.resolveAttempts === 0 ? "queued" : "failed";
}

/** 서버 오류 문자열을 사용자 문장으로 — 내부 코드는 그대로 보이지 않게 한다. */
function failureText(error?: string): string {
  const t = translate(messages);
  if (!error) {return t.requestNotFound;}
  if (error.includes("no matching concept")) {return t.requestNoMatch;}
  if (/budget|daily/i.test(error)) {return t.requestBudget;}
  return t.requestFailed;
}

/** meta: (RSS 글에서 · ) 상태 — 준비됨이면 주제 개념 이름. */
export function studyRequestMeta(req: StudyRequest, names: ReadonlyMap<string, string>): string {
  const t = translate(messages);
  const phase = phaseOf(req);
  let status = t.requestReady;
  if (phase === "ready") {
    const topic = names.get(topicConcept(req));
    if (topic) {status = topic;}
  }
  if (phase === "searching") {status = t.requestSearching;}
  if (phase === "queued") {status = t.requestQueued;}
  if (phase === "failed") {status = failureText(req.resolveError);}
  return [req.source === "rss" ? t.fromRss : "", status].filter(Boolean).join(" · ");
}

/** 진행 중(아직 안 배움)과 완료(요청한 주제 개념을 다 배움)를 나눈다. 목록에서 뺀 요청은 숨긴다. */
export function splitStudyRequests(list: readonly StudyRequest[]): { active: StudyRequest[]; done: StudyRequest[] } {
  const visible = list.filter((r) => r.status !== "dismissed");
  return { active: visible.filter((r) => !r.learned), done: visible.filter((r) => r.learned) };
}

interface StudyTopicsBundleProps {
  starting: boolean;
  onStart: (conceptId: string, query: string, studyRequestId: string) => void;
}

/**
 * 묻기·새 주제 › 공부할 주제 — 학습 요청(직접 입력·RSS "30분 학습"·용어/지도 [나중에 공부]) 중 아직 안 배운 것.
 * 셸 안·단독판이 같다. 최대 3 + "n개 더". 비면 묶음 자체가 없다(입력 칸이 곧 안내).
 */
export function StudyTopicsBundle({ starting, onStart }: StudyTopicsBundleProps) {
  const t = useT(messages);
  const toast = useToast();
  const { data, isLoading, error, refetch } = useStudyRequests(50);
  const retry = useRetryStudyRequest();
  const dismiss = useDismissStudyRequest();
  const [expanded, setExpanded] = useState(false);
  const { data: concepts } = useConcepts();
  const names = new Map((concepts ?? []).map((c) => [c.id, c.name] as const));
  if (isLoading) {
    return (
      <Bundle name={t.studyTopics}>
        <div aria-busy="true" className="px-4">
          <ListSkeleton rows={2} />
        </div>
      </Bundle>
    );
  }
  if (error) {
    return (
      <Bundle name={t.studyTopics}>
        <BoxPad>
          <ErrorState title={describePunisherError(error)} onRetry={() => refetch()} />
        </BoxPad>
      </Bundle>
    );
  }
  const { active } = splitStudyRequests(data ?? []);
  if (active.length === 0) { return null; }

  const remove = (req: StudyRequest) => {
    dismiss.mutate({ id: req.id, dismissed: true }, {
      onSuccess: () => toast.undo(t.requestDismissed, () => dismiss.mutate({ id: req.id, dismissed: false }), t.undo),
      onError: (err) => toast.show(describePunisherError(err), { tone: "danger" }),
    });
  };

  function renderRow(req: StudyRequest) {
    const conceptId = topicConcept(req);
    const prereqId = prerequisiteConcept(req);
    const phase = phaseOf(req);
    let action: ReactNode = null;
    if (phase === "searching" || phase === "queued") {
      action = <ProgressCircle size="24" tone="inherit" aria-label={t.requestFinding} />;
    } else if (phase === "failed") {
      action = (
        <Button size="sm" variant="secondary" loading={retry.isPending && retry.variables === req.id} onClick={() => retry.mutate(req.id)}>
          {t.retry}
        </Button>
      );
    } else if (conceptId) {
      action = (
        <Button size="sm" variant="secondary" disabled={starting} onClick={() => onStart(conceptId, req.query, req.id)}>
          {t.startLecture}
        </Button>
      );
    }
    const items: MenuItemDef[] = [
      ...(phase === "ready" && prereqId ? [{ key: "basics", label: t.requestFromBasics, disabled: starting, onSelect: () => onStart(prereqId, req.query, req.id) }] : []),
      { key: "dismiss", label: t.requestDismiss, onSelect: () => remove(req) },
    ];
    return (
      <ListRow
        key={req.id}
        title={req.query}
        meta={studyRequestMeta(req, names)}
        trailing={
          <span className="flex items-center gap-1">
            {action}
            <Menu label={t.requestActions(req.query)} items={items} />
          </span>
        }
      />
    );
  }

  const visible = expanded ? active : active.slice(0, BUNDLE_LIMIT);
  const hidden = active.length - BUNDLE_LIMIT;
  return (
    <Bundle name={t.studyTopics} info={String(active.length)}>
      <BundleList>{visible.map((req) => renderRow(req))}</BundleList>
      <MoreToggle hidden={hidden} expanded={expanded} onToggle={() => setExpanded((v) => !v)} />
    </Bundle>
  );
}
