
import { usePathname, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { useCallback } from "react";
import { ErrorState, ListSkeleton, useIsSplitLayout } from "@/src/shared/ui/kit";
import { usePrompts } from "@/src/entities/punisher";
import { PromptBlockEditor, PromptBlockList } from "@/src/features/punisher/prompt-edit";
import { describePunisherError, type FeedbackHints } from "@/src/shared/api/punisher";

/**
 * 학습 설정 > 강의 프롬프트. 모바일: 단계별 목록 → 블록 상세. 폴드 펼침(600px+): 목록 | 상세 2단.
 * URL 상태: ?block=<id>&from=<피드백 종류>(강의 피드백에서 온 제안 문구를 초안에 붙여 연다 — 저장은 사용자가).
 */
export function PromptSettingsWidget() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const split = useIsSplitLayout();
  const prompts = usePrompts();
  const blockParam = params.get("block");
  const from = params.get("from") as keyof FeedbackHints | null;

  const select = useCallback(
    (id: string | null) => router.replace(id ? `${pathname}?block=${encodeURIComponent(id)}` : pathname, { scroll: false }),
    [router, pathname],
  );

  if (prompts.isLoading) { return <ListSkeleton rows={8} />; }
  if (prompts.error || !prompts.data) {
    return <ErrorState message={describePunisherError(prompts.error)} onRetry={() => prompts.refetch()} retrying={prompts.isRefetching} />;
  }

  const blocks = prompts.data.blocks;
  const selected = blocks.find((b) => b.id === blockParam) ?? (split ? blocks.find((b) => !b.advanced) : undefined);
  const suggestion = selected && from && selected.id === blockParam ? prompts.data.feedbackHints[from] : undefined;
  const editor = selected ? (
    <PromptBlockEditor key={selected.id} block={selected} suggestion={suggestion} onBack={split ? undefined : () => select(null)} />
  ) : null;

  const intro = (
    <p className="mb-4 rounded-md bg-canvas-subtle px-3 py-2 text-sm text-fg-secondary">
      강의를 만드는 지시문을 직접 고칠 수 있어요. 바꾼 내용은 앞으로 만드는 강의(추천 강의 미리 만들기 포함)·질문 답변·구절 설명에 적용되고, 이미 만든 강의는 그대로예요.
    </p>
  );

  if (!split) {
    return editor ?? (
      <>
        {intro}
        <PromptBlockList blocks={blocks} selectedId={null} onSelect={select} />
      </>
    );
  }
  return (
    <>
      {intro}
      <div className="grid gap-6 fold:grid-cols-[17rem_minmax(0,1fr)]">
        <nav aria-label="프롬프트 블록" className="fold:sticky fold:top-[calc(var(--app-header-offset)+1rem)] fold:self-start">
          <PromptBlockList blocks={blocks} selectedId={selected?.id ?? null} onSelect={select} />
        </nav>
        <div className="min-w-0">{editor}</div>
      </div>
    </>
  );
}
