
import { Link } from "@/src/shared/lib/router";
import { useState } from "react";
import { PencilLine } from "lucide-react";
import { Chip } from "@/src/shared/ui/kit";
import {
  FEEDBACK_LABELS,
  QUESTION_STATUS_LABELS,
  useLectureFeedback,
  useQuestionPoll,
} from "@/src/entities/punisher";
import { localizeServerText, type FeedbackKind, type FeedbackResponse } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

const FEEDBACK_ORDER: FeedbackKind[] = ["understood", "insufficient", "too_easy", "too_hard", "want_more"];

/** 프롬프트에 반영할 수 있는 피드백(제안 문구가 있는 것) → 강사 원칙 블록 편집으로 연다(저장은 사용자가). */
const PROMPT_FEEDBACK = new Set<FeedbackKind>(["insufficient", "too_easy", "too_hard", "want_more"]);

function promptEditHref(kind: FeedbackKind): string {
  return `/settings/prompts?block=teacher.system&from=${encodeURIComponent(kind)}`;
}

function AltExplanation({ id }: { id: string }) {
  const t = useT(messages);
  const { data, error } = useQuestionPoll(id);
  if (error) { return <InlineError error={error} />; }
  const status = data?.status ?? "pending";
  return (
    <div className="border-l-2 border-line-strong pl-3 text-sm">
      <p className="mb-1 font-medium text-fg">
        {t.altExplanation} · <span className="font-normal text-fg-secondary">{QUESTION_STATUS_LABELS[status]}</span>
      </p>
      {status === "answered" && data && <Markdown className="text-base leading-[1.7]">{data.answer}</Markdown>}
      {status === "failed" && <p className="text-danger">{localizeServerText(data?.error, t.altFailed)}</p>}
    </div>
  );
}

interface FeedbackBarProps {
  lectureId: string;
  getAnchor: () => string;
}

/** 피드백 5종(칩, 선택만 강조). "설명 부족"은 재설명 요청 상태를 이어서 보여 준다. */
export function FeedbackBar({ lectureId, getAnchor }: FeedbackBarProps) {
  const t = useT(messages);
  const mutation = useLectureFeedback(lectureId);
  const [last, setLast] = useState<FeedbackResponse | null>(null);

  function send(feedback: FeedbackKind) {
    mutation.mutate({ feedback, anchor: getAnchor() }, { onSuccess: setLast });
  }

  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs font-medium text-fg-secondary">{t.howWasIt}</p>
      <div role="group" aria-label={t.feedbackGroup} className="flex flex-wrap gap-2">
        {FEEDBACK_ORDER.map((kind) => (
          <Chip
            key={kind}
            selected={last?.feedback === kind}
            disabled={mutation.isPending}
            onClick={() => send(kind)}
          >
            {FEEDBACK_LABELS[kind]}
          </Chip>
        ))}
      </div>
      <div aria-live="polite">
        <InlineError error={mutation.error} />
        {last && <p className="text-xs text-fg-muted">{t.recorded}</p>}
        {last && PROMPT_FEEDBACK.has(last.feedback) && (
          <Link
            href={promptEditHref(last.feedback)}
            className="mt-1 inline-flex min-h-touch items-center gap-1 text-sm text-accent-text underline-offset-2 hover:underline"
          >
            <PencilLine className="size-4" aria-hidden="true" />
            {t.applyToPrompt}
          </Link>
        )}
        {last?.altRequestError && (
          <p className="text-sm text-warning">{t.altStartFailed(last.altRequestError)}</p>
        )}
      </div>
      {last?.altRequestId && <AltExplanation id={last.altRequestId} />}
    </div>
  );
}
