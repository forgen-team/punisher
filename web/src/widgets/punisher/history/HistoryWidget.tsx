
import { usePathname, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { Chip } from "@/src/shared/ui/kit";
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

/** 기록: [완료한 강의 · 개념(완료·이미 앎) · 확인·복습(+ 기억 유지 지표) · 질문 해설 · 강의 중 질문 · 메모] — 상단 세그먼트와 겹치지 않게 칩으로 고른다(?tab=). */
export function HistoryWidget() {
  const t = useT(messages);
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const tabParam = params.get("tab");
  const tab: HistoryTab = TABS.find((item) => item.id === tabParam)?.id ?? "lectures";
  const Panel = PANELS[tab];

  return (
    <div className="flex w-full flex-col gap-3">
      <div role="group" aria-label={t.tabsLabel} className="-mx-4 flex gap-2 overflow-x-auto px-4 py-1.5 [scrollbar-width:none] fold:mx-0 fold:px-0">
        {TABS.map((item) => (
          <Chip
            key={item.id}
            selected={tab === item.id}
            onClick={() => router.replace(`${pathname}?tab=${item.id}`, { scroll: false })}
          >
            {t[item.label]}
          </Chip>
        ))}
      </div>
      <div aria-live="polite" className="sr-only">
        {t.showing(t[TABS.find((item) => item.id === tab)?.label ?? "tabLectures"])}
      </div>
      <Panel />
    </div>
  );
}
