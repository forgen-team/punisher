
import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { Input, ListRow, Menu, confirmAction, type MenuItem } from "@/src/shared/ui/kit";
import { GOAL_STATUS_LABELS, useDeleteGoal, useUpdateGoal } from "@/src/entities/punisher";
import type { Goal, GoalStatus } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { GoalPathView } from "./GoalPathView";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

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
  const t = useT(messages);

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
    { key: "map", label: t.showOnMap, onSelect: () => onShowPath(goal.id) },
    { key: "rename", label: t.rename, onSelect: () => setEditing(true) },
    ...STATUSES.filter((s) => s !== goal.status).map((s) => ({
      key: `status-${s}`,
      label: t.changeTo(GOAL_STATUS_LABELS[s]),
      disabled: update.isPending,
      onSelect: () => update.mutate({ id: goal.id, input: { status: s } }),
    })),
    {
      key: "delete",
      label: t.delete,
      danger: true,
      disabled: remove.isPending,
      separatorBefore: true,
      // 삭제는 되돌릴 수 없어 앱 안 확인 창으로 묻는다(기본 포커스 취소).
      onSelect: () => {
        void confirmAction({ title: t.confirmDelete(goal.title), confirmLabel: t.delete, danger: true }).then((ok) => {
          if (ok) { remove.mutate(goal.id); }
        });
      },
    },
  ];

  return (
    <li className="border-b border-line last:border-b-0">
      {editing ? (
        <div className="py-2">
          <Input
            aria-label={t.goalName}
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
          meta={t.goalMeta(GOAL_STATUS_LABELS[goal.status], goal.targetConceptIds.length)}
          trailing={<Menu items={menuItems} label={t.goalMenu(goal.title)} />}
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
