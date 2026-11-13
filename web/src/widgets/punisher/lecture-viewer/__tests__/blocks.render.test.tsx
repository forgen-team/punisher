import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { LectureBlock, LectureSection } from "@/src/shared/api/punisher";
import { LectureBody } from "../ui/LectureBody";

// 코드 하이라이트(shiki)는 지연 로드 — 렌더 테스트에서는 원문 그대로 둔다.
vi.mock("../lib/highlight", () => ({ highlightToHtml: () => new Promise<string>(() => undefined) }));

const SAMPLE: LectureBlock[] = [
  { anchor: "k", type: "keypoint", kind: "definition", text: "**오버플로**는 표현 범위를 넘는 것." },
  {
    anchor: "t",
    type: "trace",
    data: { caption: "8비트 덧셈", columns: ["단계", "값"], rows: [["1", "127"], ["2", "128 → -128"], ["3", "-127"]], note: "끝" },
  },
  {
    anchor: "c",
    type: "chart",
    data: {
      kind: "line",
      title: "연산 수",
      x: { label: "n", values: [1, 10, 100] },
      y: { label: "횟수", scale: "log" },
      series: [
        { name: "n", values: [1, 10, 100] },
        { name: "n²", values: [1, 100, 10000] },
      ],
    },
  },
  { anchor: "d", type: "diff", lang: "sql", data: { before: "a int", after: "a bigint", note: "" } },
  { anchor: "p", type: "pause", text: "출력은 무엇일까요?" },
  { anchor: "o", type: "log", text: "-128" },
  { anchor: "e", type: "exercise", text: "16비트라면?" },
  { anchor: "a", type: "answer", text: "32767 다음은 -32768" },
  { anchor: "s", type: "summary", text: "- 하나\n- 둘\n- 셋" },
  { anchor: "u", type: "hologram" as LectureBlock["type"], text: "무시" },
  { anchor: "bad", type: "chart", data: { kind: "line", series: [] } },
];

function renderBlocks(blocks: LectureBlock[]) {
  const section: LectureSection = { anchor: "s1", kind: "principles", title: "원리", minutes: 0, blocks };
  return render(<LectureBody content={{ sections: [section] }} />);
}

describe("구조 블록 렌더", () => {
  it("renders all six kinds from sample data, ignores unknown types, and isolates broken data", () => {
    renderBlocks(SAMPLE);
    expect(screen.getByText("정의")).toBeInTheDocument();
    expect(screen.getByText("8비트 덧셈")).toBeInTheDocument();
    expect(screen.getAllByText("연산 수")).toHaveLength(2); // 보이는 제목 + 대체 표 caption
    expect(screen.getByText("고치기 전 · sql")).toBeInTheDocument();
    expect(screen.getByText("고친 후 · sql")).toBeInTheDocument();
    expect(screen.getByText("예상해 보세요")).toBeInTheDocument();
    expect(screen.getByText("직접 해 보기")).toBeInTheDocument();
    expect(screen.getByLabelText("한 장 요약")).toBeInTheDocument();
    expect(screen.queryByText("무시")).not.toBeInTheDocument();
    expect(screen.getByText("이 자료를 표시할 수 없어요")).toBeInTheDocument();
  });

  it("gives the chart a hidden data table", () => {
    renderBlocks([SAMPLE[2]]);
    const table = screen.getByRole("table");
    // 표는 내용 폭 아래로 줄지 않으므로 숨김은 감싼 요소가 맡는다(412px 가로 넘침 방지).
    expect(table).not.toHaveClass("sr-only");
    expect(table.parentElement).toHaveClass("sr-only");
    expect(screen.getByRole("cell", { name: "10,000" })).toBeInTheDocument();
  });

  it("reveals trace rows one by one and all at once", () => {
    renderBlocks([SAMPLE[1]]);
    expect(screen.getAllByRole("row")).toHaveLength(2); // 머리 + 1행
    fireEvent.click(screen.getByRole("button", { name: "다음 단계" }));
    expect(screen.getAllByRole("row")).toHaveLength(3);
    fireEvent.click(screen.getByRole("button", { name: "모두 보기" }));
    expect(screen.getAllByRole("row")).toHaveLength(4);
    expect(screen.queryByRole("button", { name: "다음 단계" })).not.toBeInTheDocument();
    expect(screen.getByText("끝")).toBeInTheDocument();
  });

  it("folds the output after a pause and the answer after an exercise (aria-expanded)", () => {
    renderBlocks(SAMPLE.slice(4, 8));
    const output = screen.getByRole("button", { name: "출력 보기" });
    expect(output).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("-128")).not.toBeInTheDocument();
    fireEvent.click(output);
    expect(output).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("-128")).toBeInTheDocument();

    const answer = screen.getByRole("button", { name: "풀이 보기" });
    expect(screen.queryByText("32767 다음은 -32768")).not.toBeInTheDocument();
    fireEvent.click(answer);
    expect(answer).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("32767 다음은 -32768")).toBeInTheDocument();
  });

  it("keeps anchors of folded blocks for progress restore", () => {
    const { container } = renderBlocks(SAMPLE.slice(4, 6));
    expect(container.querySelector('[data-lecture-anchor="o"]')).not.toBeNull();
  });

  it("renders the pause question as markdown and keeps misconception line breaks", () => {
    const { container } = renderBlocks([
      { anchor: "p", type: "pause", text: "`x++` 다음 값은?" },
      { anchor: "m", type: "misconception", text: "① 하나\n② 둘" },
    ]);
    expect(container.querySelector("code")?.textContent).toBe("x++");
    expect(screen.queryByText(/`x\+\+`/)).not.toBeInTheDocument();
    expect(container.querySelectorAll("aside br")).toHaveLength(1);
  });

  it("keeps trace number columns monospace and lets prose columns wrap", () => {
    renderBlocks([{ anchor: "t", type: "trace", data: { caption: "", columns: ["i", "설명"], rows: [["1", "첫 값을 더한다"], ["2", "다음 값을 더한다"]], note: "" } }]);
    const [num, prose] = screen.getAllByRole("cell");
    expect(num).toHaveClass("font-mono");
    expect(prose).not.toHaveClass("font-mono");
    expect(prose.firstElementChild).toHaveClass("max-w-[18rem]");
  });
});
