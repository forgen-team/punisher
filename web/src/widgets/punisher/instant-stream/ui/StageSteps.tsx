import { Check } from "lucide-react";
import { cn } from "@/src/shared/lib/cn";
import { useLocale, translate } from "@/src/shared/i18n";
import { currentStep, stageLabel, type StageState, type StepKey } from "../lib/stageState";
import { messages } from "../i18n";

const STEPS: Array<{ key: StepKey; label: "stepWrite" | "stepReview" | "stepFix" }> = [
  { key: "write", label: "stepWrite" },
  { key: "review", label: "stepReview" },
  { key: "fix", label: "stepFix" },
];

/**
 * 작성 → 검수 → (고침) 진행 줄 — 끝낸 단계는 체크, 지금 단계는 점 + 단계 문구(고치는 중 n/2 등), 남은 단계는 흐리게.
 * 고침 칸은 고쳐 쓰기가 시작돼야 나온다.
 */
export function StageSteps({ state }: { state: StageState }) {
  const locale = useLocale();
  const t = translate(messages, locale);
  const now = currentStep(state.stage);
  const steps = STEPS.filter((s) => s.key !== "fix" || state.attempt > 0);
  const nowIndex = steps.findIndex((s) => s.key === now);
  const allDone = state.stage === "published";
  return (
    <ol className="flex flex-col gap-2 text-base">
      {steps.map((s, index) => {
        const done = allDone || (nowIndex >= 0 && index < nowIndex);
        const current = s.key === now;
        return (
          <li
            key={s.key}
            aria-current={current ? "step" : undefined}
            className={cn("flex items-center gap-2", current ? "font-medium text-fg" : done ? "text-fg-secondary" : "text-fg-muted")}
          >
            {done ? (
              <Check className="size-4 shrink-0 text-success" aria-hidden="true" />
            ) : (
              <span aria-hidden="true" className={cn("size-2 shrink-0 rounded-full", current ? "bg-accent-solid" : "bg-line-strong")} />
            )}
            {current ? stageLabel(state, locale) : t[s.label]}
          </li>
        );
      })}
    </ol>
  );
}
