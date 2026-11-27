
import { Badge, ErrorState, List, ListRow, ListSkeleton } from "@forgen-team/ui";
import { USER_STATE_LABELS, areaLabel, useGoalPath } from "@/src/entities/punisher";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

interface GoalPathViewProps {
  goalId: string;
  /** 단계 선택 시: 지도에서 이 목표 경로 + 개념 상세를 연다 */
  onSelectStep: (conceptId: string) => void;
}

/** 목표별 선행 사슬. 다음 학습 단계는 선택 막대 + 배지 1개, 잠김은 meta 텍스트로. */
export function GoalPathView({ goalId, onSelectStep }: GoalPathViewProps) {
  const { data, isLoading, error, refetch } = useGoalPath(goalId);
  const t = useT(messages);
  if (isLoading) { return <ListSkeleton rows={3} />; }
  if (error) {
    return <ErrorState title={describePunisherError(error)} onRetry={() => { void refetch(); }} />;
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
              {path.completed && <span className="ml-2 font-normal text-success">{t.pathCompleted}</span>}
            </p>
            <List>
              {steps.map((s, index) => (
                <ListRow
                  key={s.conceptId}
                  onClick={() => onSelectStep(s.conceptId)}
                  selected={s.isNext}
                  title={`${index + 1}. ${s.name}`}
                  meta={[areaLabel(s.area), USER_STATE_LABELS[s.state] ?? s.state, s.locked ? t.locked : "", s.dismissed ? t.dismissed : ""]
                    .filter(Boolean)
                    .join(" · ")}
                  trailing={s.isNext ? <Badge tone="brand" variant="weak">{t.nextUp}</Badge> : undefined}
                />
              ))}
            </List>
          </div>
        );
      })}
    </div>
  );
}
