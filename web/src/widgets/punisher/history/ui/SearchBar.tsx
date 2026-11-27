import { useEffect, useRef, useState } from "react";
import { Search } from "lucide-react";
import { TextField } from "@forgen-team/ui";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { useHistoryNav } from "./useHistoryNav";

const DEBOUNCE_MS = 300;

/**
 * 기록의 검색창 1개(Toolbar) — 강의·개념·질문 해설·강의 중 질문·메모 전 범위. 검색어는 주소(?q=)에 남고,
 * 입력은 디바운스 뒤 replace 한다. 마우스·키보드 환경에서는 들어오면 바로 입력할 수 있게 포커스(⌘K 로 온 경우).
 */
export function SearchBar() {
  const t = useT(messages);
  const { q, go } = useHistoryNav();
  const [text, setText] = useState(q);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const latestGo = useRef(go);
  latestGo.current = go;
  // ui TextField 의 inputRef 는 input·textarea 겸용 타입이다.
  const inputRef = useRef<HTMLInputElement & HTMLTextAreaElement>(null);

  useEffect(() => {
    if (window.matchMedia?.("(pointer: fine)").matches) { inputRef.current?.focus(); }
  }, []);
  // 뒤로/앞으로·[검색어 지우기]로 주소의 검색어가 바뀌면 입력란도 따라간다(치는 중인 값과 같으면 그대로).
  useEffect(() => {
    setText((prev) => (prev.trim() === q ? prev : q));
  }, [q]);
  useEffect(() => () => {
    if (timer.current) { clearTimeout(timer.current); }
  }, []);

  function change(value: string) {
    setText(value);
    if (timer.current) { clearTimeout(timer.current); }
    timer.current = setTimeout(() => latestGo.current({ q: value.trim() }), DEBOUNCE_MS);
  }

  return (
    <div role="search">
      <TextField
        inputRef={inputRef}
        prefixIcon={<Search aria-hidden="true" />}
        placeholder={t.searchPlaceholder}
        value={text}
        onValueChange={({ value }) => change(value)}
        hideCharacterCount
        inputProps={{ type: "search", "aria-label": t.searchLabel, maxLength: 100 }}
      />
    </div>
  );
}
