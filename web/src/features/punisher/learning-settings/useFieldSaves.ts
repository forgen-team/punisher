import { useCallback, useEffect, useRef, useState } from "react";

/** "저장했어요" 를 보여 주는 시간(ms). */
const FLASH_MS = 1800;

/** 바로 저장한 칸 하나의 "저장했어요" 를 잠깐 켜 둔다. */
export function useSavedFlash() {
  const [savedKey, setSavedKey] = useState<string | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  useEffect(() => () => clearTimeout(timer.current), []);
  const flash = useCallback((key: string) => {
    clearTimeout(timer.current);
    setSavedKey(key);
    timer.current = setTimeout(() => setSavedKey(null), FLASH_MS);
  }, []);
  return { savedKey, flash };
}

/**
 * 칸마다 바로 저장하는 설정(부분 PUT)의 공통 상태.
 * - unsaved: 저장하는 동안에도 바꾼 값이 화면에 먼저 보이게 하는 값(끝나면 서버 값으로 돌아간다, 실패하면 원래 값).
 * - errors: 칸별 저장 오류 문구(다시 저장하면 지운다).
 * - savedKey: 방금 저장한 칸(patch 의 첫 키) — 그 자리에 "저장했어요" 를 잠깐 보인다.
 * 저장은 호출마다 끝나는 mutateAsync 로 받아 그 칸만 푼다(mutate 의 콜백은 마지막 호출에만 온다).
 */
export function useFieldSaves<P extends object>(save: (patch: P) => Promise<unknown>, describe: (error: unknown) => string) {
  const [unsaved, setUnsaved] = useState<Partial<P>>({});
  const [errors, setErrors] = useState<Partial<Record<keyof P, string>>>({});
  const { savedKey, flash } = useSavedFlash();

  const saveField = (patch: P) => {
    const keys = Object.keys(patch);
    setUnsaved((prev) => ({ ...prev, ...patch }));
    setErrors((prev) => {
      const next: Record<string, string | undefined> = { ...prev };
      for (const key of keys) { delete next[key]; }
      return next as Partial<Record<keyof P, string>>;
    });
    save(patch)
      .then(() => { flash(keys[0] ?? ""); })
      .catch((err: unknown) => {
        console.warn("punisher: settings save failed", err);
        const message = describe(err);
        setErrors((prev) => ({ ...prev, ...Object.fromEntries(keys.map((key) => [key, message])) }));
      })
      .finally(() =>
        setUnsaved((prev) => {
          const next: Record<string, unknown> = { ...prev };
          // 더 새로 바꾼 값(같은 칸을 또 바꿈)은 그 저장이 끝날 때 푼다.
          for (const [key, value] of Object.entries(patch)) {
            if (Object.is(next[key], value)) { delete next[key]; }
          }
          return next as Partial<P>;
        }),
      );
  };

  return { unsaved, errors, savedKey, saveField };
}
