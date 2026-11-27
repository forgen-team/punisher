
import { useRouter } from "@/src/shared/lib/router";
import { Button, Sheet } from "@forgen-team/ui";
import { useRegenerateLecture } from "@/src/entities/punisher";
import { instantHref, lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

interface RegenerateSheetProps {
  lectureId: string;
  open: boolean;
  onClose: () => void;
}

/** 같은 개념·목표·깊이로 새 버전을 만들고 생성 화면(또는 바로 나온 강의)으로 간다. 시트·flagged 배너가 같이 쓴다. */
export function useStartRegenerate(lectureId: string) {
  const router = useRouter();
  const regenerate = useRegenerateLecture();
  function start() {
    regenerate.mutate(lectureId, {
      onSuccess: (res) => {
        if (res.jobId) { router.push(instantHref(res.jobId)); }
        else if (res.lectureId) { router.push(lectureHref(res.lectureId)); }
      },
    });
  }
  return { start, isPending: regenerate.isPending, error: regenerate.error };
}

/** "이 강의 새 프롬프트로 다시 만들기" 확인 — 같은 개념·목표·깊이로 새 버전(개요→본문→감수)을 만든다. */
export function RegenerateSheet({ lectureId, open, onClose }: RegenerateSheetProps) {
  const t = useT(messages);
  const regenerate = useStartRegenerate(lectureId);
  const start = regenerate.start;

  return (
    <Sheet
      open={open}
      onOpenChange={(o) => { if (!o) { onClose(); } }}
      title={t.regenTitle}
      description={t.regenDescription}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            {t.cancel}
          </Button>
          <Button variant="primary" loading={regenerate.isPending} onClick={start}>
            {t.regenerate}
          </Button>
        </>
      }
    >
      <ul className="list-disc space-y-1 break-keep pl-5 text-sm text-fg-secondary">
        <li>{t.regenStep1}</li>
        <li>{t.regenStep2}</li>
        <li>{t.regenStep3}</li>
      </ul>
      {regenerate.error && (
        <p role="alert" className="mt-3 text-sm text-danger">
          {describePunisherError(regenerate.error)}
        </p>
      )}
    </Sheet>
  );
}
