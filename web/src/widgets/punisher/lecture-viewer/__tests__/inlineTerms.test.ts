import { describe, expect, it } from "vitest";
import { toInlineTermLinks } from "../lib/inlineTerms";

describe("toInlineTermLinks", () => {
  it("converts legacy inline directives to term links", () => {
    expect(toInlineTermLinks("좀비는 @@term concept=os.process-lifecycle-fork-exec term=프로세스 생성과 생명주기 에서 다룹니다.")).toBe(
      "좀비는 [프로세스 생성과 생명주기](term:os.process-lifecycle-fork-exec) 에서 다룹니다.",
    );
    expect(toInlineTermLinks("한 프로세스 안의 @@term concept=os.thread term=스레드 들은 공유합니다")).toBe(
      "한 프로세스 안의 [스레드](term:os.thread) 들은 공유합니다",
    );
    expect(toInlineTermLinks("@@term concept=infra.container term=컨테이너 와 @@term concept=os.namespaces term=네임스페이스 에서")).toBe(
      "[컨테이너](term:infra.container) 와 [네임스페이스](term:os.namespaces) 에서",
    );
    expect(toInlineTermLinks("- @@term concept=os.thread term=스레드 : 설명")).toBe("- [스레드](term:os.thread) : 설명");
  });

  it("drops bare directives and leaves normal text alone", () => {
    expect(toInlineTermLinks("대표적으로 @@term 배열의 인덱스 접근")).toBe("대표적으로 배열의 인덱스 접근");
    expect(toInlineTermLinks("[스레드](term:os.thread)는 그대로")).toBe("[스레드](term:os.thread)는 그대로");
  });
});
