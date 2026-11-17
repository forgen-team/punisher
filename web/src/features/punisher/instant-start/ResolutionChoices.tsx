
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
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

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
  const t = useT(messages);
  const resolution = response.resolution;
  const nameOf = (id: string) => concepts?.find((c) => c.id === id)?.name ?? id;

  if (response.status === "already_learned") {
    const id = response.conceptId ?? resolution?.conceptIds?.[0] ?? "";
    return (
      <p role="status" className="flex flex-wrap items-center gap-1.5 text-sm text-fg">
        <CheckCircle2 className="size-4 text-success" aria-hidden="true" />
        <strong className="font-semibold">{nameOf(id)}</strong>{t.alreadyLearned}
        <Link className="text-accent-text underline underline-offset-2" href={mapHref(id)}>
          {t.viewOnMap}
        </Link>
      </p>
    );
  }

  const conceptIds = resolution?.conceptIds ?? [];
  if (!resolution || conceptIds.length === 0) {
    return (
      <p role="status" className="text-sm text-fg-secondary">
        {t.noMatch(resolution?.error ? ` (${resolution.error})` : "")}
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
              {t.startingPoint}<strong className="font-semibold">{nameOf(start)}</strong>
            </>
          ) : (
            t.pathAllLearned
          )}
        </p>
        {start && (
          <Button size="sm" disabled={disabled} onClick={() => onPick(start)}>
            {t.startFromStart}
          </Button>
        )}
      </div>
      <p className="text-xs text-fg-muted">{t.method(RESOLUTION_METHOD_LABELS[resolution.method] ?? resolution.method)}</p>
      <div role="group" aria-label={t.pickRelated} className="flex gap-2 overflow-x-auto pb-1">
        {conceptIds.map((id) => (
          <Button key={id} variant="secondary" size="sm" disabled={disabled} onClick={() => onPick(id)}>
            {nameOf(id)}
          </Button>
        ))}
      </div>
      {path.length > 0 && (
        <div>
          <p className="text-xs font-semibold text-fg-secondary">{t.prerequisitePath}</p>
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
                trailing={step.isNext ? <Badge tone="accent">{t.next}</Badge> : undefined}
              />
            ))}
          </List>
        </div>
      )}
    </div>
  );
}
