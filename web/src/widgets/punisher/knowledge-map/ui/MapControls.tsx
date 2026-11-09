
import { Info } from "lucide-react";
import { IconButton, Menu, Select, type MenuItem } from "@/src/shared/ui/kit";
import { AREA_LABELS, GOAL_STATUS_LABELS } from "@/src/entities/punisher";
import { PUNISHER_AREAS, type AreaCoverage, type Goal } from "@/src/shared/api/punisher";
import { areaSignals } from "./MapPanels";

interface MapControlsProps {
  area: string;
  goalId: string;
  goals: Goal[];
  /** 영역 선택지에 "학습 n · 요청 n"을 붙이기 위한 전체 영역 요약 */
  coverage: AreaCoverage[];
  onAreaChange: (area: string) => void;
  onGoalChange: (goalId: string) => void;
  onOpenLegend: () => void;
  onOpenGoals: () => void;
  onOpenKnown: () => void;
}

/** 지도 컨트롤 한 줄: 영역 · 목표 경로 · 범례 · ⋯(목표 관리, 아는 개념 표시). */
export function MapControls({
  area, goalId, goals, coverage, onAreaChange, onGoalChange, onOpenLegend, onOpenGoals, onOpenKnown,
}: MapControlsProps) {
  const menuItems: MenuItem[] = [
    { key: "goals", label: "목표 관리", onSelect: onOpenGoals },
    { key: "known", label: "이미 아는 개념 표시", onSelect: onOpenKnown },
  ];

  return (
    <div className="flex items-center gap-2">
      <div className="min-w-0 flex-1">
        <Select
          aria-label="영역"
          value={area}
          disabled={Boolean(goalId)}
          title={goalId ? "목표 경로를 보는 중에는 영역 대신 경로가 표시됩니다" : undefined}
          onChange={(event) => onAreaChange(event.target.value)}
        >
          <option value="all">전체 영역</option>
          {PUNISHER_AREAS.map((a) => {
            const signals = coverage.find((c) => c.area === a);
            const summary = signals ? areaSignals(signals) : "";
            return (
              <option key={a} value={a}>
                {summary ? `${AREA_LABELS[a]} · ${summary}` : AREA_LABELS[a]}
              </option>
            );
          })}
        </Select>
      </div>
      <div className="min-w-0 flex-1">
        <Select aria-label="목표 경로" value={goalId} onChange={(event) => onGoalChange(event.target.value)}>
          <option value="">목표 경로 없음</option>
          {goals.map((g) => (
            <option key={g.id} value={g.id}>
              {g.title} ({GOAL_STATUS_LABELS[g.status]})
            </option>
          ))}
        </Select>
      </div>
      <IconButton aria-label="범례" onClick={onOpenLegend}>
        <Info aria-hidden="true" />
      </IconButton>
      <Menu items={menuItems} label="지도 메뉴" />
    </div>
  );
}
