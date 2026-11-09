import { Check, CheckCheck, Circle, CircleDashed, Lock, Star, type LucideIcon } from "lucide-react";
import { cn } from "@/src/shared/ui/kit";
import type { MapNodeState } from "./constants";

const ICONS: Record<MapNodeState, LucideIcon> = {
  completed: Check,
  known: CheckCheck,
  "in_progress": CircleDashed,
  recommended: Star,
  "not_started": Circle,
  locked: Lock,
};

/** 지도 상태 아이콘 — 색만으로 상태를 전달하지 않도록 항상 텍스트 레이블과 함께 쓴다. */
export function MapStateIcon({ state, className }: { state: MapNodeState; className?: string }) {
  const Icon = ICONS[state];
  return <Icon aria-hidden="true" className={cn("size-3.5 shrink-0", className)} />;
}
