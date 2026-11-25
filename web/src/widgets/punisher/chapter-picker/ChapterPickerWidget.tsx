
import { useRouter } from "@/src/shared/lib/router";
import { useMemo, useState } from "react";
import { Sparkles } from "lucide-react";
import { Button, ErrorState, Section, Textarea } from "@/src/shared/ui/kit";
import { useApplyChapterDraft, useChapterSuggestions, useCreateChapterDraft, useGraph, useStartDiagnostic } from "@/src/entities/punisher";
import { checkHref } from "@/src/features/punisher/check-start";
import { PunisherApiError, describePunisherError, type ChapterDraftResponse } from "@/src/shared/api/punisher";
import { useDraftState } from "@/src/shared/lib/drafts";
import { isWitems } from "@/src/shared/config/runtime";
import { DraftPreview } from "./ui/DraftPreview";
import { PlacementOffer } from "./ui/PlacementOffer";
import { SuggestionCards, SuggestionCardsSkeleton } from "./ui/SuggestionCards";
import { FREE_TEXT_MAX, MAX_PICKS } from "./ui/constants";
import { useLocale, translate } from "@/src/shared/i18n";
import { messages } from "./i18n";
import { pickSummary } from "./lib/pickSummary";

/** 422 CHAPTER_INVALID 이면 서버가 준 문제 목록을 함께 보여 준다. */
function designProblems(error: unknown): string[] {
  return error instanceof PunisherApiError && error.status === 422 ? error.problems : [];
}

interface PickerDraft {
  selected: string[];
  freeText: string;
  /** 설계해 본 초안(미리보기) — 다른 화면에 갔다 와도 그대로 본다 */
  preview: ChapterDraftResponse | null;
}

const EMPTY_DRAFT: PickerDraft = { selected: [], freeText: "", preview: null };

/**
 * 고른 제안·직접 입력·설계 미리보기는 이 기기에 보관한다(점검 17) — 반영('이대로 열기')하면 지운다.
 * 다음 챕터 고르기(질문형): 자동 제안 카드(여러 개) + 직접 입력 → 설계(미리보기) → '이대로 열기'.
 * 생성은 Claude 게이트웨이로 하고, 반영 전까지 지도는 바뀌지 않는다.
 */
export function ChapterPickerWidget() {
  const router = useRouter();
  const locale = useLocale();
  const t = translate(messages, locale);
  const suggestions = useChapterSuggestions();
  const graph = useGraph();
  const create = useCreateChapterDraft();
  const apply = useApplyChapterDraft();
  const draft = useDraftState<PickerDraft>("chapter-picker", EMPTY_DRAFT, "", (v) => v.selected.length === 0 && !v.freeText.trim() && !v.preview);
  const setDraft = draft.setValue;
  const selected = useMemo(() => new Set(draft.value.selected), [draft.value.selected]);
  const freeText = draft.value.freeText;
  const setFreeText = (text: string) => setDraft((v) => ({ ...v, freeText: text }));
  const preview = create.data ?? draft.value.preview;
  const [openedChapterId, setOpenedChapterId] = useState("");
  const placement = useStartDiagnostic();

  const names = useMemo(() => new Map((graph.data?.nodes ?? []).map((n) => [n.id, n.name])), [graph.data]);

  function toggle(id: string) {
    setDraft((v) => {
      const next = new Set(v.selected);
      if (next.has(id)) { next.delete(id); } else if (next.size < MAX_PICKS) { next.add(id); }
      return { ...v, selected: [...next] };
    });
  }

  function design() {
    create.mutate(
      { suggestionIds: [...selected], freeText: freeText.trim() },
      { onSuccess: (res) => setDraft((v) => ({ ...v, preview: res })) },
    );
  }

  function openChapter() {
    if (!preview) { return; }
    apply.mutate(preview.draftId, {
      onSuccess: (res) => {
        draft.discard();
        setOpenedChapterId(res.chapterId);
      },
    });
  }

  function goToMap() {
    router.push(`/map?chapter=${encodeURIComponent(openedChapterId)}`);
  }

  function startPlacement() {
    placement.mutate(
      { kind: "placement", chapterId: openedChapterId },
      { onSuccess: (detail) => { if (detail) { router.push(checkHref(detail.session.id)); } } },
    );
  }

  if (openedChapterId) {
    return (
      <div className="flex flex-col gap-3 pt-4 lg:pt-6">
        <PlacementOffer starting={placement.isPending} error={placement.error} onStart={startPlacement} onSkip={goToMap} />
      </div>
    );
  }

  if (preview) {
    return (
      <div className="flex flex-col gap-3">
        <DraftPreview
          draft={preview.draft}
          names={names}
          applying={apply.isPending}
          onApply={openChapter}
          onBack={() => {
            create.reset();
            apply.reset();
            setDraft((v) => ({ ...v, preview: null }));
          }}
        />
        {apply.error && (
          <p role="alert" className="text-sm text-danger">
            {describePunisherError(apply.error)}
          </p>
        )}
      </div>
    );
  }

  const QuestionHeading = isWitems() ? "h2" : "h1";
  const canDesign = (selected.size > 0 || freeText.trim() !== "") && !create.isPending;
  const problems = designProblems(create.error);

  return (
    <div className="flex flex-col gap-5">
      <header className="pt-4 lg:pt-6">
        {/* Witems: h1 은 셸 화면 제목('다음 챕터 고르기') — 이 질문은 그 아래 h2. */}
        <QuestionHeading className="text-xl font-semibold text-fg">{suggestions.data?.question ?? t.defaultQuestion}</QuestionHeading>
      </header>

      {/* xl: 추천 카드 | 직접 입력(오른쪽 열 sticky). 실행은 아래 고정 바 하나 — 어느 쪽을 골라도 '선택 n개 · 설계해 보기'가 늘 보인다. */}
      <div className="flex flex-col gap-5 xl:grid xl:grid-cols-[minmax(0,1fr)_22rem] xl:items-start xl:gap-6">
        <Section title={t.suggestions} description={t.maxPicks(MAX_PICKS)}>
          {suggestions.isLoading ? (
            <SuggestionCardsSkeleton />
          ) : suggestions.error ? (
            <ErrorState compact message={describePunisherError(suggestions.error)} onRetry={() => suggestions.refetch()} />
          ) : (
            <SuggestionCards suggestions={suggestions.data?.suggestions ?? []} selected={selected} onToggle={toggle} />
          )}
        </Section>

        <Section
          title={t.freeText}
          description={t.charCount(freeText.length, FREE_TEXT_MAX)}
          className="xl:sticky xl:top-[calc(var(--app-header-offset)+1.5rem)]"
        >
          {/* 좁은 폭과 같은 구성(상자 없이 입력란 + 오류) — 넓은 폭에서만 카드로 감싸면 두 폭의 모양이 달라진다. */}
          <div className="flex flex-col gap-5">
            <Textarea
              aria-label={t.freeText}
              value={freeText}
              maxLength={FREE_TEXT_MAX}
              rows={3}
              placeholder={t.freeTextPlaceholder}
              onChange={(event) => setFreeText(event.target.value)}
            />

            <div aria-live="polite" className="empty:hidden">
              {create.error && (
                <div role="alert" className="rounded-md bg-danger-subtle px-3 py-2 text-sm text-danger">
                  <p>{describePunisherError(create.error)}</p>
                  {problems.length > 0 && (
                    <ul className="mt-1 list-disc pl-5 text-xs">
                      {problems.slice(0, 5).map((p) => (
                        <li key={p}>{p}</li>
                      ))}
                    </ul>
                  )}
                </div>
              )}
            </div>
          </div>
        </Section>
      </div>

      <div className="sticky bottom-[var(--app-bottom-offset)] z-nav -mx-4 flex items-center justify-end gap-3 border-t border-line bg-canvas px-4 py-3 fold:-mx-6 fold:px-6 lg:mx-0 lg:px-0">
        <p aria-live="polite" className="min-w-0 flex-1 truncate text-sm text-fg-secondary">
          {pickSummary(selected.size, freeText, locale)}
        </p>
        <Button onClick={design} disabled={!canDesign} loading={create.isPending}>
          <Sparkles className="size-4" aria-hidden="true" />
          {create.isPending ? t.designing : t.design}
        </Button>
      </div>
    </div>
  );
}
