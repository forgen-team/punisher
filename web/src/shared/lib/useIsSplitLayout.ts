
import { useEffect, useState } from "react";

/** Tailwind preset 의 `fold` 브레이크포인트와 같은 값 — 폴드 펼침(약 680~900px)부터 목록+상세 2단. */
export const SPLIT_LAYOUT_MIN_WIDTH = 600;
export const SPLIT_LAYOUT_QUERY = `(min-width: ${SPLIT_LAYOUT_MIN_WIDTH}px)`;

/**
 * 2단(목록+상세) 레이아웃 여부. 첫 렌더부터 실제 값을 읽어(초기 false → true 로 바뀌는 레이아웃 시프트 방지)
 * 접기/펴기(창 크기 변경) 때 바로 갱신한다.
 */
export function useIsSplitLayout(): boolean {
  const [split, setSplit] = useState(() => typeof window !== "undefined" && window.matchMedia(SPLIT_LAYOUT_QUERY).matches);
  useEffect(() => {
    const mql = window.matchMedia(SPLIT_LAYOUT_QUERY);
    const onChange = () => setSplit(mql.matches);
    onChange();
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, []);
  return split;
}
