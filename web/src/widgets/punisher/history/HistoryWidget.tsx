import { useEffect } from "react";
import { useRouter } from "@/src/shared/lib/router";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";
import { HistoryTabs } from "./ui/HistoryTabs";
import { LearnTab } from "./ui/LearnTab";
import { NotesTab } from "./ui/NotesTab";
import { ReviewTab } from "./ui/ReviewTab";
import { SearchBar } from "./ui/SearchBar";
import { useHistoryNav } from "./ui/useHistoryNav";

const PANELS = { learn: LearnTab, review: ReviewTab, notes: NotesTab } as const;
const LABEL_KEYS = { learn: "tabLearn", review: "tabReview", notes: "tabNotes" } as const;

/**
 * 기록(T-list + 탭 3): 화면 안 탭 [학습 · 복습·통계 · 질문·메모] → 검색창 1개(전 범위) → 탭 본문. 제목·페이지 동작·폭은 틀이 준다.
 * 탭·칩·검색어는 주소(?tab=&q=&area=&kind=)에 남고 replace 로 바뀐다. 옛 탭 값(?tab=lectures 등)은 새 탭으로 옮긴다.
 */
export function HistoryWidget() {
  const t = useT(messages);
  const router = useRouter();
  const { view, hrefFor } = useHistoryNav();
  const Panel = PANELS[view.tab];

  // 옛 주소(6탭 값)는 열자마자 새 주소로 바꾼다(기록에 쌓지 않는다).
  const canonical = hrefFor({ tab: view.tab, kind: view.kind });
  useEffect(() => {
    if (view.legacy) { router.replace(canonical, { scroll: false }); }
  }, [view.legacy, canonical, router]);

  return (
    <div className="flex w-full flex-col gap-3">
      <HistoryTabs />
      <div aria-live="polite" className="sr-only">{t.showing(t[LABEL_KEYS[view.tab]])}</div>
      <SearchBar />
      <Panel />
    </div>
  );
}
