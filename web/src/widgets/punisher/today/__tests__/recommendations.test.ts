import { describe, expect, it } from "vitest";
import { chapterDaysLeft, lectureMinutes, mapNodeHref, recommendationHref } from "../lib/recommendations";

describe("오늘의 추천 행", () => {
  it("강의가 있으면 그 강의, 없으면 /start 로 만들기(진짜 링크)", () => {
    expect(recommendationHref({ conceptId: "net.tcp", lectureId: "l 1" })).toBe("/lectures/l%201");
    expect(recommendationHref({ conceptId: "net.tcp" })).toBe("/start?concept=net.tcp");
  });

  it("지도에서 보기는 ?node=", () => {
    expect(mapNodeHref("db.index")).toBe("/map?node=db.index");
  });

  it("머리의 강의 분: 세션 합계 우선, 없으면 추천 분량 합", () => {
    expect(lectureMinutes({ lectureMinutes: 27.4, checkMinutes: 0, reviewMinutes: 0 }, [])).toBe(27);
    expect(lectureMinutes(undefined, [{ estimatedMinutes: 10 }, { estimatedMinutes: 12 }])).toBe(22);
    expect(lectureMinutes(undefined, [])).toBe(0);
  });

  it("남은 강의 일 수 = 남은 분 ÷ 하루 학습량", () => {
    expect(chapterDaysLeft({ remainingMinutes: 95, thresholdMinutes: 420, dailyMinutes: 30 })).toBe(3);
    expect(chapterDaysLeft({ remainingMinutes: 95, thresholdMinutes: 420, dailyMinutes: 0 })).toBe(0);
  });
});
