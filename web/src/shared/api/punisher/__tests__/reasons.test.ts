import { afterEach, describe, expect, it } from "vitest";
import { setCurrentLocale } from "@/src/shared/i18n";
import { reduceStage, INITIAL_STAGE } from "@/src/widgets/punisher/instant-stream/lib/stageState";
import { PunisherApiError, describePunisherError, translateReasonCodes } from "..";
import { reasonMessages } from "../i18n";

afterEach(() => setCurrentLocale("ko"));

describe("서버 사유 code 번역", () => {
  it("아는 code 는 화면 언어로, 모르는 code 가 섞이면 빈 배열", () => {
    expect(translateReasonCodes(["review.factual_error", "volume"], "en")).toEqual([
      reasonMessages.en["review.factual_error"],
      reasonMessages.en.volume,
    ]);
    expect(translateReasonCodes(["format", "llm_written"], "en")).toEqual([]);
    expect(translateReasonCodes(undefined)).toEqual([]);
  });

  it("stage 실패 이벤트는 reasonCodes 를 우선하고 없으면 reasons 를 쓴다", () => {
    setCurrentLocale("en");
    const coded = reduceStage(INITIAL_STAGE, "stage", { stage: "failed", reasons: ["원문"], reasonCodes: ["timeout"] });
    expect(coded.reasons).toEqual([reasonMessages.en.timeout]);
    const raw = reduceStage(INITIAL_STAGE, "stage", { stage: "failed", reasons: ["model text"] });
    expect(raw.reasons).toEqual(["model text"]);
  });

  it("오류 code 가 카탈로그에 있으면 서버 문구 대신 화면 언어 안내", () => {
    setCurrentLocale("en");
    const err = new PunisherApiError(422, "CHAPTER_INVALID", "server text");
    expect(describePunisherError(err)).toMatch(/^The chapter design/);
  });
});
