/** 편집기 머리가 화면 안(상단 고정 바 아래 ~ 화면 아래)에 있지 않으면 true — 목록 아래쪽 항목을 골랐을 때 편집기로 데려간다. */
export function editorOutOfView(top: number, viewportHeight: number, headerOffset = 0): boolean {
  return top < headerOffset || top > viewportHeight * 0.6;
}
