import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { CheckTurn, Prediction } from "@/src/shared/api/punisher";
import { isRevealNotice, showPrediction, topBarTitle } from "../lib/checkView";
import { CheckTurnItem } from "../ui/CheckTurnItem";
import { SYSTEM_TURN_LABELS } from "../ui/constants";

function turn(patch: Partial<CheckTurn>): CheckTurn {
  return { id: "t1", itemId: "i1", seq: 1, role: "tutor", move: "reveal", text: "", status: "ready", aided: true, reusedItem: false, createdAt: "", ...patch };
}

const prediction: Prediction = { objectiveId: "c#1", prompt: "어디서 읽을까요?", answer: "디스크", dontKnow: false, at: "" };

describe("상단 바 제목", () => {
  it("확인·복습·진단 모두 '제목 · 문항 n/N' 한 형식", () => {
    expect(topBarTitle("페이지 캐시", "문항 4/4")).toBe("페이지 캐시 · 문항 4/4");
    expect(topBarTitle("복습", "문항 3/3")).toBe("복습 · 문항 3/3");
    expect(topBarTitle("이미 앎 확인", "문항 3/3")).toBe("이미 앎 확인 · 문항 3/3");
    expect(topBarTitle("확인하기", "")).toBe("확인하기");
  });
});

describe("처음 예측 → 지금 답 줄", () => {
  it("예측과 지금 답이 모두 있을 때만 보인다", () => {
    expect(showPrediction({ prediction, answer: "메모리" })).toBe(true);
    expect(showPrediction({ prediction, answer: "  " })).toBe(false);
    expect(showPrediction({ prediction, answer: "" })).toBe(false);
    expect(showPrediction({ prediction: null, answer: "메모리" })).toBe(false);
  });
});

describe("답 공개 턴", () => {
  it("완료된 튜터 reveal 턴만 시스템 표기", () => {
    expect(isRevealNotice(turn({}))).toBe(true);
    expect(isRevealNotice(turn({ status: "pending" }))).toBe(false);
    expect(isRevealNotice(turn({ move: "narrow" }))).toBe(false);
    expect(isRevealNotice(turn({ role: "learner" }))).toBe(false);
  });

  it("빈 막대 대신 한 줄 '답을 공개했어요'", () => {
    render(
      <ul>
        <CheckTurnItem turn={turn({ verdict: "revealed" })} />
      </ul>,
    );
    expect(screen.getByRole("listitem").textContent).toBe(SYSTEM_TURN_LABELS.revealed);
  });
});
