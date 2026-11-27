import { useState } from "react";
import { usePathname, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { Button, Sheet } from "@forgen-team/ui";
import { describeSettingsError } from "@/src/entities/punisher";
import type { ChosenReaderLevel, Familiarity, FamiliarityArea } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";
import { AreasStep, LevelStep, SummaryStep } from "./steps";
import { defaultFamiliarity, useFirstSetupGate, useFirstSetupSave, type FamiliarityByArea } from "./useFirstSetup";

const TOTAL_STEPS = 3;
type Step = 1 | 2 | 3;

interface FirstSetupSheetProps {
  open: boolean;
  /** 건너뛰기·×·배경 — 기본 수준을 저장하고 닫는다(2주 뒤 다시) */
  onSkip: () => void;
  /** 시작하기 저장을 마쳤을 때 */
  onDone: () => void;
}

/** 처음 설정(3단계) — 개발 경험 → 영역별 아는 정도 → 요약. */
export function FirstSetupSheet({ open, onSkip, onDone }: FirstSetupSheetProps) {
  const t = useT(messages);
  const [step, setStep] = useState<Step>(1);
  const [level, setLevel] = useState<ChosenReaderLevel | null>(null);
  const [familiarity, setFamiliarity] = useState<FamiliarityByArea | null>(null);
  /** familiarity 기본값을 만든 수준 — 1단계에서 수준을 바꾸면 2단계 기본값을 다시 만든다. */
  const [familiarityLevel, setFamiliarityLevel] = useState<ChosenReaderLevel | null>(null);
  const [failedIds, setFailedIds] = useState<string[]>([]);
  const { skip, finish, markKnown, saveError, savingSettings, markingKnown, progress } = useFirstSetupSave();
  const busy = savingSettings || markingKnown;

  function goAreas() {
    if (!level) { return; }
    if (familiarityLevel !== level || !familiarity) {
      setFamiliarity(defaultFamiliarity(level));
      setFamiliarityLevel(level);
    }
    setStep(2);
  }

  function pick(area: FamiliarityArea, value: Familiarity) {
    setFamiliarity((prev) => (prev ? { ...prev, [area]: value } : prev));
  }

  // 건너뛰기와 × 는 같다: 기본 수준을 저장하고 닫는다. 저장이 실패해도 닫는다(수준이 빈 채로 2주 뒤 다시 묻는다).
  async function handleSkip() {
    try {
      await skip();
    } catch (err) {
      console.warn("punisher: first setup skip failed", err);
    }
    onSkip();
  }

  async function handleStart() {
    if (!level || !familiarity) { return; }
    try {
      const result = await finish(level, familiarity);
      if (result.ok) { onDone(); return; }
      setFailedIds(result.failedIds);
    } catch (err) {
      console.warn("punisher: first setup save failed", err);
    }
  }

  async function handleRetry() {
    const result = await markKnown(failedIds);
    if (result.failedIds.length === 0) { onDone(); return; }
    setFailedIds(result.failedIds);
  }

  const failed = failedIds.length > 0 && !markingKnown;
  const startLabel = markingKnown && progress ? t.savingKnown(progress.done, progress.total) : savingSettings ? t.savingSettings : t.start;

  const footer =
    step === 1 ? (
      <>
        <Button variant="ghost" onClick={handleSkip} loading={savingSettings} disabled={busy}>
          {t.skip}
        </Button>
        <Button variant="primary" onClick={goAreas} disabled={!level || busy}>
          {t.next}
        </Button>
      </>
    ) : step === 2 ? (
      <>
        <Button variant="ghost" onClick={() => setStep(1)}>
          {t.back}
        </Button>
        <Button variant="primary" onClick={() => setStep(3)}>{t.next}</Button>
      </>
    ) : failed ? (
      <>
        <Button variant="ghost" onClick={onDone}>
          {t.close}
        </Button>
        <Button variant="primary" onClick={handleRetry}>{t.retry}</Button>
      </>
    ) : (
      <>
        <Button variant="ghost" onClick={() => setStep(2)} disabled={busy}>
          {t.back}
        </Button>
        <Button variant="primary" onClick={handleStart} loading={busy}>
          {startLabel}
        </Button>
      </>
    );

  const title = step === 1 ? t.levelTitle : step === 2 ? t.areasTitle : t.summaryTitle;
  const description = step === 1 ? t.levelHelper : step === 2 ? t.areasHelper : undefined;

  return (
    <Sheet
      open={open}
      onOpenChange={(o) => {
        if (o || busy) { return; }
        // 시작하기까지 저장했으면(이미 앎 실패 안내 중) 닫기만, 아니면 건너뛰기와 같다.
        if (failed) { onDone(); } else { void handleSkip(); }
      }}
      title={
        <>
          <span className="block text-xs font-normal text-fg-muted">{t.eyebrow(step, TOTAL_STEPS)}</span>
          {title}
        </>
      }
      description={description}
      footer={footer}
      desktop="dialog"
    >
      <div aria-live="polite" className="text-sm empty:hidden">
        {saveError && !busy && (
          <p role="alert" className="mb-2 text-danger">
            {describeSettingsError(saveError)}
          </p>
        )}
        {failed && (
          <p role="alert" className="mb-2 text-danger">
            {t.knownFailed(failedIds.length)}
          </p>
        )}
      </div>
      {step === 1 && <LevelStep value={level} onChange={setLevel} disabled={busy} />}
      {step === 2 && familiarity && <AreasStep value={familiarity} onChange={pick} />}
      {step === 3 && level && familiarity && <SummaryStep level={level} familiarity={familiarity} />}
    </Sheet>
  );
}

/** 시트를 연 주소 — 정책 20 §3(시트는 주소, push). 자동으로 열 때는 주소를 바꾸지 않는다. */
export const FIRST_SETUP_PARAM = "first";

/**
 * 오늘 화면에 둔다 — 설명 수준을 아직 정하지 않은 사용자(또는 2주 전에 건너뛴 사용자)에게 열리고, `?first=1` 로도 열린다.
 * 열릴 때만 시트를 그린다(개념 목록 조회·단계 상태는 열 때마다 새로).
 */
export function FirstSetup() {
  const gate = useFirstSetupGate();
  const params = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const fromAddress = params.get(FIRST_SETUP_PARAM) === "1";
  if (!gate.open && !fromAddress) { return null; }
  const leaveAddress = () => {
    if (!fromAddress) { return; }
    const next = new URLSearchParams(params);
    next.delete(FIRST_SETUP_PARAM);
    const query = next.toString();
    router.replace(query ? `${pathname}?${query}` : pathname);
  };
  return (
    <FirstSetupSheet
      open
      onSkip={() => { gate.skip(); leaveAddress(); }}
      onDone={() => { gate.finish(); leaveAddress(); }}
    />
  );
}
