export function lectureHref(id: string): string {
  return `/lectures/${encodeURIComponent(id)}`;
}

/**
 * 개념으로 강의를 여는 주소(/start?concept=) — 강의가 있으면 그 강의로, 없으면 만들기로 이어 간다(StartLectureView).
 * 오늘의 추천·강의 끝 [다음 강의]·Witems 오늘 홈이 같은 주소를 쓴다(진짜 링크, 새 탭 가능).
 */
export function startLectureHref(conceptId: string, query?: string): string {
  const params = new URLSearchParams({ concept: conceptId });
  if (query) { params.set("q", query); }
  return `/start?${params.toString()}`;
}

/**
 * 강의의 한 자리로 가는 주소 — `?at=<앵커>`. #해시는 Witems 셸이 앱에 넘기지 않아(계약 §3 path·search)
 * 쿼리로 둔다. 빈 앵커면 강의 처음(저장된 진도).
 */
export function lectureAnchorHref(id: string, anchor: string | null | undefined): string {
  return anchor ? `${lectureHref(id)}?at=${encodeURIComponent(anchor)}` : lectureHref(id);
}

/** 주소에서 강의 자리 앵커: `?at=` 우선, 옛 링크의 `#pa-` 도 읽는다. */
export function anchorFromLocation(search: string, hash: string): string {
  const at = new URLSearchParams(search).get("at");
  if (at) { return at; }
  if (!hash.startsWith("#pa-")) { return ""; }
  try {
    return decodeURIComponent(hash.slice(4));
  } catch {
    return hash.slice(4);
  }
}
