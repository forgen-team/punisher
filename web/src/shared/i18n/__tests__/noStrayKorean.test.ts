import { describe, expect, it } from "vitest";

/**
 * Punisher 화면 문구는 모듈별 i18n 카탈로그에만 둔다(설계 i18n §4). 카탈로그 밖에 한국어 리터럴이나
 * "ko-KR" 고정 로캘이 새로 생기면 실패한다. 주석·테스트와 아래 허용 파일(강의 본문 해석 표지, 서버로 보내는 프롬프트 글)은 제외.
 * 단독판: @types/node 없이 Vite 의 import.meta.glob(?raw)으로 읽고, 단독판 셸·화면(app·pages·shared/ui·
 * widgets/claude-status 등)까지 본다. 부품(@forgen-team/ui) 안 문구는 app/i18n.ts uiMessages 로 넘긴다.
 */
const SOURCES = import.meta.glob(
  [
    "../../../entities/punisher/**/*.{ts,tsx}",
    "../../../features/punisher/**/*.{ts,tsx}",
    "../../../widgets/**/*.{ts,tsx}",
    "../../../shared/ui/**/*.{ts,tsx}",
    "../../../shared/api/**/*.{ts,tsx}",
    "../../../shared/lib/**/*.{ts,tsx}",
    "../../../app/**/*.{ts,tsx}",
    "../../../pages/**/*.{ts,tsx}",
    "../../../main.tsx",
    "!../../../**/__tests__/**",
    "!../../../**/*.test.{ts,tsx}",
    "!../../../**/i18n.ts",
    "!../../../**/i18n/**",
  ],
  { query: "?raw", import: "default", eager: true },
) as Record<string, string>;

const ALLOWED = new Set([
  "entities/punisher/ui/constants.ts", // *_BY_LOCALE 의 ko 표
  "widgets/punisher/lecture-viewer/lib/blockData.ts", // 표 글 열 판별·내부 실패 이유
  "widgets/punisher/lecture-viewer/lib/inlineTerms.ts", // 한국어 강의 본문 복구 정규식
  "widgets/punisher/lecture-viewer/lib/afterSummary.ts", // 강의 본문의 '다음에 배울 개념' 문단 표지
  "features/punisher/lecture-exercise/lib/lab.ts", // 실습 표지 보는 곳:/예상:
  "features/punisher/lecture-exercise/lib/recall.ts", // 회상 표지 떠올릴 것: (서버 lecture.RecallParts)
  "features/punisher/prompt-edit/lib/lineDiff.ts", // 서버로 보내는 프롬프트 머리글
]);
const HANGUL = /[가-힣]/;
const COMMENT = /^\s*(\/\/|\*|\/\*|\{\/\*)/;
/** 코드 뒤 " // 설명" 주석(URL의 "://"는 앞에 공백이 없어 걸리지 않는다). */
const TRAILING_COMMENT = /\s\/\/\s.*$/;

describe("Punisher 화면 문구", () => {
  it("검사할 파일을 찾는다", () => {
    const paths = Object.keys(SOURCES).map((p) => p.replace(/^(\.\.\/)+/, ""));
    expect(paths).toContain("app/AppShell.tsx");
    expect(paths).toContain("pages/SettingsPage.tsx");
    expect(paths).toContain("widgets/punisher/today/TodayWidget.tsx");
  });

  it("카탈로그 밖에 한국어 리터럴·ko-KR 고정 로캘이 없다", () => {
    const stray: string[] = [];
    for (const [path, source] of Object.entries(SOURCES)) {
      const relativePath = path.replace(/^(\.\.\/)+/, "");
      let inBlock = false;
      source.split("\n").forEach((line, index) => {
        const opens = line.includes("/*"), closes = line.includes("*/");
        if (inBlock || COMMENT.test(line)) {
          inBlock = (inBlock || opens) && !closes;
          return;
        }
        if (opens && !closes) { inBlock = true; }
        const code = line.replace(TRAILING_COMMENT, "");
        if (code.includes('"ko-KR"') || (HANGUL.test(code) && !ALLOWED.has(relativePath))) {
          stray.push(`${relativePath}:${index + 1}: ${line.trim().slice(0, 80)}`);
        }
      });
    }
    expect(stray).toEqual([]);
  });
});
