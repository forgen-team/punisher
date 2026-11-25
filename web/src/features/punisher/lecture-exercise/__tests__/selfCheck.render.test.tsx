import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ExerciseAttempt, ExerciseInput } from "@/src/shared/api/punisher";
import { blockContentHash } from "../lib/contentHash";
import { parseRecall } from "../lib/recall";
import { PausePanel, RecallPanel, SummaryRecall } from "../ui/SelfCheck";

const submit = vi.fn((input: ExerciseInput) => Promise.resolve({ ...input } as unknown as ExerciseAttempt));
let attempts: ExerciseAttempt[] = [];

vi.mock("../model/ExerciseContext", () => ({
  useExerciseContext: () => ({ attempts, submit, remove: vi.fn(), submitError: null }),
  findAttempt: (list: ExerciseAttempt[], anchor: string, hash: string) => list.find((a) => a.anchor === anchor && a.contentHash === hash),
}));

function attempt(over: Partial<ExerciseAttempt>): ExerciseAttempt {
  return { id: "1", lectureId: "l", anchor: "a", contentHash: "", kind: "recall", answer: "", verdict: "", feedback: "", status: "done", createdAt: "", ...over };
}

beforeEach(() => {
  submit.mockClear();
  attempts = [];
});

describe("예측 멈춤", () => {
  it("opens the output after a prediction or a confidence, then one tap grades it", () => {
    render(<PausePanel anchor="w-2" question="무엇이 나올까요?" outputText="-128" output={<pre>-128</pre>} fallback={null} />);
    const reveal = screen.getByRole("button", { name: "출력 보기" });
    expect(reveal).toBeDisabled();
    expect(screen.queryByText("-128")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("radio", { name: "확실" }));
    fireEvent.change(screen.getByLabelText("내 예측(한 줄)"), { target: { value: "128" } });
    fireEvent.click(reveal);
    expect(screen.getByText("-128")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "달랐어요" }));
    expect(submit).toHaveBeenCalledWith({
      anchor: "w-2", contentHash: blockContentHash("무엇이 나올까요?", "-128"), verdict: "incorrect", answer: "128", confidence: "sure",
    });
  });

  it("can be skipped", () => {
    render(<PausePanel anchor="w-2" question="q" outputText="o" output={<pre>o</pre>} fallback={null} />);
    fireEvent.click(screen.getByRole("button", { name: "예측 없이 보기" }));
    expect(submit).toHaveBeenCalledWith({ anchor: "w-2", contentHash: blockContentHash("q", "o"), skip: true });
  });
});

describe("떠올려 보기", () => {
  const text = "방금 본 쿼리의 결과는?\n떠올릴 것: 1 한 행";

  it("hides the key until recalled, then grades with one tap", () => {
    render(<RecallPanel anchor="w-4" text={text} renderBody={(body) => <p>{body}</p>} />);
    expect(screen.getByText("방금 본 쿼리의 결과는?")).toBeInTheDocument();
    expect(screen.queryByText("1 한 행")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "떠올렸어요, 답 보기" }));
    expect(screen.getByText("1 한 행")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "흐릿해요" }));
    expect(submit).toHaveBeenCalledWith({ anchor: "w-4", contentHash: blockContentHash(text, ""), verdict: "partial" });
  });

  it("shows the grade and the review it went into", () => {
    attempts = [attempt({ anchor: "w-4", contentHash: blockContentHash(text, ""), verdict: "forgot", reviewItemId: "item" })];
    render(<RecallPanel anchor="w-4" text={text} renderBody={(body) => <p>{body}</p>} />);
    expect(screen.getByRole("status")).toHaveTextContent("못 떠올렸어요 · 하루 뒤 복습에 넣었어요");
  });

  it("splits the question and the key like the server", () => {
    expect(parseRecall("Q?\nRecall: a\nb")).toEqual({ question: "Q?", key: "a\nb" });
    expect(parseRecall("질문만")).toEqual({ question: "질문만", key: "" });
  });
});

describe("요약 전 떠올리기", () => {
  it("keeps the summary closed until a recall is written or skipped", () => {
    render(<SummaryRecall anchor="j-2" text="- a" summary={<p>요약 본문</p>} />);
    expect(screen.queryByText("요약 본문")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "요약 보기" })).toBeDisabled();
    fireEvent.change(screen.getByLabelText("요약을 열기 전에 핵심을 떠올려 적어 보세요"), { target: { value: "세 가지" } });
    fireEvent.click(screen.getByRole("button", { name: "요약 보기" }));
    expect(submit).toHaveBeenCalledWith({ anchor: "j-2", contentHash: blockContentHash("- a", ""), answer: "세 가지" });
  });

  it("opens at once once recalled", () => {
    attempts = [attempt({ anchor: "j-2", contentHash: blockContentHash("- a", ""), kind: "summary", verdict: "written" })];
    render(<SummaryRecall anchor="j-2" text="- a" summary={<p>요약 본문</p>} />);
    expect(screen.getByText("요약 본문")).toBeInTheDocument();
  });
});
