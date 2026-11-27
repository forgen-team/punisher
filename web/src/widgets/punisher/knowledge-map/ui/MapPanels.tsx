import { Sheet } from "@forgen-team/ui";
import { MAP_STATE_STYLES, MapStateIcon } from "@/src/entities/punisher";
import { useT } from "@/src/shared/i18n";
import { cn } from "@/src/shared/lib/cn";
import { messages } from "../i18n";
import { stateLabel, type MapDisplayState } from "../lib/displayState";

const BASE_STATE: Record<MapDisplayState, Parameters<typeof MapStateIcon>[0]["state"]> = {
  completed: "completed",
  known: "known",
  in_progress: "in_progress",
  recommended: "recommended",
  not_started: "not_started",
  locked: "locked",
  requested: "not_started",
  dismissed: "not_started",
};

/** 상태 견본(노드와 같은 색·테두리 모양·아이콘) — 범례와 지도 목록 보기의 행 앞에 쓴다. 요청됨은 점선 테두리. */
export function MapStateSwatch({ state }: { state: MapDisplayState }) {
  const base = BASE_STATE[state];
  const style = MAP_STATE_STYLES[base];
  return (
    <span
      aria-hidden="true"
      // ListRow 의 leading 은 아이콘을 20px 로 키운다 — 견본 안 아이콘은 노드와 같은 14px 로 고정.
      className={cn("inline-flex size-6 shrink-0 items-center justify-center border-2 text-fg [&_svg]:!size-3.5", state === "dismissed" && "opacity-60")}
      // 필요 없음은 노드처럼 잠김 바탕에 흐리게.
      style={{
        borderColor: style.border,
        background: state === "dismissed" ? MAP_STATE_STYLES.locked.background : style.background,
        borderStyle: state === "requested" ? "dashed" : style.borderStyle,
      }}
    >
      <MapStateIcon state={base} />
    </span>
  );
}

const LEGEND_ORDER: MapDisplayState[] = ["completed", "in_progress", "recommended", "known", "requested", "dismissed", "locked", "not_started"];

/** 범례 대화상자 — 색 + 테두리 모양 + 아이콘 + 텍스트(색만으로 구분하지 않음). */
export function MapLegendSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT(messages);
  const description: Record<MapDisplayState, string> = {
    completed: t.legendCompleted,
    in_progress: t.legendInProgress,
    recommended: t.legendRecommended,
    known: t.legendKnown,
    requested: t.legendRequested,
    dismissed: t.legendDismissed,
    locked: t.legendLocked,
    not_started: t.legendNotStarted,
  };
  return (
    <Sheet open={open} onOpenChange={(o) => { if (!o) { onClose(); } }} title={t.legend} desktop="dialog">
      <ul className="flex flex-col gap-3 text-sm">
        {LEGEND_ORDER.map((state) => (
          <li key={state} className="flex items-center gap-3">
            <MapStateSwatch state={state} />
            <span className="w-20 shrink-0 font-medium text-fg">{stateLabel(t, state)}</span>
            <span className="min-w-0 text-fg-secondary">{description[state]}</span>
          </li>
        ))}
      </ul>
      <p className="mt-4 text-xs text-fg-muted">{t.legendNote}</p>
    </Sheet>
  );
}
