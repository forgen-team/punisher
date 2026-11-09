import { BackLink } from "@/src/shared/ui/BackLink";
import { ChapterPickerWidget } from "@/src/widgets/punisher/chapter-picker";

/** 다음 챕터 고르기 (witems: workspace/punisher/chapters/new — 넓은 화면은 2열, ← 학습). */
export function NewChapterPage() {
  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-2 xl:max-w-6xl">
      <BackLink href="/" label="학습" />
      <ChapterPickerWidget />
    </div>
  );
}
