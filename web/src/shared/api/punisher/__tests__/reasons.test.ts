import { afterEach, describe, expect, it } from "vitest";
import { setCurrentLocale } from "@/src/shared/i18n";
import { reduceStage, INITIAL_STAGE } from "@/src/widgets/punisher/instant-stream/lib/stageState";
import { PunisherApiError, describePunisherError, localizeServerText, translateReasonCodes } from "..";
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

  it("로컬 Claude 실패 사유도 영어 화면에서 영어로(서버는 한국어 문구와 code 를 같이 보낸다)", () => {
    setCurrentLocale("en");
    const failed = reduceStage(INITIAL_STAGE, "stage", {
      stage: "failed",
      reasons: ["Claude 응답이 너무 오래 걸려 중단했습니다. 다시 시도해 주세요."],
      reasonCodes: ["claude.timeout"],
    });
    expect(failed.reasons).toEqual([reasonMessages.en["claude.timeout"]]);
    expect(Object.keys(reasonMessages.en).filter((k) => k.startsWith("claude."))).toHaveLength(8);
  });

  it("저장된 실패 문구: 아는 사유는 번역, 모르는 문구는 ko 에서만 그대로", () => {
    const timeoutKo = reasonMessages.ko["claude.timeout"];
    expect(localizeServerText(timeoutKo, "fallback", "en")).toBe(reasonMessages.en["claude.timeout"]);
    expect(localizeServerText(timeoutKo, "fallback", "ko")).toBe(timeoutKo);
    expect(localizeServerText("budget: 4/4 lectures", "fallback", "en")).toBe("fallback");
    expect(localizeServerText("내부 오류 원문", "fallback", "ko")).toBe("내부 오류 원문");
    expect(localizeServerText(undefined, "fallback", "ko")).toBe("fallback");
  });

  it("오류 code 가 카탈로그에 있으면 서버 문구 대신 화면 언어 안내", () => {
    setCurrentLocale("en");
    const err = new PunisherApiError(422, "CHAPTER_INVALID", "server text");
    expect(describePunisherError(err)).toMatch(/^The chapter design/);
  });
});
