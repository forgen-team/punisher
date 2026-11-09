import { ChapterPickerWidget } from "@/src/widgets/punisher/chapter-picker";

/** 다음 챕터 고르기 (witems: workspace/punisher/chapters/new). */
export function NewChapterPage() {
  return (
    <div className="mx-auto w-full max-w-3xl">
      <ChapterPickerWidget />
    </div>
  );
}
