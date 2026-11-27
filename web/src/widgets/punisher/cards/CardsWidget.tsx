import { useMemo, useState, type MouseEvent, type ReactNode } from "react";
import { Link, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { Badge, Button, EmptyState, ErrorState, LinkTabs, List, ListRow, ListSkeleton, Menu, useToast, type MenuItemDef } from "@forgen-team/ui";
import { lectureAnchorHref, useCards, useDeleteCard, useGradeCard, useRestoreCard } from "@/src/entities/punisher";
import { describePunisherError, type CardRating, type ReviewCard } from "@/src/shared/api/punisher";
import { useStudyShortcuts } from "@/src/shared/lib/shortcuts";
import { useOnline } from "@/src/shared/lib/useOnline";
import { formatDate, useLocale, useT, type Locale } from "@/src/shared/i18n";
import { InlineError } from "@/src/shared/ui/punisher";
import { START, graded, reveal } from "./lib/session";
import { messages } from "./i18n";

type CardsTab = "review" | "all";

/** 자기 채점 — SRS 등급 값 1·2·3(FSRS Again·Hard·Good)과 그 단축키. 주 버튼은 3 하나. */
const RATINGS: ReadonlyArray<readonly [CardRating, "again" | "hard" | "good"]> = [[1, "again"], [2, "hard"], [3, "good"]];

const FOOTER_CLASS =
  "sticky bottom-[var(--app-bottom-offset)] z-nav -mx-4 mt-6 flex items-center gap-2 border-t border-line bg-canvas px-4 py-3 fold:-mx-6 fold:px-6 lg:mx-0 lg:px-0";

/** 고정 Footer — 하단 탭 위(--app-bottom-offset). 안은 한 줄(오른쪽에 주 동작). */
function Footer({ children }: { children: ReactNode }) {
  return <div className={FOOTER_CLASS}>{children}</div>;
}

/**
 * 카드 복습 — 탭 [복습 n | 모든 카드](?tab=all). 탭 전환은 기록을 쌓지 않는다(replace).
 * 복습: 앞면 → Footer [답 보기] → [다시][어려움][알았음](SRS 등급 1·2·3, 주 버튼은 [알았음]). 단축키(Space/Enter 답 보기 · 1 다시 · 2 어려움 · 3 알았음)는 설정에서 끌 수 있다.
 * 제목·←·설정은 틀이 그린다.
 */
export function CardsWidget() {
  const t = useT(messages);
  const router = useRouter();
  const params = useSearchParams();
  const tab: CardsTab = params.get("tab") === "all" ? "all" : "review";
  const due = useCards(true);
  // 탭 링크는 진짜 링크(새 탭 가능)로 두고, 보통 클릭만 replace 로 바꾼다.
  function replaceTab(event: MouseEvent<HTMLElement>) {
    const link = (event.target as HTMLElement).closest("a");
    const href = link?.getAttribute("href");
    if (!href || event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) { return; }
    event.preventDefault();
    router.replace(href);
  }
  return (
    <div className="flex w-full flex-col gap-4">
      <div onClickCapture={replaceTab}>
        <LinkTabs
          label={t.tabsLabel}
          value={tab}
          items={[
            { value: "review", label: t.tabReview, href: "/cards", count: due.data?.length },
            { value: "all", label: t.tabAll, href: "/cards?tab=all" },
          ]}
        />
      </div>
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
    grade.mutate({ id: card.id, rating }, { onSuccess: () => setS((prev) => graded(prev, rating)) });
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
    return <ErrorState title={describePunisherError(cards.error)} onRetry={() => cards.refetch()} />;
  }
  if (!list || list.length === 0) {
    return (
      <EmptyState
        title={t.emptyTitle}
        description={t.emptyBody}
        action={<Button asChild variant="secondary" size="sm"><Link href="/">{t.toToday}</Link></Button>}
      />
    );
  }
  if (!card) {
    return (
      <div className="flex flex-col items-start gap-2 pt-4">
        <p className="text-xl font-semibold text-fg">{t.doneTitle}</p>
        <p className="text-sm text-fg-secondary">{t.doneBody(s.graded - s.again - s.hard, s.hard, s.again)}</p>
        <Button asChild variant="primary" className="mt-3"><Link href="/">{t.toToday}</Link></Button>
      </div>
    );
  }

  return (
    <section aria-label={t.title} className="flex flex-col gap-4">
      <div className="flex items-center justify-between text-sm text-fg-secondary">
        <span className="tabular-nums">{t.progress(s.index + 1, list.length)}</span>
        {keys && <span className="hidden text-xs text-fg-muted md:inline">{t.shortcutsHint}</span>}
      </div>
      {!online && <p role="status" className="bg-warning-subtle px-3 py-2 text-sm text-warning">{t.offline}</p>}
      <article className="flex min-h-48 flex-col gap-4 border border-line bg-canvas-elevated p-4">
        {card.stale && <Badge variant="weak" tone="warning" className="self-start" title={t.staleHint}>{t.stale}</Badge>}
        <p className="whitespace-pre-wrap break-keep text-xl font-medium text-fg">{card.question}</p>
        {s.revealed ? (
          <div className="border-t border-line pt-4">
            <p className="whitespace-pre-wrap break-keep text-base text-fg-secondary">{card.answer}</p>
            {card.lectureId && (
              <Link href={lectureAnchorHref(card.lectureId, card.anchor)} className="mt-3 inline-block text-sm text-accent-text underline-offset-2 hover:underline">
                {t.fromLecture}
              </Link>
            )}
            {card.sourceKind === "external" && <ExternalSource card={card} />}
          </div>
        ) : null}
      </article>
      <InlineError error={grade.error} />
      <Footer>
        <span className="flex-1" />
        {s.revealed ? (
          <>
            {RATINGS.map(([rating, label]) => (
              <Button key={rating} variant={rating === 3 ? "primary" : "secondary"} disabled={!online || grade.isPending}
                loading={grade.isPending && grade.variables?.rating === rating} onClick={() => rate(rating)}
                aria-keyshortcuts={keys ? String(rating) : undefined}>
                {t[label]}
              </Button>
            ))}
          </>
        ) : (
          <Button variant="primary" onClick={show} aria-keyshortcuts={keys ? "Space Enter" : undefined}>{t.reveal}</Button>
        )}
      </Footer>
    </section>
  );
}

/** 모든 카드 — 목록. 행 ⋯ "카드 삭제"는 확인 없이 바로 지우고 토스트 [되돌리기] 로 되살린다(서버는 표시만 하고 복습 기록을 남긴다). */
function CardList() {
  const t = useT(messages);
  const locale = useLocale();
  const cards = useCards(false);
  const remove = useDeleteCard();
  const restore = useRestoreCard();
  const toast = useToast();
  const items = useMemo(() => cards.data ?? [], [cards.data]);
  if (cards.isLoading) { return <ListSkeleton rows={5} />; }
  if (cards.error) {
    return <ErrorState title={describePunisherError(cards.error)} onRetry={() => cards.refetch()} />;
  }
  if (items.length === 0) { return <EmptyState title={t.allEmpty} />; }
  function del(card: ReviewCard) {
    remove.mutate(card.id, {
      onSuccess: () => toast.undo(t.deleted, () => {
        restore.mutateAsync(card.id).catch(() => toast.show(t.undoFailed, { tone: "danger" }));
      }),
    });
  }
  return (
    <>
      <InlineError error={remove.error} />
      <List>
        {items.map((card) => {
          const menu: MenuItemDef[] = [{ key: "delete", label: t.delete, danger: true, onSelect: () => del(card) }];
          return (
            <ListRow
              key={card.id}
              title={card.question}
              meta={cardMeta(card, t, locale)}
              trailing={<Menu items={menu} label={t.cardMenu(card.question.slice(0, 30))} />}
            />
          );
        })}
      </List>
    </>
  );
}

/** 외부 카드(atlas 등)를 보낸 앱 이름 — externalId 의 "app:" 앞부분. */
function externalApp(card: ReviewCard): string {
  return card.externalId.split(":")[0] || card.externalId;
}

function ExternalSource({ card }: { card: ReviewCard }) {
  const t = useT(messages);
  const label = t.source(externalApp(card));
  if (!card.sourceUrl) { return <p className="mt-3 text-sm text-fg-muted">{label}</p>; }
  return (
    <a href={card.sourceUrl} target="_blank" rel="noopener noreferrer"
      className="mt-3 inline-block text-sm text-accent-text underline-offset-2 hover:underline">
      {label}
    </a>
  );
}

function cardMeta(card: ReviewCard, t: (typeof messages)["ko"], locale: Locale): string | undefined {
  const parts: string[] = [];
  if (card.stale) { parts.push(t.stale); }
  if (card.sourceKind === "external") { parts.push(t.source(externalApp(card))); }
  if (card.nextReviewAt) { parts.push(t.nextReview(formatDate(new Date(card.nextReviewAt), locale, { month: "short", day: "numeric" }))); }
  return parts.length > 0 ? parts.join(" · ") : undefined;
}
