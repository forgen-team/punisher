
import { Link } from "@/src/shared/lib/router";
import { CheckCircle2 } from "lucide-react";
import { Badge, Button, List, ListRow } from "@/src/shared/ui/kit";
import {
  RESOLUTION_METHOD_LABELS,
  USER_STATE_LABELS,
  areaLabel,
  useConcepts,
} from "@/src/entities/punisher";
import type { InstantResponse } from "@/src/shared/api/punisher";

interface ResolutionChoicesProps {
  response: InstantResponse;
  disabled: boolean;
  onPick: (conceptId: string) => void;
}

function mapHref(id: string): string {
  return `/map?concept=${encodeURIComponent(id)}`;
}

/** 자유 질의 해석 결과: 오늘의 출발점 · 관련 개념 · 선행 경로. */
export function ResolutionChoices({ response, disabled, onPick }: ResolutionChoicesProps) {
  const { data: concepts } = useConcepts();
  const resolution = response.resolution;
  const nameOf = (id: string) => concepts?.find((c) => c.id === id)?.name ?? id;

  if (response.status === "already_learned") {
    const id = response.conceptId ?? resolution?.conceptIds?.[0] ?? "";
    return (
      <p role="status" className="flex flex-wrap items-center gap-1.5 text-sm text-fg">
        <CheckCircle2 className="size-4 text-success" aria-hidden="true" />
        <strong className="font-semibold">{nameOf(id)}</strong>의 학습 목표를 이미 모두 마쳤습니다.
        <Link className="text-accent-text underline underline-offset-2" href={mapHref(id)}>
          지도에서 보기
        </Link>
      </p>
    );
  }

  const conceptIds = resolution?.conceptIds ?? [];
  if (!resolution || conceptIds.length === 0) {
    return (
      <p role="status" className="text-sm text-fg-secondary">
        입력한 내용과 맞는 개념을 찾지 못했습니다.{resolution?.error ? ` (${resolution.error})` : ""} 다른 표현으로 다시
        입력해 보세요.
      </p>
    );
  }

  const path = resolution.path ?? [];
  const start = resolution.startingConceptId;

  return (
    <div className="flex flex-col gap-3 border-y border-line py-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-fg">
          {start ? (
            <>
              오늘의 출발점: <strong className="font-semibold">{nameOf(start)}</strong>
            </>
          ) : (
            "이 경로의 개념을 모두 학습했습니다."
          )}
        </p>
        {start && (
          <Button size="sm" disabled={disabled} onClick={() => onPick(start)}>
            출발점부터 시작
          </Button>
        )}
      </div>
      <p className="text-xs text-fg-muted">해석 방식: {RESOLUTION_METHOD_LABELS[resolution.method] ?? resolution.method}</p>
      <div role="group" aria-label="관련 개념 직접 선택" className="flex gap-2 overflow-x-auto pb-1">
        {conceptIds.map((id) => (
          <Button key={id} variant="secondary" size="sm" disabled={disabled} onClick={() => onPick(id)}>
            {nameOf(id)}
          </Button>
        ))}
      </div>
      {path.length > 0 && (
        <div>
          <p className="text-xs font-semibold text-fg-secondary">선행 경로</p>
          <List>
            {path.map((step, index) => (
              <ListRow
                key={step.conceptId}
                href={mapHref(step.conceptId)}
                linkAs={Link}
                selected={step.isNext}
                titleLines={1}
                title={`${index + 1}. ${step.name}`}
                meta={`${areaLabel(step.area)} · ${USER_STATE_LABELS[step.state] ?? step.state}`}
                trailing={step.isNext ? <Badge tone="accent">다음</Badge> : undefined}
              />
            ))}
          </List>
        </div>
      )}
    </div>
  );
}
