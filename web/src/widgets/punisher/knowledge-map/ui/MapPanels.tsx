
import { Link } from "@/src/shared/lib/router";
import { Inbox } from "lucide-react";
import { BottomSheet, EmptyState, List, ListRow, Section } from "@/src/shared/ui/kit";
import {
  MAP_STATE_LABELS,
  MAP_STATE_ORDER,
  MAP_STATE_STYLES,
  MapStateIcon,
  STUDY_REQUEST_SOURCE_LABELS,
  areaLabel,
} from "@/src/entities/punisher";
import type { AreaCoverage, FrontierItem, MapRequest } from "@/src/shared/api/punisher";
import { translate, useLocale, useT, type Locale } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** 범례 시트 — 색 + 테두리 모양 + 아이콘 + 텍스트(색만으로 구분하지 않음). */
export function MapLegendSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT(messages);
  return (
    <BottomSheet open={open} onClose={onClose} title={t.legend} desktop="dialog">
      <ul className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
        {MAP_STATE_ORDER.map((state) => {
          const style = MAP_STATE_STYLES[state];
          return (
            <li key={state} className="flex items-center gap-2 text-fg-secondary">
              <span
                aria-hidden="true"
                className="inline-flex size-6 shrink-0 items-center justify-center rounded-md border-2 text-fg"
                style={{ borderColor: style.border, background: style.background, borderStyle: style.borderStyle }}
              >
                <MapStateIcon state={state} />
              </span>
              {MAP_STATE_LABELS[state]}
            </li>
          );
        })}
      </ul>
      <p className="mt-4 text-xs text-fg-muted">{t.legendNote}</p>
    </BottomSheet>
  );
}

/** 영역별 학습 신호 한 줄("학습 3 · 진행 1 · 요청 1") — 강점/약점 판단의 기초 데이터. 모두 0이면 "". */
export function areaSignals(c: AreaCoverage, locale: Locale): string {
  const t = translate(messages, locale);
  return [
    c.completed + c.known > 0 ? t.signalLearned(c.completed + c.known) : "",
    c.inProgress > 0 ? t.signalInProgress(c.inProgress) : "",
    c.requested > 0 ? t.signalRequested(c.requested) : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

export function CoverageList({ coverage }: { coverage: AreaCoverage[] }) {
  const locale = useLocale();
  const t = translate(messages, locale);
  if (coverage.length === 0) { return null; }
  return (
    <Section title={t.coverageTitle}>
      <ul role="list" className="flex flex-col gap-2.5">
        {coverage.map((c) => {
          const done = c.completed + c.known;
          const percent = c.total > 0 ? Math.round((done / c.total) * 100) : 0;
          return (
            <li key={c.area} className="text-xs">
              <div className="flex justify-between text-fg-secondary">
                <span>{areaLabel(c.area)}</span>
                <span title={t.coverageHint}>
                  {done}/{c.total} ({percent}%)
                </span>
              </div>
              {areaSignals(c, locale) && <p className="text-fg-muted">{areaSignals(c, locale)}</p>}
              <div
                role="progressbar"
                aria-label={t.coverageLabel(areaLabel(c.area))}
                aria-valuenow={percent}
                aria-valuemin={0}
                aria-valuemax={100}
                className="mt-1 flex h-1.5 overflow-hidden rounded-full bg-canvas-hover"
              >
                <div className="h-full bg-success" style={{ width: `${c.total > 0 ? (c.completed / c.total) * 100 : 0}%` }} />
                <div className="h-full bg-fg-muted" style={{ width: `${c.total > 0 ? (c.known / c.total) * 100 : 0}%` }} />
              </div>
            </li>
          );
        })}
      </ul>
    </Section>
  );
}

export function FrontierList({ items, onSelect }: { items: FrontierItem[]; onSelect: (id: string) => void }) {
  const t = useT(messages);
  return (
    <Section title={t.frontierTitle}>
      {items.length === 0 ? (
        <EmptyState compact title={t.frontierEmptyTitle} description={t.frontierEmptyDescription} />
      ) : (
        <List>
          {items.map((f) => (
            <ListRow
              key={f.id}
              onClick={() => onSelect(f.id)}
              leading={<MapStateIcon state="recommended" className="size-4 text-accent-text" />}
              titleLines={1}
              title={f.name}
              meta={t.frontierMeta(areaLabel(f.area), f.unlocks)}
            />
          ))}
        </List>
      )}
    </Section>
  );
}

/** 아직 개념을 찾는 중인 학습 요청(RSS "30분 학습" 등) — 해석되면 지도 노드에 "요청됨"으로 붙는다. */
export function PendingRequestList({ requests, highlightId }: { requests: MapRequest[]; highlightId: string | null }) {
  const t = useT(messages);
  if (requests.length === 0) { return null; }
  return (
    <Section title={t.pendingRequestsTitle} description={t.pendingRequestsDescription}>
      <List>
        {requests.map((r) => (
          <ListRow
            key={r.id}
            href="/"
            linkAs={Link}
            selected={r.id === highlightId}
            leading={<Inbox aria-hidden="true" />}
            titleLines={1}
            title={r.query}
            meta={t.pendingRequestMeta(STUDY_REQUEST_SOURCE_LABELS[r.source] ?? r.source)}
          />
        ))}
      </List>
    </Section>
  );
}
