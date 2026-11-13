
import { BottomSheet, Button } from "@/src/shared/ui/kit";
import { describeCheckError } from "@/src/entities/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import type { KnownChoice } from "./useKnownChoice";

const TITLE = "짧게 확인할까요?";

function ChoiceButtons({ choice }: { choice: KnownChoice }) {
  return (
    <div className="flex flex-wrap justify-end gap-2">
      <Button variant="ghost" onClick={choice.skip} disabled={choice.verifying}>
        건너뛰기
      </Button>
      <Button onClick={choice.verify} loading={choice.verifying}>
        확인하기
      </Button>
    </div>
  );
}

function ChoiceError({ choice }: { choice: KnownChoice }) {
  if (choice.knownError) { return <InlineError error={choice.knownError} />; }
  if (!choice.error) { return null; }
  return (
    <p role="alert" className="text-sm text-danger">
      {describeCheckError(choice.error)}
    </p>
  );
}

/** '이미 앎'을 누른 뒤의 선택 — dialog: 대화상자, inline: 그 자리(시트 안 동작 영역)에서. */
export function KnownVerifyChoice({ choice, variant = "dialog" }: { choice: KnownChoice; variant?: "dialog" | "inline" }) {
  if (variant === "inline") {
    if (!choice.asking) { return null; }
    return (
      <div className="flex w-full flex-col gap-2">
        <p className="text-sm font-medium text-fg">{TITLE}</p>
        <ChoiceButtons choice={choice} />
        <ChoiceError choice={choice} />
      </div>
    );
  }
  return (
    <BottomSheet open={choice.asking} onClose={choice.close} title={TITLE} footer={<ChoiceButtons choice={choice} />}>
      <ChoiceError choice={choice} />
    </BottomSheet>
  );
}
