
import { useEffect, useState } from "react";

/**
 * 넓은 화면(xl, 1280px 이상)인가 — 상세를 시트 대신 옆 열에 고정할 때 쓴다.
 * 재기 전(null)에는 시트로도 패널로도 열지 않아, 주소에 선택값이 있을 때 첫 화면에서 시트가 번쩍이지 않게 한다.
 */
export function useWideLayout(): boolean | null {
  const [wide, setWide] = useState<boolean | null>(null);
  useEffect(() => {
    const query = window.matchMedia("(min-width: 1280px)");
    const update = () => setWide(query.matches);
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);
  return wide;
}
