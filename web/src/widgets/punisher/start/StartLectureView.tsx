import { Link } from "@/src/shared/lib/router";
import { useEffect, useRef, useState } from "react";
import { Button, EmptyState, Skeleton } from "@forgen-team/ui";
import { ResolutionChoices, useLaunchInstant } from "@/src/features/punisher/instant-start";
import type { InstantResponse } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/**
 * 개념으로 강의 시작 — 기존 즉석 강의 흐름(useLaunchInstant)을 그대로 쓴다: 이미 있는 강의면 그 강의로, 아니면 생성 화면으로 이동.
 * 선택이 필요하거나 이미 배운 개념이면 그 결과를 여기서 보여준다.
 */
export function StartLectureView({ conceptId, query }: { conceptId: string; query: string }) {
  const { launch, isPending, error } = useLaunchInstant();
  const [result, setResult] = useState<InstantResponse | null>(null);
  const started = useRef(false);
  const t = useT(messages);

  useEffect(() => {
    if (!conceptId || started.current) { return; }
    started.current = true;
    launch({ conceptId, query: query || undefined }).then(setResult, () => undefined);
  }, [conceptId, launch, query]);

  if (!conceptId) {
    return <EmptyState title={t.emptyTitle} action={<Button asChild variant="secondary" size="sm"><Link href="/">{t.backToLearn}</Link></Button>} />;
  }
  return (
    <div className="flex flex-col gap-4 pt-4" aria-live="polite">
      {(isPending || (!error && !result)) && (
        <>
          <p className="text-sm text-fg-secondary">{t.opening}</p>
          <Skeleton lines={4} />
        </>
      )}
      <InlineError error={error} />
      {result && <ResolutionChoices response={result} disabled={isPending} onPick={(id) => void launch({ conceptId: id, query: query || undefined }).then(setResult, () => undefined)} />}
    </div>
  );
}
