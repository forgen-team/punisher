
import { atomWithStorage, createJSONStorage } from "jotai/utils";

/** Reading position saved before jumping to a deeper concept ("더 깊게"). */
export interface LectureBookmark {
  lectureId: string;
  anchor: string;
  title: string;
}

export const lectureBookmarkAtom = atomWithStorage<LectureBookmark | null>(
  "punisher:return-bookmark",
  null,
  createJSONStorage<LectureBookmark | null>(() => sessionStorage),
  { getOnInit: false },
);
