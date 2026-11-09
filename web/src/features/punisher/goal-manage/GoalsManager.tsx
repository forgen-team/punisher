
import { Target } from "lucide-react";
import { EmptyState, ErrorState, ListSkeleton, Section } from "@/src/shared/ui/kit";
import { useGoals } from "@/src/entities/punisher";
import { describePunisherError } from "@/src/shared/api/punisher";
import { GoalForm } from "./GoalForm";
import { GoalItem } from "./GoalItem";

interface GoalsManagerProps {
  onShowPath: (goalId: string, conceptId?: string) => void;
}

/** 목표 관리(시트 본문): 내 목표 목록 + 새 목표. 지도 화면의 "목표 관리" 시트에서 쓴다. */
export function GoalsManager({ onShowPath }: GoalsManagerProps) {
  const goals = useGoals();
  const list = goals.data ?? [];

  return (
    <div className="flex flex-col gap-6">
      <Section title="내 목표" description="목표를 누르면 필요한 선행 사슬과 다음 학습이 펼쳐집니다.">
        {goals.isLoading ? (
          <div aria-busy="true">
            <ListSkeleton rows={3} />
          </div>
        ) : goals.error ? (
          <ErrorState compact message={describePunisherError(goals.error)} onRetry={() => goals.refetch()} />
        ) : list.length === 0 ? (
          <EmptyState compact icon={<Target />} title="아직 목표가 없습니다" description="아래에서 첫 목표를 추가해 보세요." />
        ) : (
          <ul role="list">
            {list.map((g) => (
              <GoalItem key={g.id} goal={g} onShowPath={onShowPath} />
            ))}
          </ul>
        )}
      </Section>
      <Section title="새 목표" className="border-t border-line pt-4">
        <GoalForm />
      </Section>
    </div>
  );
}
