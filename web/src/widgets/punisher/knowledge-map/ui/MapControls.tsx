import { Info } from "lucide-react";
import { IconButton, SegmentedControl, SegmentedControlItem, Select } from "@forgen-team/ui";
import { AREA_LABELS, GOAL_STATUS_LABELS } from "@/src/entities/punisher";
import { PUNISHER_AREAS, type AreaCoverage, type Goal, type GraphChapter } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** 지도 보기 방식 — URL ?view= (없으면 폰은 목록, 넓은 화면은 그래프). */
export type MapView = "graph" | "list";

/** 영역 선택값 중 챕터를 뜻하는 접두사(값 = chapter:<id>). */
export const CHAPTER_OPTION_PREFIX = "chapter:";

/** 영역 진행 % — 배움 + 이미 앎 / 전체. */
export function coveragePercent(c: Pick<AreaCoverage, "total" | "completed" | "known">): number {
  return c.total > 0 ? Math.round(((c.completed + c.known) / c.total) * 100) : 0;
}

interface MapControlsProps {
  area: string;
  view: MapView;
  onViewChange: (view: MapView) => void;
  goalId: string;
  goals: Goal[];
  /** 내 챕터(영역 선택지에 '새 챕터: <이름>'으로) */
  chapters: GraphChapter[];
  /** 영역 선택지에 "백엔드 · 38%" 처럼 진행 %를 붙이기 위한 영역 요약 */
  coverage: AreaCoverage[];
  onAreaChange: (area: string) => void;
  onGoalChange: (goalId: string) => void;
  onOpenLegend: () => void;
}

/** 지도 Toolbar 한 줄: [영역 ▾ 진행 %] [목표 경로 ▾] [목록 | 그래프] [범례 ⓘ]. */
export function MapControls({
  area, view, onViewChange, chapters, goalId, goals, coverage, onAreaChange, onGoalChange, onOpenLegend,
}: MapControlsProps) {
  const t = useT(messages);
  return (
    <div className="flex flex-wrap items-end gap-2">
      <div className="min-w-0 flex-1 basis-40">
        <Select
          label={t.area}
          value={area}
          disabled={Boolean(goalId)}
          onValueChange={onAreaChange}
          options={[
            { value: "all", label: t.allAreas },
            ...chapters.map((c) => ({ value: `${CHAPTER_OPTION_PREFIX}${c.id}`, label: t.chapterOption(c.title) })),
            ...PUNISHER_AREAS.map((a) => {
              const row = coverage.find((c) => c.area === a);
              return { value: a, label: row ? t.areaOption(AREA_LABELS[a], coveragePercent(row)) : AREA_LABELS[a] };
            }),
          ]}
        />
      </div>
      {/* 목표가 없으면 고를 것이 '없음' 하나뿐이라 숨긴다(목표는 ⋯ 의 목표 관리에서 만든다). */}
      {goals.length > 0 && (
        <div className="min-w-0 flex-1 basis-40">
          <Select
            label={t.goalPath}
            value={goalId}
            onValueChange={onGoalChange}
            options={[
              { value: "", label: t.noGoalPath },
              ...goals.map((g) => ({ value: g.id, label: `${g.title} (${GOAL_STATUS_LABELS[g.status]})` })),
            ]}
          />
        </div>
      )}
      <SegmentedControl aria-label={t.viewLabel} value={view} onValueChange={(v) => onViewChange(v as MapView)}>
        <SegmentedControlItem value="list">{t.viewList}</SegmentedControlItem>
        <SegmentedControlItem value="graph">{t.viewGraph}</SegmentedControlItem>
      </SegmentedControl>
      <IconButton aria-label={t.legend} onClick={onOpenLegend} icon={<Info aria-hidden="true" />} />
    </div>
  );
}
