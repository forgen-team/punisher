
import { useEffect, useMemo, useState } from "react";
import { Button, ErrorState, ListSkeleton, Section, Sheet } from "@forgen-team/ui";
import { areaLabel, useBatchMarkKnown, useConcepts } from "@/src/entities/punisher";
import { PUNISHER_AREAS, describePunisherError, type Concept } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

interface AreaGroup {
  area: string;
  concepts: Concept[];
}

function groupDepthOne(concepts: Concept[]): AreaGroup[] {
  const byArea = new Map<string, Concept[]>();
  for (const c of concepts) {
    if (c.depth !== 1 || c.status !== "active") { continue; }
    byArea.set(c.area, [...(byArea.get(c.area) ?? []), c]);
  }
  const order: string[] = [...PUNISHER_AREAS, ...[...byArea.keys()].filter((a) => !(PUNISHER_AREAS as readonly string[]).includes(a))];
  return order.flatMap((area) => {
    const list = byArea.get(area);
    return list && list.length > 0 ? [{ area, concepts: list }] : [];
  });
}

function isLearned(c: Concept): boolean {
  return c.userState === "completed" || c.userState === "known";
}

interface KnownAreasSheetProps {
  open: boolean;
  onClose: () => void;
  /** 전부 저장에 성공했을 때 (안내 닫기 기억 등) */
  onSaved: () => void;
}

/** "이미 아는 영역 표시" — 영역별 깊이 1 개념을 체크해 일괄로 이미 앎 처리. */
export function KnownAreasSheet({ open, onClose, onSaved }: KnownAreasSheetProps) {
  const t = useT(messages);
  const concepts = useConcepts();
  const groups = useMemo(() => groupDepthOne(concepts.data ?? []), [concepts.data]);
  const [selected, setSelected] = useState<Set<string>>(() => new Set());
  const [failedCount, setFailedCount] = useState(0);
  const { run, running, progress } = useBatchMarkKnown();

  useEffect(() => {
    if (!open) { return; }
    setSelected(new Set());
    setFailedCount(0);
  }, [open]);

  function toggle(id: string, checked: boolean) {
    setSelected((prev) => {
      if (prev.has(id) === checked) { return prev; }
      const next = new Set(prev);
      if (checked) { next.add(id); } else { next.delete(id); }
      return next;
    });
  }

  function toggleArea(group: AreaGroup, checked: boolean) {
    setSelected((prev) => {
      const next = new Set(prev);
      for (const c of group.concepts) {
        if (isLearned(c)) { continue; }
        if (checked) { next.add(c.id); } else { next.delete(c.id); }
      }
      return next;
    });
  }

  async function save() {
    if (selected.size === 0 || running) { return; }
    const result = await run([...selected]);
    if (result.failedIds.length === 0) {
      onSaved();
      onClose();
      return;
    }
    setFailedCount(result.failedIds.length);
    setSelected(new Set(result.failedIds));
  }

  const footer = (
    <>
      <Button variant="ghost" onClick={onClose} disabled={running}>
        {t.later}
      </Button>
      <Button variant="primary" onClick={save} loading={running} disabled={selected.size === 0}>
        {running && progress ? t.saving(progress.done, progress.total) : t.saveCount(selected.size)}
      </Button>
    </>
  );

  return (
    <Sheet
      open={open}
      onOpenChange={(o) => { if (!o && !running) { onClose(); } }}
      title={t.sheetTitle}
      description={t.sheetDescription}
      footer={footer}
      desktop="dialog"
    >
      <div aria-live="polite" className="text-sm">
        {failedCount > 0 && !running && (
          <p role="alert" className="mb-2 text-danger">
            {t.saveFailed(failedCount)}
          </p>
        )}
      </div>
      {concepts.isLoading ? (
        <div aria-busy="true">
          <ListSkeleton rows={6} />
        </div>
      ) : concepts.error ? (
        <ErrorState title={describePunisherError(concepts.error)} onRetry={() => { void concepts.refetch(); }} />
      ) : (
        <div className="flex flex-col gap-4">
          {groups.map((group) => {
            const pending = group.concepts.filter((c) => !isLearned(c));
            const allChecked = pending.length > 0 && pending.every((c) => selected.has(c.id));
            return (
              <Section
                key={group.area}
                headingLevel={3}
                title={areaLabel(group.area)}
                action={
                  pending.length > 0 ? (
                    <Button variant="ghost" size="sm" disabled={running} onClick={() => toggleArea(group, !allChecked)}>
                      {allChecked ? t.clearAll : t.markArea}
                    </Button>
                  ) : (
                    <span className="text-xs text-fg-muted">{t.allMarked}</span>
                  )
                }
              >
                <ul role="list" className="border-y border-line">
                  {group.concepts.map((c) => {
                    const learned = isLearned(c);
                    return (
                      <li key={c.id} className="border-b border-line last:border-b-0">
                        <label className="flex min-h-touch cursor-pointer items-center gap-3 py-2">
                          <input
                            type="checkbox"
                            className="size-5 shrink-0 accent-accent-solid"
                            checked={learned || selected.has(c.id)}
                            disabled={learned || running}
                            onChange={(event) => toggle(c.id, event.target.checked)}
                          />
                          <span className="min-w-0 flex-1">
                            <span className="block truncate text-sm font-medium text-fg">{c.name}</span>
                            {(learned || c.summary) && (
                              <span className="block truncate text-xs text-fg-muted">
                                {learned ? t.alreadyMarked : c.summary}
                              </span>
                            )}
                          </span>
                        </label>
                      </li>
                    );
                  })}
                </ul>
              </Section>
            );
          })}
        </div>
      )}
    </Sheet>
  );
}
