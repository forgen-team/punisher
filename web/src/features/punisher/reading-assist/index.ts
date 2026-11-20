export { useTextSelection, type TextSelectionSnapshot } from "./useTextSelection";
export { ReadingThreadsProvider, useReadingThreads, type ThreadFocus, type ThreadPassage } from "./ReadingThreadsProvider";
export { PassageMarkers, PassageThreadList, ReadingAssist, ThreadCountBadge, THREAD_PANEL_QUERY } from "./ThreadSurfaces";
export { ThreadView } from "./ThreadView";
export { splitSentences, sentenceAt, type SentenceSpan } from "./lib/sentences";
export { countPassageThreads, groupThreads, groupPassages, isLiveThread, passageQuestionIds, type ReadingThread, type PassageThreads } from "./lib/threads";
