
import { usePathname, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { Chip } from "@/src/shared/ui/kit";
import { CompletedByArea } from "./ui/CompletedByArea";
import { InquiryHistoryList } from "./ui/InquiryHistoryList";
import { LearnedConceptList } from "./ui/LearnedConceptList";
import { NoteHistoryList } from "./ui/NoteHistoryList";
import { QuestionHistoryList } from "./ui/QuestionHistoryList";

type HistoryTab = "lectures" | "learned" | "inquiries" | "questions" | "notes";

const TABS: { id: HistoryTab; label: string }[] = [
  { id: "lectures", label: "완료한 강의·해설" },
  { id: "learned", label: "개념" },
  { id: "inquiries", label: "질문 해설" },
  { id: "questions", label: "강의 중 질문" },
  { id: "notes", label: "메모" },
];

const PANELS: Record<HistoryTab, () => React.JSX.Element> = {
  lectures: CompletedByArea,
  learned: LearnedConceptList,
  inquiries: InquiryHistoryList,
  questions: QuestionHistoryList,
  notes: NoteHistoryList,
};

/** 기록: [완료한 강의 · 개념(완료·이미 앎) · 질문 해설 · 강의 중 질문 · 메모] — 상단 세그먼트와 겹치지 않게 칩으로 고른다(?tab=). */
export function HistoryWidget() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const tabParam = params.get("tab");
  const tab: HistoryTab = TABS.find((t) => t.id === tabParam)?.id ?? "lectures";
  const Panel = PANELS[tab];

  return (
    <div className="flex w-full flex-col gap-3">
      <div role="group" aria-label="기록 종류" className="-mx-4 flex gap-2 overflow-x-auto px-4 py-1.5 fold:mx-0 fold:px-0">
        {TABS.map((t) => (
          <Chip
            key={t.id}
            selected={tab === t.id}
            onClick={() => router.replace(`${pathname}?tab=${t.id}`, { scroll: false })}
          >
            {t.label}
          </Chip>
        ))}
      </div>
      <div aria-live="polite" className="sr-only">
        {TABS.find((t) => t.id === tab)?.label} 표시 중
      </div>
      <Panel />
    </div>
  );
}
