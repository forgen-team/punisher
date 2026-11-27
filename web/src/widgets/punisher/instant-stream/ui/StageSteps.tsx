import { cn } from "@/src/shared/lib/cn";
import { useLocale, translate } from "@/src/shared/i18n";
import { currentStep, stageLabel, type StageState, type StepKey } from "../lib/stageState";
import { messages } from "../i18n";

const STEPS: Array<{ key: StepKey; label: "stepWrite" | "stepReview" | "stepFix" }> = [
  { key: "write", label: "stepWrite" },
  { key: "review", label: "stepReview" },
  { key: "fix", label: "stepFix" },
];

/** 작성 → 검수 → (고침) — 지금 칸만 단계 이름(고치는 중 n/2 등)으로 강조한다. 고침 칸은 고쳐 쓰기가 시작돼야 나온다. */
export function StageSteps({ state }: { state: StageState }) {
  const locale = useLocale();
  const t = translate(messages, locale);
  const now = currentStep(state.stage);
  const steps = STEPS.filter((s) => s.key !== "fix" || state.attempt > 0);
  return (
    <ol className="flex flex-wrap items-center gap-2 text-sm">
      {steps.map((s, index) => (
        <li key={s.key} aria-current={s.key === now ? "step" : undefined} className="flex items-center gap-2">
          {index > 0 && <span aria-hidden="true" className="text-fg-muted">→</span>}
          <span
            className={cn(
              "rounded-full px-2.5 py-1",
              s.key === now ? "bg-accent-subtle font-medium text-accent-text" : "text-fg-muted",
            )}
          >
            {s.key === now ? stageLabel(state, locale) : t[s.label]}
          </span>
        </li>
      ))}
    </ol>
  );
}
