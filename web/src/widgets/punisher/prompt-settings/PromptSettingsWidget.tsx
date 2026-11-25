
import { usePathname, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { useCallback, useRef } from "react";
import { ErrorState, ListSkeleton, useIsSplitLayout } from "@/src/shared/ui/kit";
import { usePrompts } from "@/src/entities/punisher";
import { PromptBlockEditor, PromptBlockList } from "@/src/features/punisher/prompt-edit";
import { describePunisherError, type FeedbackHints } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { editorOutOfView } from "./lib/editorScroll";
import { messages } from "./i18n";

/**
 * 학습 설정 > 강의 프롬프트. 모바일: 단계별 목록 → 블록 상세(뒤로가기는 셸 상단 바 하나 — 상세는 기록에 쌓아 연다).
 * 폴드 펼침(600px+): 목록 | 상세 2단 — 목록 아래쪽 항목을 고르면 편집기 머리로 스크롤한다.
 * URL 상태: ?block=<id>&from=<피드백 종류>(강의 피드백에서 온 제안 문구를 초안에 붙여 연다 — 저장은 사용자가).
 * 블록을 고르면 넓은 화면도 기록에 쌓는다(화면=주소). 고친 초안은 블록마다 이 기기에 보관된다(PromptBlockEditor).
 */
export function PromptSettingsWidget() {
  const t = useT(messages);
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const split = useIsSplitLayout();
  const prompts = usePrompts();
  const editorRef = useRef<HTMLDivElement>(null);
  const blockParam = params.get("block");
  const from = params.get("from") as keyof FeedbackHints | null;

  const select = useCallback(
    (id: string) => {
      const href = `${pathname}?block=${encodeURIComponent(id)}`;
      // 상세(블록)는 주소에 남긴다(push — 같은 블록이면 이동하지 않음). 고친 초안은 블록마다 보관되므로 묻지 않는다.
      router.push(href, { scroll: false, guard: false });
      if (!split) { return; }
      const el = editorRef.current;
      const headerOffset = Number.parseFloat(getComputedStyle(document.documentElement).getPropertyValue("--app-header-offset")) || 0;
      if (el && editorOutOfView(el.getBoundingClientRect().top, window.innerHeight, headerOffset)) {
        el.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    },
    [router, pathname, split],
  );

  if (prompts.isLoading) { return <ListSkeleton rows={8} />; }
  if (prompts.error || !prompts.data) {
    return <ErrorState message={describePunisherError(prompts.error)} onRetry={() => prompts.refetch()} retrying={prompts.isRefetching} />;
  }

  const blocks = prompts.data.blocks;
  const selected = blocks.find((b) => b.id === blockParam) ?? (split ? blocks.find((b) => !b.advanced) : undefined);
  const suggestion = selected && from && selected.id === blockParam ? prompts.data.feedbackHints[from] : undefined;
  const editor = selected ? <PromptBlockEditor key={selected.id} block={selected} suggestion={suggestion} /> : null;

  if (!split) {
    return editor ?? <PromptBlockList blocks={blocks} selectedId={null} onSelect={select} />;
  }
  return (
    <div className="grid gap-6 fold:grid-cols-[17rem_minmax(0,1fr)]">
      <nav aria-label={t.blocksNav} className="fold:sticky fold:top-[calc(var(--app-header-offset)+1rem)] fold:self-start">
        <PromptBlockList blocks={blocks} selectedId={selected?.id ?? null} onSelect={select} />
      </nav>
      <div ref={editorRef} className="min-w-0 scroll-mt-[calc(var(--app-header-offset)+1rem)]">
        {editor}
      </div>
    </div>
  );
}
