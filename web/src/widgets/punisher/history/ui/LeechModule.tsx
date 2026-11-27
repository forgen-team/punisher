import { ErrorState, List, ListRow, Menu, Section, Skeleton, useToast, type MenuItemDef } from "@forgen-team/ui";
import { useLeechAction, useLeeches } from "@/src/entities/punisher";
import { useLaunchInstant } from "@/src/features/punisher/instant-start";
import { describePunisherError, type Leech } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/**
 * 자꾸 막히는 개념(반복 실패) — 행 ⋯ 에 다르게 설명한 강의 · 앞 개념부터 · 잠시 보류(되돌리기) · 그대로 복습.
 * 없으면 모듈을 그리지 않는다(좋은 소식). 불러오지 못한 것은 숨기지 않는다.
 */
export function LeechModule() {
  const t = useT(messages);
  const leeches = useLeeches();
  const action = useLeechAction();
  const toast = useToast();
  const { launch, error } = useLaunchInstant();

  if (leeches.error) {
    return (
      <Section variant="card" title={t.leechTitle}>
        <ErrorState title={describePunisherError(leeches.error)} onRetry={() => void leeches.refetch()} />
      </Section>
    );
  }
  if (leeches.isLoading) { return <Section variant="card" title={t.leechTitle}><Skeleton lines={2} /></Section>; }
  const list = leeches.data?.leeches ?? [];
  if (list.length === 0) { return null; }

  async function relearn(l: Leech) {
    try {
      await launch({ conceptId: l.conceptId, query: t.leechQuery(l.objectiveText) });
    } catch (err) {
      // 실패 문구는 위 InlineError(error) 가 보인다.
      console.warn("punisher: relearn launch failed", err);
    }
  }

  function suspend(l: Leech) {
    action.mutate({ objectiveId: l.objectiveId, action: "suspend" }, {
      onSuccess: () =>
        toast.undo(t.leechSuspended, () => {
          action.mutateAsync({ objectiveId: l.objectiveId, action: "resume" }).catch(() => toast.show(t.undoFailed, { tone: "danger" }));
        }),
    });
  }

  function menuOf(l: Leech): MenuItemDef[] {
    return [
      { key: "relearn", label: t.leechRelearn, onSelect: () => void relearn(l) },
      { key: "prereq", label: t.leechPrereq, href: `/map?concept=${encodeURIComponent(l.conceptId)}` },
      { key: "suspend", label: t.leechSuspend, onSelect: () => suspend(l) },
      { key: "keep", label: t.leechKeep, onSelect: () => action.mutate({ objectiveId: l.objectiveId, action: "keep" }) },
    ];
  }

  return (
    <Section variant="card" title={t.leechTitle}>
      <InlineError error={error ?? action.error} />
      <List>
        {list.map((l) => (
          <ListRow
            key={l.objectiveId}
            title={l.objectiveText}
            meta={t.leechMeta(l.conceptName, l.failCount)}
            trailing={<Menu items={menuOf(l)} label={t.leechActions(l.objectiveText.slice(0, 30))} />}
          />
        ))}
      </List>
    </Section>
  );
}
