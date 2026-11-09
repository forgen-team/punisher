
import { useState, type ReactNode } from "react";
import { Badge, Button, EmptyState, ErrorState, List, ListRow, ListSkeleton, Menu, Spinner } from "@/src/shared/ui/kit";
import {
  STUDY_REQUEST_SOURCE_LABELS,
  STUDY_REQUEST_STATUS_LABELS,
  useConcepts,
  useDismissStudyRequest,
  useRetryStudyRequest,
  useStudyRequests,
} from "@/src/entities/punisher";
import { describePunisherError, type StudyRequest } from "@/src/shared/api/punisher";

/** 요청한 주제 자체(첫 해석 개념) — '시작'은 여기로 간다. */
function topicConcept(req: StudyRequest): string {
  return req.resolvedConceptIds?.[0] || req.startingConceptId || "";
}

/** 주제로 가는 길에서 아직 안 배운 첫 선행 개념 — 주제와 다를 때만 '기초부터'로 제안한다. */
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
  if (!error) {return "개념을 찾지 못했어요.";}
  if (error.includes("no matching concept")) {return "지도에 맞는 개념이 없어요. 문구를 바꿔 다시 요청해 보세요.";}
  if (/budget|daily/i.test(error)) {return "오늘 생성 한도에 걸렸어요. 잠시 뒤 다시 시도해 주세요.";}
  return "개념을 찾는 중 문제가 생겼어요. 다시 시도해 주세요.";
}

function metaOf(req: StudyRequest, phase: ResolvePhase, names: ReadonlyMap<string, string>): string {
  const source = STUDY_REQUEST_SOURCE_LABELS[req.source] ?? req.source;
  let status = req.learned ? "" : (STUDY_REQUEST_STATUS_LABELS[req.status] ?? req.status);
  if (phase === "ready" && !req.learned) {
    const topic = names.get(topicConcept(req));
    const prereq = names.get(prerequisiteConcept(req));
    if (topic) {status = prereq ? `${topic} · 선행 개념: ${prereq}` : topic;}
  }
  if (phase === "searching") {status = "개념 찾는 중 · 보통 1분 안에 끝나요";}
  if (phase === "queued") {status = "곧 개념 찾기를 시작해요";}
  if (phase === "failed") {status = failureText(req.resolveError);}
  return [source, status].filter(Boolean).join(" · ");
}

interface StudyRequestListProps {
  starting: boolean;
  onStart: (conceptId: string, query: string) => void;
}

/** 진행 중(아직 안 배움)과 완료(요청한 주제 개념을 다 배움)를 나눈다. 목록에서 뺀 요청은 숨긴다. */
export function splitStudyRequests(list: readonly StudyRequest[]): { active: StudyRequest[]; done: StudyRequest[] } {
  const visible = list.filter((r) => r.status !== "dismissed");
  return { active: visible.filter((r) => !r.learned), done: visible.filter((r) => r.learned) };
}

/**
 * 학습 요청(단독판: 직접 입력, witems: RSS 등) (design §3.7) — 행: 요청 문구 / 출처 · 상태, 오른쪽에 시작 + ⋯(목록에서 빼기).
 * 요청한 주제 개념의 강의를 다 보면 아래 접힌 '완료 N'으로 옮긴다.
 */
export function StudyRequestList({ starting, onStart }: StudyRequestListProps) {
  const { data, isLoading, error, refetch, isRefetching } = useStudyRequests(50);
  const retry = useRetryStudyRequest();
  const dismiss = useDismissStudyRequest();
  const [lastDismissed, setLastDismissed] = useState<StudyRequest | null>(null);
  const { data: concepts } = useConcepts();
  const names = new Map((concepts ?? []).map((c) => [c.id, c.name] as const));
  if (isLoading) {
    return (
      <div aria-busy="true">
        <ListSkeleton rows={2} />
      </div>
    );
  }
  if (error) {
    return <ErrorState compact message={describePunisherError(error)} onRetry={() => refetch()} retrying={isRefetching} />;
  }
  const { active, done } = splitStudyRequests(data ?? []);

  const remove = (req: StudyRequest) => {
    setLastDismissed(req);
    dismiss.mutate({ id: req.id, dismissed: true });
  };
  const menu = (req: StudyRequest) => (
    <Menu label={`${req.query} 동작`} items={[{ key: "dismiss", label: "목록에서 빼기", onSelect: () => remove(req) }]} />
  );

  function renderActive(req: StudyRequest) {
    const conceptId = topicConcept(req);
    const prereqId = prerequisiteConcept(req);
    const phase = phaseOf(req);
    let action: ReactNode = null;
    if (phase === "searching" || phase === "queued") {
      action = <Spinner size="sm" label="개념 찾는 중" />;
    } else if (phase === "failed") {
      action = (
        <Button size="sm" variant="secondary" loading={retry.isPending && retry.variables === req.id} onClick={() => retry.mutate(req.id)}>
          다시 시도
        </Button>
      );
    } else if (conceptId) {
      action = (
        <>
          {prereqId ? (
            <Button size="sm" variant="ghost" disabled={starting} onClick={() => onStart(prereqId, req.query)}>
              기초부터
            </Button>
          ) : null}
          <Button size="sm" variant="secondary" disabled={starting} onClick={() => onStart(conceptId, req.query)}>
            시작
          </Button>
        </>
      );
    }
    return (
      <ListRow
        key={req.id}
        titleLines={1}
        title={req.query}
        meta={metaOf(req, phase, names)}
        trailing={
          <span className="flex items-center gap-1">
            {action}
            {menu(req)}
          </span>
        }
      />
    );
  }

  return (
    <div className="space-y-2">
      {active.length === 0 ? (
        <EmptyState compact title="진행 중인 학습 요청이 없습니다" description="공부하고 싶은 주제를 위에 적어 두면 개념을 찾아 강의를 미리 만들어 둬요." />
      ) : (
        <List>{active.map((req) => renderActive(req))}</List>
      )}
      {lastDismissed && (
        <p role="status" className="flex items-center gap-2 text-sm text-fg-secondary">
          목록에서 뺐어요 · {lastDismissed.query}
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              dismiss.mutate({ id: lastDismissed.id, dismissed: false });
              setLastDismissed(null);
            }}
          >
            되돌리기
          </Button>
        </p>
      )}
      {done.length > 0 && (
        <details className="group">
          <summary className="flex min-h-touch cursor-pointer list-none items-center justify-between text-sm font-medium text-fg-secondary">
            완료 {done.length}
            <span className="text-xs font-normal text-fg-muted">요청한 주제를 다 배운 요청 · 펼치기</span>
          </summary>
          <List>
            {done.map((req) => (
              <ListRow
                key={req.id}
                titleLines={1}
                title={req.query}
                meta={metaOf(req, "ready", names)}
                trailing={
                  <span className="flex items-center gap-1">
                    <Badge tone="success">학습함</Badge>
                    {menu(req)}
                  </span>
                }
              />
            ))}
          </List>
        </details>
      )}
    </div>
  );
}
