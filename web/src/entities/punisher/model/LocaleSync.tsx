
import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAtom } from "jotai";
import { isLocale, localeAtom, rememberLocale, setCurrentLocale } from "@/src/shared/i18n";
import { isWitems } from "@/src/shared/config/runtime";
import { useLearningSettings } from "./reading";

/**
 * 학습 설정의 언어를 화면 언어(atom·<html lang>)로 맞추고, 바뀌면 서버 값(개념 이름 등)을 다시 읽는다.
 * 단독판: 맞춘 언어를 localStorage 에 기억해 다음 실행의 첫 화면을 그 언어로 그린다(shared/i18n initialLocale).
 */
export function LocaleSync() {
  const { data } = useLearningSettings();
  const [locale, setLocale] = useAtom(localeAtom);
  const queryClient = useQueryClient();
  const next = data?.settings.locale;

  // 첫 화면은 기억한 언어(localStorage)로 그리므로 <html lang> 도 처음부터 맞춘다.
  // Witems 안에서는 <html> 이 셸의 것이라 건드리지 않는다.
  useEffect(() => {
    if (isWitems()) { return; }
    document.documentElement.lang = locale;
  }, [locale]);

  useEffect(() => {
    if (!isLocale(next)) { return; }
    rememberLocale(next);
    if (next === locale) { return; }
    setCurrentLocale(next);
    setLocale(next);
    void queryClient.invalidateQueries({ predicate: (q) => q.queryKey[0] === "punisher" && q.queryKey[1] !== "settings" });
  }, [next, locale, setLocale, queryClient]);

  return null;
}
