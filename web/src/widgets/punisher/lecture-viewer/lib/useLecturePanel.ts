import { useCallback, useEffect, useRef } from "react";
import { usePathname, useRouter, useSearchParams } from "@/src/shared/lib/router";

export type LecturePanel = "toc" | "notes";

export function parsePanel(value: string | null): LecturePanel | null {
  return value === "toc" || value === "notes" ? value : null;
}

/** 다음 두 프레임 뒤(시트가 닫히고 스크롤 잠금이 풀린 뒤) 실행. */
function afterPaint(run: () => void) {
  requestAnimationFrame(() => requestAnimationFrame(run));
}

/**
 * 폰 목차·질문과 메모 시트의 열림 상태 = 주소 `?panel=toc|notes` (push — 뒤로가기가 닫기).
 * 닫기: 이 화면에서 연 시트면 뒤로(기록 하나 되돌림), 주소로 바로 들어왔으면 panel 만 지운다(replace).
 * after: 닫은 뒤 할 일(목차 항목으로 스크롤 등) — 뒤로가기의 스크롤 복원이 끝난 다음에 부른다.
 */
export function useLecturePanel() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const panel = parsePanel(params.get("panel"));
  const pushed = useRef(false);

  useEffect(() => {
    if (!panel) { pushed.current = false; }
  }, [panel]);

  const hrefWith = useCallback(
    (next: LecturePanel | null) => {
      const query = new URLSearchParams(params);
      if (next) { query.set("panel", next); } else { query.delete("panel"); }
      const search = query.toString();
      return search ? `${pathname}?${search}` : pathname;
    },
    [params, pathname],
  );

  const open = useCallback(
    (next: LecturePanel) => {
      pushed.current = true;
      router.push(hrefWith(next), { scroll: false, guard: false });
    },
    [router, hrefWith],
  );

  const close = useCallback(
    (after?: () => void) => {
      if (pushed.current) {
        pushed.current = false;
        if (after) { window.addEventListener("popstate", () => afterPaint(after), { once: true }); }
        router.back();
        return;
      }
      router.replace(hrefWith(null), { scroll: false, guard: false });
      if (after) { afterPaint(after); }
    },
    [router, hrefWith],
  );

  return { panel, open, close };
}
