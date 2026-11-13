
import { useRouter } from "@/src/shared/lib/router";
import { BottomSheet, Button } from "@/src/shared/ui/kit";
import { useRegenerateLecture } from "@/src/entities/punisher";
import { instantHref, lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError } from "@/src/shared/api/punisher";

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
  const regenerate = useStartRegenerate(lectureId);
  const start = regenerate.start;

  return (
    <BottomSheet
      open={open}
      onClose={onClose}
      title="새 프롬프트로 다시 만들기"
      description="지금 강의는 그대로 두고 새 버전을 만들어요"
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            취소
          </Button>
          <Button loading={regenerate.isPending} onClick={start}>
            다시 만들기
          </Button>
        </>
      }
    >
      <ul className="list-disc space-y-1 break-keep pl-5 text-sm text-fg-secondary">
        <li>같은 개념·학습 목표·깊이로, 지금 저장된 내 프롬프트를 써서 개요 → 본문 → 감수 순서로 만들어요.</li>
        <li>몇 분 걸려요. 만들어지면 이 강의 아래 &lsquo;새 프롬프트로 다시 만든 강의&rsquo;에 달려요.</li>
        <li>이미 앎·완료 표시와 진도는 바뀌지 않아요.</li>
      </ul>
      {regenerate.error && (
        <p role="alert" className="mt-3 text-sm text-danger">
          {describePunisherError(regenerate.error)}
        </p>
      )}
    </BottomSheet>
  );
}
