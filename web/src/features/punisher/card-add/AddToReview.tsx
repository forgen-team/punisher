import { useEffect, useState } from "react";
import { Layers } from "lucide-react";
import { Button, ListSkeleton, Sheet, TextField, useToast } from "@forgen-team/ui";
import { useCardDraft, useDeleteCard, useSaveCard } from "@/src/entities/punisher";
import type { CardSourceKind } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

interface AddToReviewProps {
  kind: CardSourceKind;
  sourceId: string;
  /** 이미 카드가 있으면(목록이 아는 경우) 버튼 글자를 바꾼다 */
  className?: string;
}

/**
 * 메모·구절 대화 → 복습 카드(W5a). 사용자가 고른 것만, 글은 사용자가 확인·고친 그대로(AI 로 만들지 않는다).
 * 저장하면 되돌리기 알림(카드 삭제), 내일부터 복습 카드로 나온다.
 */
export function AddToReview({ kind, sourceId, className }: AddToReviewProps) {
  const t = useT(messages);
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button variant="ghost" size="sm" onClick={() => setOpen(true)} className={className}>
        <Layers className="size-4" aria-hidden="true" />
        {t.addToReview}
      </Button>
      {open && <CardSheet kind={kind} sourceId={sourceId} onClose={() => setOpen(false)} />}
    </>
  );
}

function CardSheet({ kind, sourceId, onClose }: { kind: CardSourceKind; sourceId: string; onClose: () => void }) {
  const t = useT(messages);
  const draft = useCardDraft(kind, sourceId);
  const save = useSaveCard();
  const remove = useDeleteCard();
  const toast = useToast();
  const [question, setQuestion] = useState("");
  const [answer, setAnswer] = useState("");
  const data = draft.data;
  useEffect(() => {
    if (data) {
      setQuestion(data.question);
      setAnswer(data.answer);
    }
  }, [data]);
  const existed = Boolean(data?.cardId);
  const valid = question.trim() !== "" && answer.trim() !== "";

  function submit() {
    if (!valid) { return; }
    save.mutate({ kind, sourceId, question, answer }, {
      onSuccess: (card) => {
        onClose();
        // 새로 만든 카드만 되돌린다(이미 있던 카드의 글 고침은 되돌리기 없음).
        if (!existed) {
          toast.undo(t.saved, () => {
            remove.mutateAsync(card.id).catch(() => toast.show(t.undoFailed, { tone: "danger" }));
          });
        }
      },
    });
  }

  return (
    <Sheet
      open
      onOpenChange={(o) => { if (!o) { onClose(); } }}
      title={t.sheetTitle}
      description={t.sheetDescription}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>{t.cancel}</Button>
          <Button variant="primary" onClick={submit} disabled={!valid} loading={save.isPending}>{t.save}</Button>
        </>
      }
    >
      {draft.isLoading ? (
        <ListSkeleton rows={2} />
      ) : (
        <div className="flex flex-col gap-3">
          <InlineError error={draft.error ?? save.error} />
          <TextField
            multiline
            label={t.question}
            value={question}
            onValueChange={({ value }) => setQuestion(value)}
            textareaProps={{ rows: 3, maxLength: 500 }}
          />
          <TextField
            multiline
            label={t.answer}
            description={data && !data.answer ? t.noAnswer : undefined}
            value={answer}
            onValueChange={({ value }) => setAnswer(value)}
            textareaProps={{ rows: 4, maxLength: 2000 }}
          />
        </div>
      )}
    </Sheet>
  );
}
