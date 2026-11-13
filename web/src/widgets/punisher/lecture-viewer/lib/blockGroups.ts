/**
 * 접는 블록 묶기 — `@@pause` 는 바로 다음 출력(code·log)을 접어 두고, `@@exercise` 는 바로 다음 `@@answer` 를 접힌 풀이로 품는다.
 * 짝이 없으면 혼자 그린다(pause 는 질문만, answer 는 접힌 풀이만).
 */
import type { LectureBlock } from "@/src/shared/api/punisher";

export type BlockGroup =
  | { type: "single"; block: LectureBlock }
  | { type: "pause"; pause: LectureBlock; output?: LectureBlock }
  | { type: "exercise"; exercise: LectureBlock; answer?: LectureBlock };

export function groupBlocks(blocks: LectureBlock[]): BlockGroup[] {
  const groups: BlockGroup[] = [];
  for (let index = 0; index < blocks.length; index += 1) {
    const block = blocks[index];
    const next = blocks[index + 1] as LectureBlock | undefined;
    if (block.type === "pause") {
      const output = next && (next.type === "code" || next.type === "log") ? next : undefined;
      groups.push({ type: "pause", pause: block, output });
      if (output) { index += 1; }
    } else if (block.type === "exercise") {
      const answer = next?.type === "answer" ? next : undefined;
      groups.push({ type: "exercise", exercise: block, answer });
      if (answer) { index += 1; }
    } else {
      groups.push({ type: "single", block });
    }
  }
  return groups;
}

/** trace 한 행씩 펼치기 — 처음엔 1행, '다음 단계'는 1행 더, '모두 보기'는 전부 */
export function revealNext(shown: number, total: number): number {
  return Math.min(total, Math.max(1, shown + 1));
}

export function initialReveal(total: number): number {
  return Math.min(1, total);
}
