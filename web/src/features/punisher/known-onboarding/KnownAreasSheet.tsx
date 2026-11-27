import { useEffect, useMemo, useState } from "react";
import { Button, Chip, ErrorState, ListSkeleton, Sheet, useToast } from "@forgen-team/ui";
import { areaLabel, useBatchMarkKnown, useConcepts } from "@/src/entities/punisher";
import { PUNISHER_AREAS, describePunisherError, type Concept } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

interface AreaGroup {
  area: string;
  /** 이미 앎·완료가 아닌 기초(깊이 1) 개념 id */
  pendingIds: string[];
}

function isLearned(c: Concept): boolean {
  return c.userState === "completed" || c.userState === "known";
}

/** 영역별 기초(깊이 1) 개념 — 영역 순서는 PUNISHER_AREAS, 그 밖 영역은 뒤에. 표시할 개념이 없는 영역은 뺀다. */
function groupDepthOne(concepts: Concept[]): AreaGroup[] {
  const byArea = new Map<string, Concept[]>();
  for (const c of concepts) {
    if (c.depth !== 1 || c.status !== "active") { continue; }
    byArea.set(c.area, [...(byArea.get(c.area) ?? []), c]);
  }
  const known: string[] = [...PUNISHER_AREAS];
  const order = [...known, ...[...byArea.keys()].filter((a) => !known.includes(a))];
  return order.flatMap((area) => {
    const list = byArea.get(area);
    return list && list.length > 0 ? [{ area, pendingIds: list.filter((c) => !isLearned(c)).map((c) => c.id) }] : [];
  });
}

interface KnownAreasSheetProps {
  open: boolean;
  onClose: () => void;
  /** 전부 저장에 성공했을 때 */
  onSaved?: () => void;
}

/** "이미 아는 개념 표시" — 영역 칩을 여러 개 골라 그 영역의 기초 개념을 한 번에 이미 앎으로 저장한다. */
export function KnownAreasSheet({ open, onClose, onSaved }: KnownAreasSheetProps) {
  const t = useT(messages);
  const toast = useToast();
  const concepts = useConcepts();
  const groups = useMemo(() => groupDepthOne(concepts.data ?? []), [concepts.data]);
  const [areas, setAreas] = useState<Set<string>>(() => new Set());
  const [failedCount, setFailedCount] = useState(0);
  const { run, running, progress } = useBatchMarkKnown();

  useEffect(() => {
    if (!open) { return; }
    setAreas(new Set());
    setFailedCount(0);
  }, [open]);

  const selectedIds = useMemo(
    () => groups.filter((g) => areas.has(g.area)).flatMap((g) => g.pendingIds),
    [groups, areas],
  );

  function toggleArea(area: string, on: boolean) {
    setAreas((prev) => {
      const next = new Set(prev);
      if (on) { next.add(area); } else { next.delete(area); }
      return next;
    });
  }

  async function save() {
    if (selectedIds.length === 0 || running) { return; }
    const result = await run(selectedIds);
    if (result.failedIds.length === 0) {
      toast.show(t.savedToast(result.total));
      onSaved?.();
      onClose();
      return;
    }
    // 실패한 개념이 든 영역만 남겨 다시 시도하게 한다.
    const failed = new Set(result.failedIds);
    setFailedCount(result.failedIds.length);
    setAreas(new Set(groups.filter((g) => g.pendingIds.some((id) => failed.has(id))).map((g) => g.area)));
  }

  const footer = (
    <Button variant="primary" onClick={() => void save()} loading={running} disabled={selectedIds.length === 0}>
      {running && progress ? t.saving(progress.done, progress.total) : t.markKnown}
    </Button>
  );

  return (
    <Sheet
      open={open}
      onOpenChange={(o) => { if (!o && !running) { onClose(); } }}
      title={t.sheetTitle}
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
          <ListSkeleton rows={3} />
        </div>
      ) : concepts.error ? (
        <ErrorState title={describePunisherError(concepts.error)} onRetry={() => { void concepts.refetch(); }} />
      ) : (
        <div role="group" aria-label={t.areasLabel} className="flex flex-wrap gap-2">
          {groups.map((g) => (
            <Chip
              key={g.area}
              selected={areas.has(g.area)}
              onSelectedChange={(on) => toggleArea(g.area, on)}
              disabled={running || g.pendingIds.length === 0}
            >
              {g.pendingIds.length === 0 ? `${areaLabel(g.area)} · ${t.allMarked}` : areaLabel(g.area)}
            </Chip>
          ))}
        </div>
      )}
    </Sheet>
  );
}
