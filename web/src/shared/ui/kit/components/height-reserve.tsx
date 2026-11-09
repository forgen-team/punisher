
import { useEffect, useRef, useState, type ReactNode } from "react";

const PREFIX = "witems:h:";
/** 폭이 비슷하면(50px 단위) 높이도 비슷하다 — 폭 구간별로 기억한다. */
const WIDTH_BUCKET = 50;

function storageKey(id: string): string {
  const bucket = typeof window === "undefined" ? 0 : Math.round(window.innerWidth / WIDTH_BUCKET);
  return `${PREFIX}${id}:${bucket}`;
}

function readHeight(key: string): number {
  try {
    return Number(window.localStorage.getItem(key)) || 0;
  } catch {
    return 0; // 저장소를 못 쓰면(사파리 사생활 보호 등) 예약 없이 스켈레톤 높이만 쓴다
  }
}

interface HeightReserveProps {
  /** 화면·섹션별 고유 키(예: "home-reads") */
  id: string;
  /** 데이터가 준비됐는가 — 준비 전에는 skeleton 을 그리고, 지난번 실제 높이만큼 자리를 잡는다 */
  ready: boolean;
  skeleton: ReactNode;
  children: ReactNode;
  className?: string;
}

/**
 * 로딩 뒤 레이아웃 시프트 방지 — 지난번에 그린 실제 높이(폭 구간별, localStorage)를 기억해 두었다가
 * 다음 로딩 때 스켈레톤 자리를 정확히 그 높이로 잡는다(스켈레톤이 더 길거나 짧아도 자리는 같음).
 * 처음 한 번은 skeleton 자체 높이로 잡으므로 skeleton 은 최종 모양과 비슷하게 그린다.
 */
export function HeightReserve({ id, ready, skeleton, children, className }: HeightReserveProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [key] = useState(() => storageKey(id));
  const [remembered] = useState(() => (typeof window === "undefined" ? 0 : readHeight(key)));

  useEffect(() => {
    const el = ref.current;
    if (!ready || !el || typeof ResizeObserver === "undefined") {return;}
    const save = () => {
      try {
        window.localStorage.setItem(key, String(Math.round(el.getBoundingClientRect().height)));
      } catch {
        // 저장 실패는 다음 로딩에서 예약만 못 할 뿐 — 화면에는 영향 없음
      }
    };
    const observer = new ResizeObserver(save);
    observer.observe(el);
    return () => observer.disconnect();
  }, [key, ready]);

  return (
    <div ref={ref} className={className} style={!ready && remembered > 0 ? { height: remembered, overflow: "hidden" } : undefined}>
      {ready ? children : skeleton}
    </div>
  );
}
