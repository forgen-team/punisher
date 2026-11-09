import { BackLink } from "@/src/shared/ui/BackLink";
import { ChapterPickerWidget } from "@/src/widgets/punisher/chapter-picker";

/** 다음 챕터 고르기 (witems: workspace/punisher/chapters/new). */
export function NewChapterPage() {
  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-2">
      <BackLink href="/map" label="지도" />
      <ChapterPickerWidget />
    </div>
  );
}
