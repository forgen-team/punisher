export interface OverlayPlacementInput {
  /** 대상의 뷰포트 좌표 */
  anchor: { top: number; bottom: number; left: number };
  /** 패널이 줄지 않았을 때의 전체 높이(헤더·본문 전부·푸터) */
  naturalHeight: number;
  width: number;
  viewportW: number;
  viewportH: number;
  /** 위·아래 고정 바가 차지하는 높이(앱 헤더·하단 탭) */
  topInset: number;
  bottomInset: number;
  gap: number;
  margin: number;
}

export interface OverlayPlacement {
  top: number;
  left: number;
  maxHeight: number;
}

/**
 * 팝오버 자리(뷰포트 좌표) — 화면·고정 바 밖으로 나가지 않게 한다.
 * 대상 아래 → 위 순서로 전부 들어가는 쪽에 붙이고, 어느 쪽에도 안 들어가면 대상을 일부 덮더라도 화면 안에서 내용을 최대한 보여 준다.
 * 그래도 넘치면 maxHeight 로 줄이고 패널 안에서 스크롤한다.
 */
export function placeOverlay(i: OverlayPlacementInput): OverlayPlacement {
  const minTop = i.topInset + i.margin;
  const maxBottom = i.viewportH - i.bottomInset - i.margin;
  const left = Math.max(i.margin, Math.min(i.anchor.left, i.viewportW - i.width - i.margin));
  const below = i.anchor.bottom + i.gap;
  if (below + i.naturalHeight <= maxBottom) {
    return { top: below, left, maxHeight: maxBottom - below };
  }
  const above = i.anchor.top - i.gap - i.naturalHeight;
  if (above >= minTop) {
    return { top: above, left, maxHeight: i.naturalHeight };
  }
  const available = maxBottom - minTop;
  const top = Math.max(minTop, Math.min(i.anchor.top, maxBottom - Math.min(i.naturalHeight, available)));
  return { top, left, maxHeight: maxBottom - top };
}
