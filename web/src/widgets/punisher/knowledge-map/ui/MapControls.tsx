
import { Info } from "lucide-react";
import { IconButton, Menu, SegmentedControl, SegmentedControlItem, Select, type MenuItemDef } from "@forgen-team/ui";
import { AREA_LABELS, GOAL_STATUS_LABELS } from "@/src/entities/punisher";
import { PUNISHER_AREAS, type AreaCoverage, type Goal, type GraphChapter } from "@/src/shared/api/punisher";
import { translate, useLocale } from "@/src/shared/i18n";
import { areaSignals } from "./MapPanels";
import { messages } from "../i18n";

/** 지도 보기 방식 — URL ?view= (없으면 휴대폰은 목록, 그 밖은 그래프). */
export type MapView = "graph" | "list";

interface MapControlsProps {
  area: string;
  view: MapView;
  onViewChange: (view: MapView) => void;
  goalId: string;
  goals: Goal[];
  /** 내 챕터(영역 선택지에 '새 챕터: <이름>'으로) */
  chapters: GraphChapter[];
  /** 새 챕터 열기(다음 챕터 고르기 화면) */
  onOpenChapter: () => void;
  /** 영역 선택지에 "학습 n · 요청 n"을 붙이기 위한 전체 영역 요약 */
  coverage: AreaCoverage[];
  onAreaChange: (area: string) => void;
  onGoalChange: (goalId: string) => void;
  onOpenLegend: () => void;
  onOpenGoals: () => void;
  onOpenKnown: () => void;
}

/** 영역 선택값 중 챕터를 뜻하는 접두사(값 = chapter:<id>). */
export const CHAPTER_OPTION_PREFIX = "chapter:";

/** 지도 컨트롤 한 줄: 영역 · 목표 경로 · 그래프|목록 · 범례 · ⋯(목표 관리, 아는 개념 표시). */
export function MapControls({
  area, view, onViewChange, chapters, goalId, goals, coverage, onAreaChange, onOpenChapter, onGoalChange, onOpenLegend, onOpenGoals, onOpenKnown,
}: MapControlsProps) {
  const locale = useLocale();
  const t = translate(messages, locale);
  const menuItems: MenuItemDef[] = [
    { key: "chapter", label: t.openChapter, onSelect: onOpenChapter },
    { key: "goals", label: t.manageGoals, onSelect: onOpenGoals },
    { key: "known", label: t.markKnown, onSelect: onOpenKnown },
  ];

  return (
    <div className="flex flex-col gap-2 fold:flex-row fold:items-center">
      <div className="min-w-0 fold:flex-1">
        <Select
          label={t.area}
          value={area}
          disabled={Boolean(goalId)}
          description={goalId ? t.areaDisabledHint : undefined}
          onValueChange={onAreaChange}
          options={[
            { value: "all", label: t.allAreas },
            ...chapters.map((c) => ({ value: `${CHAPTER_OPTION_PREFIX}${c.id}`, label: t.chapterOption(c.title) })),
            ...PUNISHER_AREAS.map((a) => {
              const signals = coverage.find((c) => c.area === a);
              const summary = signals ? areaSignals(signals, locale) : "";
              return { value: a, label: summary ? `${AREA_LABELS[a]} · ${summary}` : AREA_LABELS[a] };
            }),
          ]}
        />
      </div>
      <div className="flex min-w-0 items-center gap-2 fold:flex-1">
        {/* 목표가 없으면 고를 것이 '없음' 하나뿐이라 숨긴다(목표 관리는 ⋯ 메뉴). */}
        <div className="min-w-0 flex-1">
          {goals.length > 0 && (
            <Select
              label={t.goalPath}
              value={goalId}
              onValueChange={onGoalChange}
              options={[
                { value: "", label: t.noGoalPath },
                ...goals.map((g) => ({ value: g.id, label: `${g.title} (${GOAL_STATUS_LABELS[g.status]})` })),
              ]}
            />
          )}
        </div>
        <div className="shrink-0">
          <SegmentedControl aria-label={t.viewLabel} value={view} onValueChange={(v) => onViewChange(v as MapView)}>
            <SegmentedControlItem value="graph">{t.viewGraph}</SegmentedControlItem>
            <SegmentedControlItem value="list">{t.viewList}</SegmentedControlItem>
          </SegmentedControl>
        </div>
        <IconButton aria-label={t.legend} onClick={onOpenLegend} icon={<Info aria-hidden="true" />} />
        <Menu items={menuItems} label={t.mapMenu} />
      </div>
    </div>
  );
}
