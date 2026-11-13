
import { Link } from "@/src/shared/lib/router";
import { forwardRef, useState } from "react";
import { Button, Section, buttonVariants } from "@/src/shared/ui/kit";
import { useBatchMarkKnown } from "@/src/features/punisher/known-onboarding";
import type { PlacementResult as Placement } from "@/src/shared/api/punisher";
import { PLACEMENT_LABELS } from "./constants";

export function chapterMapHref(chapterId: string): string {
  return `/map?chapter=${encodeURIComponent(chapterId)}`;
}

/** 배치 진단 결과 — 맞힌 개념을 이미 앎 후보로 보이고, 고른 것만 확정하면 '이미 앎'으로 표시한다. */
export const PlacementResult = forwardRef<HTMLElement, { placement: Placement }>(function PlacementResult({ placement }, ref) {
  const [picked, setPicked] = useState<ReadonlySet<string>>(() => new Set(placement.candidates.map((c) => c.conceptId)));
  const { run, running, progress } = useBatchMarkKnown({ queueVerify: false });
  const [done, setDone] = useState(false);
  const failed = progress?.failedIds.length ?? 0;

  function toggle(id: string) {
    setPicked((prev) => {
      const next = new Set(prev);
      if (next.has(id)) { next.delete(id); } else { next.add(id); }
      return next;
    });
  }

  async function confirm() {
    const result = await run([...picked]);
    setDone(result.failedIds.length === 0);
  }

  return (
    <Section ref={ref} tabIndex={-1} title={PLACEMENT_LABELS.candidates} className="mt-8 border-t border-line pt-4 outline-none">
      {placement.candidates.length === 0 ? (
        <p className="py-3 text-sm text-fg-secondary">{PLACEMENT_LABELS.none}</p>
      ) : (
        <ul className="flex flex-col divide-y divide-line">
          {placement.candidates.map((c) => (
            <li key={c.conceptId}>
              <label className="flex min-h-touch cursor-pointer items-center gap-3 py-2 text-sm font-medium text-fg">
                <input
                  type="checkbox"
                  className="size-5 shrink-0 accent-accent-solid"
                  checked={picked.has(c.conceptId)}
                  disabled={running || done}
                  onChange={() => toggle(c.conceptId)}
                />
                {c.conceptName}
              </label>
            </li>
          ))}
        </ul>
      )}
      <div className="mt-4 flex flex-wrap items-center gap-2">
        {placement.candidates.length > 0 && (
          <Button onClick={confirm} loading={running} disabled={done || picked.size === 0}>
            {done ? PLACEMENT_LABELS.confirmed : PLACEMENT_LABELS.confirm}
          </Button>
        )}
        {placement.chapterId && (
          <Link href={chapterMapHref(placement.chapterId)} className={buttonVariants({ variant: "secondary" })}>
            {PLACEMENT_LABELS.map}
          </Link>
        )}
      </div>
      {failed > 0 && (
        <p role="alert" className="mt-2 text-sm text-danger">
          {PLACEMENT_LABELS.failed(failed)}
        </p>
      )}
    </Section>
  );
});
