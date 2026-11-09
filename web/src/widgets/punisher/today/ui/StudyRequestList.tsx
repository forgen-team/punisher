
import type { ReactNode } from "react";
import { Badge, Button, EmptyState, ErrorState, List, ListRow, ListSkeleton } from "@/src/shared/ui/kit";
import {
  STUDY_REQUEST_SOURCE_LABELS,
  STUDY_REQUEST_STATUS_LABELS,
  useStudyRequests,
} from "@/src/entities/punisher";
import { describePunisherError, type StudyRequest } from "@/src/shared/api/punisher";

function startConcept(req: StudyRequest): string {
  return req.startingConceptId || req.resolvedConceptIds?.[0] || "";
}

function metaOf(req: StudyRequest): string {
  const source = STUDY_REQUEST_SOURCE_LABELS[req.source] ?? req.source;
  const status = STUDY_REQUEST_STATUS_LABELS[req.status] ?? req.status;
  return [source, req.learned ? "" : status, req.resolveError ?? ""].filter(Boolean).join(" · ");
}

interface StudyRequestListProps {
  starting: boolean;
  onStart: (conceptId: string, query: string) => void;
}

/** RSS 등에서 온 학습 요청 — 행: 요청 문구 / 출처 · 상태, 오른쪽에 시작 또는 "이미 학습함" (design §3.7). */
export function StudyRequestList({ starting, onStart }: StudyRequestListProps) {
  const { data, isLoading, error, refetch, isRefetching } = useStudyRequests();
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
  const items = (data ?? []).filter((r) => r.status !== "dismissed");
  if (items.length === 0) {
    return <EmptyState compact title="학습 요청이 없습니다" description="RSS에서 '30분 학습'을 누르면 여기에 쌓입니다." />;
  }

  return (
    <List>
      {items.map((req) => {
        const conceptId = startConcept(req);
        let trailing: ReactNode = null;
        if (req.learned) { trailing = <Badge tone="success">이미 학습함</Badge>; }
        else if (conceptId) {
          trailing = (
            <Button size="sm" variant="ghost" disabled={starting} onClick={() => onStart(conceptId, req.query)}>
              시작
            </Button>
          );
        }
        return <ListRow key={req.id} titleLines={1} title={req.query} meta={metaOf(req)} trailing={trailing} />;
      })}
    </List>
  );
}
