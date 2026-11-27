
import { EmptyState, ErrorState, ListSkeleton, Section } from "@forgen-team/ui";
import { useGoals } from "@/src/entities/punisher";
import { describePunisherError } from "@/src/shared/api/punisher";
import { GoalForm } from "./GoalForm";
import { GoalItem } from "./GoalItem";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

interface GoalsManagerProps {
  onShowPath: (goalId: string, conceptId?: string) => void;
}

/** 목표 관리(시트 본문): 내 목표 목록 + 목표 추가. 지도 화면의 `?panel=goals` 시트에서 쓴다. */
export function GoalsManager({ onShowPath }: GoalsManagerProps) {
  const goals = useGoals();
  const t = useT(messages);
  const list = goals.data ?? [];

  return (
    <div className="flex flex-col gap-6">
      <Section title={t.myGoals}>
        {goals.isLoading ? (
          <div aria-busy="true">
            <ListSkeleton rows={3} />
          </div>
        ) : goals.error ? (
          <ErrorState title={describePunisherError(goals.error)} onRetry={() => { void goals.refetch(); }} />
        ) : list.length === 0 ? (
          <EmptyState title={t.emptyTitle} />
        ) : (
          <ul role="list">
            {list.map((g) => (
              <GoalItem key={g.id} goal={g} onShowPath={onShowPath} />
            ))}
          </ul>
        )}
      </Section>
      <Section title={t.newGoal} className="border-t border-line pt-4">
        <GoalForm />
      </Section>
    </div>
  );
}
