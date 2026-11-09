
import { Badge, ErrorState, List, ListRow, ListSkeleton } from "@/src/shared/ui/kit";
import { USER_STATE_LABELS, areaLabel, useGoalPath } from "@/src/entities/punisher";
import { describePunisherError } from "@/src/shared/api/punisher";

interface GoalPathViewProps {
  goalId: string;
  /** 단계 선택 시: 지도에서 이 목표 경로 + 개념 상세를 연다 */
  onSelectStep: (conceptId: string) => void;
}

/** 목표별 선행 사슬. 다음 학습 단계는 선택 막대 + 배지 1개, 잠김은 meta 텍스트로. */
export function GoalPathView({ goalId, onSelectStep }: GoalPathViewProps) {
  const { data, isLoading, error, refetch, isRefetching } = useGoalPath(goalId);
  if (isLoading) { return <ListSkeleton rows={3} />; }
  if (error) {
    return <ErrorState compact message={describePunisherError(error)} onRetry={() => refetch()} retrying={isRefetching} />;
  }
  if (!data) { return null; }

  return (
    <div className="flex flex-col gap-3 pl-2">
      {data.paths.map((path) => {
        const steps = path.steps ?? [];
        const target = steps.find((s) => s.isTarget);
        return (
          <div key={path.targetConceptId}>
            <p className="text-xs font-semibold text-fg-secondary">
              {target?.name ?? path.targetConceptId}
              {path.completed && <span className="ml-2 font-normal text-success">경로 완료</span>}
            </p>
            <List>
              {steps.map((s, index) => (
                <ListRow
                  key={s.conceptId}
                  onClick={() => onSelectStep(s.conceptId)}
                  selected={s.isNext}
                  titleLines={1}
                  title={`${index + 1}. ${s.name}`}
                  meta={[areaLabel(s.area), USER_STATE_LABELS[s.state] ?? s.state, s.locked ? "잠김(선행 미완)" : ""]
                    .filter(Boolean)
                    .join(" · ")}
                  trailing={s.isNext ? <Badge tone="accent">다음 학습</Badge> : undefined}
                />
              ))}
            </List>
          </div>
        );
      })}
    </div>
  );
}
