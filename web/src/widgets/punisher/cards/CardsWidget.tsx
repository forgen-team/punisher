import { useMemo, useState } from "react";
import { Link, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { Button, ChipLink, EmptyState, ErrorState, List, ListRow, ListSkeleton, Menu, buttonVariants, confirmAction, type MenuItem } from "@/src/shared/ui/kit";
import { lectureAnchorHref, useCards, useDeleteCard, useGradeCard } from "@/src/entities/punisher";
import { describePunisherError, type CardRating, type ReviewCard } from "@/src/shared/api/punisher";
import { useStudyShortcuts } from "@/src/shared/lib/shortcuts";
import { useOnline } from "@/src/shared/lib/useOnline";
import { formatDate, useLocale, useT } from "@/src/shared/i18n";
import { InlineError } from "@/src/shared/ui/punisher";
import { START, graded, reveal } from "./lib/session";
import { messages } from "./i18n";

type CardsTab = "review" | "all";

/**
 * 복습 카드(W5a) — ?tab=review(오늘 카드, 기본) · ?tab=all(모든 카드). 탭은 링크(화면=주소).
 * 오늘 카드: 질문 → (Space/Enter) 답 보기 → 1 다시 · 2 어려움 · 3 알맞음. 단축키는 설정에서 끌 수 있다.
 */
export function CardsWidget() {
  const t = useT(messages);
  const params = useSearchParams();
  const tab: CardsTab = params.get("tab") === "all" ? "all" : "review";
  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
      <nav aria-label={t.tabsLabel} className="flex gap-2">
        <ChipLink href="/cards" linkAs={Link} selected={tab === "review"}>{t.tabReview}</ChipLink>
        <ChipLink href="/cards?tab=all" linkAs={Link} selected={tab === "all"}>{t.tabAll}</ChipLink>
      </nav>
      {tab === "review" ? <CardReview /> : <CardList />}
    </div>
  );
}

function CardReview() {
  const t = useT(messages);
  const router = useRouter();
  const cards = useCards(true);
  const grade = useGradeCard();
  const online = useOnline();
  // 시작할 때의 목록으로 한 판을 돈다(채점으로 목록이 다시 읽혀도 순서가 흔들리지 않게).
  const [deck, setDeck] = useState<ReviewCard[] | null>(null);
  const list = deck ?? cards.data ?? null;
  const [s, setS] = useState(START);
  const card = list?.[s.index];

  function show() {
    if (!deck && cards.data) { setDeck(cards.data); }
    setS(reveal);
  }
  function rate(rating: CardRating) {
    if (!card || !s.revealed || grade.isPending || !online) { return; }
    if (!deck && cards.data) { setDeck(cards.data); }
    grade.mutate({ id: card.id, rating }, { onSuccess: () => setS(graded) });
  }
  const keys = useStudyShortcuts({
    " ": card ? show : undefined,
    Enter: card ? (s.revealed ? undefined : show) : () => router.push("/"),
    "1": () => rate(1),
    "2": () => rate(2),
    "3": () => rate(3),
  }, Boolean(list));

  if (cards.isLoading) { return <ListSkeleton rows={3} />; }
  if (cards.error && !list) {
    return <ErrorState message={describePunisherError(cards.error)} onRetry={() => cards.refetch()} retrying={cards.isRefetching} />;
  }
  if (!list || list.length === 0) {
    return (
      <EmptyState
        title={t.emptyTitle}
        description={t.emptyBody}
        action={<Link href="/" className={buttonVariants({ variant: "secondary", size: "sm" })}>{t.toToday}</Link>}
      />
    );
  }
  if (!card) {
    return (
      <EmptyState
        title={t.doneTitle}
        description={t.doneBody(s.graded)}
        action={<Link href="/" className={buttonVariants({ variant: "secondary", size: "sm" })}>{t.toToday}</Link>}
      />
    );
  }

  const ratings: Array<[CardRating, string]> = [[1, t.again], [2, t.hard], [3, t.good]];
  return (
    <section aria-label={t.title} className="flex flex-col gap-4">
      <div className="flex items-center justify-between text-sm text-fg-secondary">
        <span className="tabular-nums">{t.progress(s.index + 1, list.length)}</span>
        {keys && <span className="hidden text-xs text-fg-muted md:inline">{t.shortcutsHint}</span>}
      </div>
      {!online && <p role="status" className="rounded-md bg-warning-subtle px-3 py-2 text-sm text-warning">{t.offline}</p>}
      <article className="flex min-h-48 flex-col gap-4 rounded-xl border border-line bg-canvas-elevated p-5 shadow-xs">
        <p className="whitespace-pre-wrap break-keep text-lg font-medium text-fg">{card.question}</p>
        {s.revealed ? (
          <div className="border-t border-line pt-4">
            <p className="whitespace-pre-wrap break-keep text-base text-fg-secondary">{card.answer}</p>
            {card.lectureId && (
              <Link href={lectureAnchorHref(card.lectureId, card.anchor)} className="mt-3 inline-block text-sm text-accent-text underline-offset-2 hover:underline">
                {t.fromLecture}
              </Link>
            )}
          </div>
        ) : null}
      </article>
      <InlineError error={grade.error} />
      {s.revealed ? (
        <div role="group" aria-label={t.title} className="grid grid-cols-3 gap-2">
          {ratings.map(([rating, label]) => (
            <Button key={rating} variant={rating === 3 ? "primary" : "secondary"} disabled={!online || grade.isPending}
              loading={grade.isPending && grade.variables?.rating === rating} onClick={() => rate(rating)}
              aria-keyshortcuts={keys ? String(rating) : undefined}>
              {keys && <kbd className="hidden rounded-sm border border-line px-1 text-xs text-fg-muted md:inline">{rating}</kbd>}
              {label}
            </Button>
          ))}
        </div>
      ) : (
        <Button onClick={show} aria-keyshortcuts={keys ? "Space Enter" : undefined}>{t.reveal}</Button>
      )}
    </section>
  );
}

function CardList() {
  const t = useT(messages);
  const locale = useLocale();
  const cards = useCards(false);
  const remove = useDeleteCard();
  const items = useMemo(() => cards.data ?? [], [cards.data]);
  if (cards.isLoading) { return <ListSkeleton rows={5} />; }
  if (cards.error) {
    return <ErrorState message={describePunisherError(cards.error)} onRetry={() => cards.refetch()} retrying={cards.isRefetching} />;
  }
  if (items.length === 0) { return <EmptyState compact title={t.allEmpty} />; }
  async function del(card: ReviewCard) {
    const ok = await confirmAction({ title: t.deleteTitle, description: t.deleteBody, confirmLabel: t.delete, danger: true });
    if (ok) { remove.mutate(card.id); }
  }
  return (
    <>
      <InlineError error={remove.error} />
      <List>
        {items.map((card) => {
          const menu: MenuItem[] = [{ key: "delete", label: t.delete, danger: true, onSelect: () => void del(card) }];
          return (
            <ListRow
              key={card.id}
              title={card.question}
              titleLines={2}
              meta={card.nextReviewAt ? t.nextReview(formatDate(new Date(card.nextReviewAt), locale, { month: "short", day: "numeric" })) : undefined}
              trailing={<Menu items={menu} label={t.cardMenu(card.question.slice(0, 30))} />}
            />
          );
        })}
      </List>
    </>
  );
}
