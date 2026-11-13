import { BackLink } from "@/src/shared/ui/BackLink";
import { ChapterPickerWidget } from "@/src/widgets/punisher/chapter-picker";

/** 다음 챕터 고르기 (witems: workspace/punisher/chapters/new — 넓은 화면은 2열, ← 학습). */
export function NewChapterPage() {
  return (
    // xl 이상은 다른 학습 화면처럼 셸 좌우 여백(lg 32px)만 둔다 — 가운데 정렬 폭으로 여백이 더 생기지 않게
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-2 xl:max-w-none">
      <BackLink href="/" label="학습" />
      <ChapterPickerWidget />
    </div>
  );
}
