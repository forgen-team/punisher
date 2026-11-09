
import { Link } from "@/src/shared/lib/router";
import { Compass } from "lucide-react";
import { Section, buttonVariants } from "@/src/shared/ui/kit";
import type { NextChapterPrompt } from "@/src/shared/api/punisher";

export const NEW_CHAPTER_PATH = "/chapters/new";

function days(minutes: number, daily: number): number {
  return daily > 0 ? Math.max(0, Math.floor(minutes / daily)) : 0;
}

/** 남은 강의가 하루 학습량 × 14일보다 적을 때: 다음 챕터 고르기로 안내. */
export function NextChapterCard({ prompt }: { prompt: NextChapterPrompt }) {
  const left = days(prompt.remainingMinutes, prompt.dailyMinutes);
  return (
    <Section variant="card" title="다음 챕터 고르기">
      <div className="flex items-start gap-3 py-4">
        <Compass className="mt-0.5 size-5 shrink-0 text-accent-text" aria-hidden="true" />
        <div className="min-w-0 flex-1">
          <p className="break-keep text-sm text-fg">
            {left > 0
              ? `지금 지도에 남은 강의는 하루 ${prompt.dailyMinutes}분 기준 약 ${left}일 치예요.`
              : "지금 지도에서 배울 강의를 거의 다 마쳤어요."}{" "}
            다음엔 어떤 걸 공부해 보고 싶은지 골라 새 챕터를 열어 보세요.
          </p>
          <Link href={NEW_CHAPTER_PATH} className={`${buttonVariants({ variant: "primary", size: "sm" })} mt-3`}>
            다음 챕터 고르기
          </Link>
        </div>
      </div>
    </Section>
  );
}
