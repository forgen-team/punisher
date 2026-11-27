import { useId } from "react";
import { cn } from "@/src/shared/lib/cn";

/** 브랜드 한 줄 — 설정 > 정보에만 둔다. */
export const BRAND_LINE = "punisher · part of the forgen family · © 2026 jang-ujin";

/**
 * Punisher 로고(assets/logo.svg 와 같은 기하): 반쪽 시계(하루 30분) 아래 펼친 책에서 forgen 불꽃이 피어오른다.
 * Handoff 와 같은 그라디언트·선 굵기를 쓴다(forgen 패밀리).
 */
export function PunisherLogo({ size = 24, className }: { size?: number; className?: string }) {
  const id = useId().replace(/[^a-zA-Z0-9]/g, "");
  const accent = `pa${id}`;
  const flame = `pf${id}`;
  return (
    <svg viewBox="0 0 64 64" width={size} height={size} fill="none" aria-hidden="true" className={cn("shrink-0", className)}>
      <defs>
        <linearGradient id={accent} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#f97316" />
          <stop offset="50%" stopColor="#ef4444" />
          <stop offset="100%" stopColor="#f59e0b" />
        </linearGradient>
        <linearGradient id={flame} x1="0%" y1="100%" x2="0%" y2="0%">
          <stop offset="0%" stopColor="#ef4444" />
          <stop offset="55%" stopColor="#f97316" />
          <stop offset="100%" stopColor="#f59e0b" />
        </linearGradient>
      </defs>
      <path d="M10 33 A22 22 0 0 1 54 33" stroke={`url(#${accent})`} strokeWidth="6" strokeLinecap="round" />
      <path d="M32 55 C26 49.5 16 48.5 6 50.5" stroke={`url(#${accent})`} strokeWidth="6" strokeLinecap="round" />
      <path d="M32 55 C38 49.5 48 48.5 58 50.5" stroke={`url(#${accent})`} strokeWidth="6" strokeLinecap="round" />
      <path
        transform="translate(1.6 4.25) scale(0.95)"
        d="M32 15 C37 22 42 27 41 35 C40.3 41 36.5 45 32 45 C27 45 23 41 23.5 35.5 C24 31 27 28.5 27.5 23.5 C30 25.5 30.8 28 30.8 30.5 C33 27 33.5 21 32 15 Z"
        fill={`url(#${flame})`}
      />
    </svg>
  );
}
