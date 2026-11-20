
import { useCallback, useEffect, useMemo, useState } from "react";
import { useConcepts, useLearningSettings } from "@/src/entities/punisher";

const DISMISS_KEY = "punisher.map.knownOnboarding.dismissed";
/** 완료·앎 개념이 이 수 미만이면 "거의 아무것도 표시하지 않은" 첫 사용자로 본다. */
const FEW_LEARNED = 3;

function readDismissed(): boolean {
  try {
    return window.localStorage.getItem(DISMISS_KEY) === "1";
  } catch (err) {
    console.warn("punisher: localStorage read failed", err);
    return false;
  }
}

function writeDismissed(): void {
  try {
    window.localStorage.setItem(DISMISS_KEY, "1");
  } catch (err) {
    console.warn("punisher: localStorage write failed", err);
  }
}

/**
 * 첫 진입 안내 표시 여부 + 닫기(한 번 닫으면 localStorage에 기억).
 * 처음 설정(설명 수준)을 마친 사용자에게는 띄우지 않는다 — 그 시트가 이 안내를 대신한다.
 */
export function useKnownOnboardingPrompt() {
  const concepts = useConcepts();
  const settings = useLearningSettings();
  const setupDone = Boolean(settings.data?.settings.readerLevel);
  const settingsSettled = Boolean(settings.data) || Boolean(settings.error);
  const [dismissed, setDismissed] = useState(true);

  useEffect(() => setDismissed(readDismissed()), []);

  const learnedCount = useMemo(
    () => (concepts.data ?? []).filter((c) => c.userState === "completed" || c.userState === "known").length,
    [concepts.data],
  );

  const dismiss = useCallback(() => {
    writeDismissed();
    setDismissed(true);
  }, []);

  return { show: Boolean(concepts.data) && settingsSettled && !setupDone && !dismissed && learnedCount < FEW_LEARNED, dismiss };
}

export const KNOWN_ONBOARDING_DISMISS_KEY = DISMISS_KEY;
export { writeDismissed as rememberKnownOnboardingDismissed };
