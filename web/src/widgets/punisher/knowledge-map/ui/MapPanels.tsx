
import { Link } from "@/src/shared/lib/router";
import { Inbox } from "lucide-react";
import { BottomSheet, EmptyState, List, ListRow, Section } from "@/src/shared/ui/kit";
import {
  MAP_STATE_LABELS,
  MAP_STATE_ORDER,
  MAP_STATE_STYLES,
  MapStateIcon,
  areaLabel,
} from "@/src/entities/punisher";
import type { AreaCoverage, FrontierItem, MapRequest } from "@/src/shared/api/punisher";

/** 커버리지 숫자의 뜻(툴팁) */
const COVERAGE_HINT = "완료 + 이미 앎 / 전체";

/** 범례 시트 — 색 + 테두리 모양 + 아이콘 + 텍스트(색만으로 구분하지 않음). */
export function MapLegendSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <BottomSheet open={open} onClose={onClose} title="범례" desktop="dialog">
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
      <p className="mt-4 text-xs text-fg-muted">노드 너비 = 깊이, 화살표 = 선행 관계(왼쪽에서 오른쪽)</p>
    </BottomSheet>
  );
}

/** 영역별 학습 신호 한 줄("학습 3 · 진행 1 · 요청 1") — 강점/약점 판단의 기초 데이터. 모두 0이면 "". */
export function areaSignals(c: AreaCoverage): string {
  return [
    c.completed + c.known > 0 ? `학습 ${c.completed + c.known}` : "",
    c.inProgress > 0 ? `진행 ${c.inProgress}` : "",
    c.requested > 0 ? `요청 ${c.requested}` : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

export function CoverageList({ coverage }: { coverage: AreaCoverage[] }) {
  if (coverage.length === 0) { return null; }
  return (
    <Section title="영역별 커버리지">
      <ul role="list" className="flex flex-col gap-2.5">
        {coverage.map((c) => {
          const done = c.completed + c.known;
          const percent = c.total > 0 ? Math.round((done / c.total) * 100) : 0;
          return (
            <li key={c.area} className="text-xs">
              <div className="flex justify-between text-fg-secondary">
                <span>{areaLabel(c.area)}</span>
                <span title={COVERAGE_HINT}>
                  {done}/{c.total} ({percent}%)
                </span>
              </div>
              {areaSignals(c) && <p className="text-fg-muted">{areaSignals(c)}</p>}
              <div
                role="progressbar"
                aria-label={`${areaLabel(c.area)} 커버리지`}
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
  return (
    <Section title="다음에 공부하면 좋은 것">
      {items.length === 0 ? (
        <EmptyState compact title="지금 열려 있는 다음 개념이 없어요" description="아는 개념을 표시하거나 강의를 완료하면 열려요." />
      ) : (
        <List>
          {items.map((f) => (
            <ListRow
              key={f.id}
              onClick={() => onSelect(f.id)}
              leading={<MapStateIcon state="recommended" className="size-4 text-accent-text" />}
              titleLines={1}
              title={f.name}
              meta={`${areaLabel(f.area)} · 배우면 열리는 개념 ${f.unlocks}개`}
            />
          ))}
        </List>
      )}
    </Section>
  );
}

/** 아직 개념을 찾는 중인 학습 요청(RSS "30분 학습" 등) — 해석되면 지도 노드에 "요청됨"으로 붙는다. */
export function PendingRequestList({ requests, highlightId }: { requests: MapRequest[]; highlightId: string | null }) {
  if (requests.length === 0) { return null; }
  return (
    <Section title="분류 중인 학습 요청" description="어느 개념에 붙을지 찾는 중이에요. 찾으면 지도에 '요청됨'으로 표시돼요.">
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
            meta={`${r.source === "rss" ? "RSS" : "직접 입력"} · 학습 화면에서 시작`}
          />
        ))}
      </List>
    </Section>
  );
}
