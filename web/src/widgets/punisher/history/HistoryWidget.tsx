
import { Link, usePathname, useSearchParams } from "@/src/shared/lib/router";
import { ChipLink, ScrollRow } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";
import { CheckHistoryList } from "./ui/CheckHistoryList";
import { CompletedByArea } from "./ui/CompletedByArea";
import { InquiryHistoryList } from "./ui/InquiryHistoryList";
import { LearnedConceptList } from "./ui/LearnedConceptList";
import { NoteHistoryList } from "./ui/NoteHistoryList";
import { QuestionHistoryList } from "./ui/QuestionHistoryList";

type HistoryTab = "lectures" | "learned" | "checks" | "inquiries" | "questions" | "notes";

type TabLabelKey = "tabLectures" | "tabLearned" | "tabChecks" | "tabInquiries" | "tabQuestions" | "tabNotes";

const TABS: { id: HistoryTab; label: TabLabelKey }[] = [
  { id: "lectures", label: "tabLectures" },
  { id: "learned", label: "tabLearned" },
  { id: "checks", label: "tabChecks" },
  { id: "inquiries", label: "tabInquiries" },
  { id: "questions", label: "tabQuestions" },
  { id: "notes", label: "tabNotes" },
];

const PANELS: Record<HistoryTab, () => React.JSX.Element> = {
  lectures: CompletedByArea,
  learned: LearnedConceptList,
  checks: CheckHistoryList,
  inquiries: InquiryHistoryList,
  questions: QuestionHistoryList,
  notes: NoteHistoryList,
};

/**
 * 기록: [완료한 강의 · 개념(완료·이미 앎) · 확인·복습(+ 기억 유지 지표) · 질문 해설 · 강의 중 질문 · 메모] — 상단 세그먼트와
 * 겹치지 않게 칩으로 고른다(?tab=). 칩은 링크(기록에 쌓임 — 뒤로 가면 앞 칩, 새 탭 가능). 줄이 넘치면 끝이 흐려진다.
 */
export function HistoryWidget() {
  const t = useT(messages);
  const pathname = usePathname();
  const params = useSearchParams();
  const tabParam = params.get("tab");
  const tab: HistoryTab = TABS.find((item) => item.id === tabParam)?.id ?? "lectures";
  const Panel = PANELS[tab];

  return (
    <div className="flex w-full flex-col gap-3">
      <nav aria-label={t.tabsLabel}>
        <ScrollRow className="-mx-4 fold:mx-0" innerClassName="px-4 py-1.5 fold:px-0">
          {TABS.map((item) => (
            <ChipLink key={item.id} href={`${pathname}?tab=${item.id}`} linkAs={Link} selected={tab === item.id}>
              {t[item.label]}
            </ChipLink>
          ))}
        </ScrollRow>
      </nav>
      <div aria-live="polite" className="sr-only">
        {t.showing(t[TABS.find((item) => item.id === tab)?.label ?? "tabLectures"])}
      </div>
      <Panel />
    </div>
  );
}
