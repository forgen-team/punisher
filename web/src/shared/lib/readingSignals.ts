import { createContext, useContext } from "react";

/** 강의 화면의 읽기 신호 기록기(features/punisher/reading-signals 가 제공) — 없으면 기록하지 않는다. */
export interface ReadingSignalsRecorder {
  /** 본문 블록(anchor)에서 용어 팝오버를 열었다 */
  termOpened: (conceptId: string, anchor: string) => void;
}

export const ReadingSignalsContext = createContext<ReadingSignalsRecorder | null>(null);

export function useReadingSignals(): ReadingSignalsRecorder | null {
  return useContext(ReadingSignalsContext);
}
