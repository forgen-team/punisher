import { usePathname, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { historyHref, resolveHistoryView, type HistoryTab, type NoteKind } from "../lib/historyParams";

export interface HistoryPatch {
  tab?: HistoryTab;
  q?: string;
  area?: string;
  kind?: NoteKind | "";
}

/** 기록 화면의 주소 상태(?tab=&q=&area=&kind=)와 바꾸는 법. 탭·칩·검색어는 모두 replace(기록에 쌓지 않는다). */
export function useHistoryNav() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const view = resolveHistoryView(params.get("tab"), params.get("kind"));
  const q = (params.get("q") ?? "").trim();
  const area = params.get("area") ?? "";

  function hrefFor(patch: HistoryPatch): string {
    return historyHref(pathname, {
      tab: patch.tab ?? view.tab,
      q: patch.q ?? q,
      area: patch.area ?? area,
      kind: patch.kind ?? view.kind,
    });
  }

  function go(patch: HistoryPatch) {
    router.replace(hrefFor(patch), { scroll: false });
  }

  return { view, q, area, hrefFor, go };
}
