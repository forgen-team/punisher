import { BottomSheet, Button } from "@/src/shared/ui/kit";
import { describePunisherError, type LectureSection, type SectionFix } from "@/src/shared/api/punisher";
import { useStartRegenerate } from "./RegenerateSheet";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

interface FixesSheetProps {
  lectureId: string;
  fixes: SectionFix[];
  sections: LectureSection[];
  open: boolean;
  onClose: () => void;
}

/** '고칠 곳' — 내가 막힌 섹션과 이유. 이 신호를 넣어 새 버전을 만든다(지금 강의는 그대로). */
export function FixesSheet({ lectureId, fixes, sections, open, onClose }: FixesSheetProps) {
  const t = useT(messages);
  const regenerate = useStartRegenerate(lectureId);
  const title = (kind: string) => sections.find((s) => s.kind === kind)?.title ?? kind;
  return (
    <BottomSheet
      open={open}
      onClose={onClose}
      title={t.fixesTitle}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            {t.close}
          </Button>
          <Button loading={regenerate.isPending} onClick={regenerate.start}>
            {t.regenerateWithFixes}
          </Button>
        </>
      }
    >
      <ul className="flex flex-col gap-3">
        {fixes.map((f) => (
          <li key={f.section}>
            <p className="text-sm font-semibold">{title(f.section)}</p>
            <ul className="mt-0.5 list-disc pl-5 text-sm text-fg-secondary">
              {f.reasons.map((r) => (
                <li key={r} className="break-keep">{r}</li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
      {regenerate.error && (
        <p role="alert" className="mt-3 text-sm text-danger">
          {describePunisherError(regenerate.error)}
        </p>
      )}
    </BottomSheet>
  );
}
