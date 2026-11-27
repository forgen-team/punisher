import type { MouseEvent } from "react";
import { LinkTabs } from "@forgen-team/ui";
import { useCompletedAreas, useCheckHistory, useConcepts, useSearch } from "@/src/entities/punisher";
import { useLocale, useT, type Locale } from "@/src/shared/i18n";
import { HISTORY_TABS, type HistoryTab } from "../lib/historyParams";
import { matchSessions } from "../lib/checkHistory";
import { matchLearnedConcepts, noteHits } from "../lib/notesFeed";
import { messages } from "../i18n";
import { useHistoryNav } from "./useHistoryNav";

type Counts = Partial<Record<HistoryTab, number>>;

const LABEL_KEYS = { learn: "tabLearn", review: "tabReview", notes: "tabNotes" } as const;

/** 검색어가 있을 때만 쓴다 — 탭마다 맞는 기록 수(불러오는 중이면 아직 없음). */
function useTabCounts(q: string, locale: Locale): Counts {
  const areas = useCompletedAreas(q);
  const concepts = useConcepts();
  const history = useCheckHistory();
  const search = useSearch(q);
  const counts: Counts = {};
  if (areas.data && concepts.data) {
    counts.learn = areas.data.reduce((n, a) => n + a.count, 0) + matchLearnedConcepts(concepts.data, q, "").length;
  }
  if (history.data) {
    counts.review = matchSessions(history.data.pages.flatMap((p) => p.sessions), q, locale).length;
  }
  if (search.data) { counts.notes = noteHits(search.data).length; }
  return counts;
}

function CountedTabs({ q }: { q: string }) {
  return <TabsNav counts={useTabCounts(q, useLocale())} />;
}

/**
 * 화면 안 탭 3(학습·복습·통계·질문·메모). 검색어가 있으면 탭 이름 옆에 그 탭의 결과 수가 붙는다.
 * 탭은 진짜 링크(새 탭 가능)이고, 눌러 이동하면 기록에 쌓지 않고 바꾼다(replace).
 */
export function HistoryTabs() {
  const { q } = useHistoryNav();
  return q ? <CountedTabs q={q} /> : <TabsNav />;
}

function TabsNav({ counts }: { counts?: Counts }) {
  const t = useT(messages);
  const { view, hrefFor, go } = useHistoryNav();

  function replaceOnClick(event: MouseEvent<HTMLDivElement>) {
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) { return; }
    const anchor = (event.target as HTMLElement).closest("a");
    const links = Array.from(event.currentTarget.querySelectorAll("nav a"));
    const tab = anchor ? HISTORY_TABS[links.indexOf(anchor)] : undefined;
    if (!tab) { return; }
    event.preventDefault();
    go({ tab });
  }

  return (
    <div onClickCapture={replaceOnClick}>
      <LinkTabs
        label={t.tabsLabel}
        value={view.tab}
        items={HISTORY_TABS.map((id) => ({ value: id, label: t[LABEL_KEYS[id]], href: hrefFor({ tab: id }), count: counts?.[id] }))}
      />
    </div>
  );
}
