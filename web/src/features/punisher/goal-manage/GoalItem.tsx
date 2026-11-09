
import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { Input, ListRow, Menu, type MenuItem } from "@/src/shared/ui/kit";
import { GOAL_STATUS_LABELS, useDeleteGoal, useUpdateGoal } from "@/src/entities/punisher";
import type { Goal, GoalStatus } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { GoalPathView } from "./GoalPathView";

const STATUSES: GoalStatus[] = ["active", "achieved", "archived"];

interface GoalItemProps {
  goal: Goal;
  /** 지도에서 이 목표의 경로를 보기 (선택한 단계가 있으면 그 개념 상세도 연다) */
  onShowPath: (goalId: string, conceptId?: string) => void;
}

/** 목표 1행: 눌러서 경로 펼침, ⋯ 메뉴에 이름 수정·상태·지도 보기·삭제. */
export function GoalItem({ goal, onShowPath }: GoalItemProps) {
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(goal.title);
  const update = useUpdateGoal();
  const remove = useDeleteGoal();

  function saveTitle() {
    const next = title.trim();
    if (!next || next === goal.title) {
      setTitle(goal.title);
      setEditing(false);
      return;
    }
    update.mutate({ id: goal.id, input: { title: next } }, { onSuccess: () => setEditing(false) });
  }

  const menuItems: MenuItem[] = [
    { key: "map", label: "지도에서 경로 보기", onSelect: () => onShowPath(goal.id) },
    { key: "rename", label: "이름 수정", onSelect: () => setEditing(true) },
    ...STATUSES.filter((s) => s !== goal.status).map((s) => ({
      key: `status-${s}`,
      label: `${GOAL_STATUS_LABELS[s]}(으)로 변경`,
      disabled: update.isPending,
      onSelect: () => update.mutate({ id: goal.id, input: { status: s } }),
    })),
    {
      key: "delete",
      label: "삭제",
      danger: true,
      disabled: remove.isPending,
      onSelect: () => {
        if (window.confirm(`'${goal.title}' 목표를 삭제할까요?`)) { remove.mutate(goal.id); }
      },
    },
  ];

  return (
    <li className="border-b border-line last:border-b-0">
      {editing ? (
        <div className="py-2">
          <Input
            aria-label="목표 이름"
            value={title}
            maxLength={200}
            autoFocus
            onChange={(event) => setTitle(event.target.value)}
            onBlur={saveTitle}
            onKeyDown={(event) => {
              if (event.key === "Enter") { saveTitle(); }
              if (event.key === "Escape") {
                event.stopPropagation();
                setTitle(goal.title);
                setEditing(false);
              }
            }}
          />
        </div>
      ) : (
        <ListRow
          className="border-b-0"
          onClick={() => setOpen((v) => !v)}
          leading={open ? <ChevronDown aria-hidden="true" /> : <ChevronRight aria-hidden="true" />}
          title={goal.title}
          meta={`${GOAL_STATUS_LABELS[goal.status]} · 목표 개념 ${goal.targetConceptIds.length}개`}
          trailing={<Menu items={menuItems} label={`${goal.title} 목표 메뉴`} />}
        />
      )}
      <InlineError error={update.error ?? remove.error} />
      {open && !editing && (
        <div className="pb-3">
          <GoalPathView goalId={goal.id} onSelectStep={(conceptId) => onShowPath(goal.id, conceptId)} />
        </div>
      )}
    </li>
  );
}
