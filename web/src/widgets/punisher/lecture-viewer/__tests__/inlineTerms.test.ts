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

  // 2026-11-10: "이 점은 @@term 에서 다루는 ALU 설계"가 "이 점은 에서 다루는…"으로 비어 보였다.
  it("never leaves a hole where a nameless directive was", () => {
    expect(toInlineTermLinks("이 점은 @@term 에서 다루는 ALU 설계와도 이어집니다.")).toBe("이 점은 관련 개념에서 다루는 ALU 설계와도 이어집니다.");
    expect(toInlineTermLinks("자세한 내용은 @@term 를 봅니다")).toBe("자세한 내용은 관련 개념을 봅니다");
    expect(toInlineTermLinks("다음에 볼 것: @@term.")).toBe("다음에 볼 것: 관련 개념.");
    expect(toInlineTermLinks("@@term concept=os.thread 에서 다룹니다")).toBe("관련 개념에서 다룹니다");
    expect(toInlineTermLinks("이것은 @@term 이고 저것은 @@term 라고 합니다")).toBe("이것은 관련 개념이고 저것은 관련 개념이라고 합니다");
    expect(toInlineTermLinks("@@term{}를 봅니다")).toBe("관련 개념을 봅니다");
  });

  it("keeps the name in every stored form", () => {
    expect(toInlineTermLinks("새 @@term{프로세스}를 하나 만듭니다.")).toBe("새 프로세스를 하나 만듭니다.");
    expect(toInlineTermLinks("(@@term{자원 제한}).")).toBe("(자원 제한).");
    expect(toInlineTermLinks("@@term term=스레드 를 봅니다")).toBe("스레드 를 봅니다");
    expect(toInlineTermLinks("@@term concept=os.ebpf proposed term=eBPF 로 봅니다")).toBe("[eBPF](term:os.ebpf) 로 봅니다");
    expect(toInlineTermLinks("관련된 @@term 엔디언은 다시 설명하지 않습니다.")).toBe("관련된 엔디언은 다시 설명하지 않습니다.");
    expect(toInlineTermLinks("이어집니다. → @@term 부동소수점")).toBe("이어집니다. → 부동소수점");
  });

  it("drops bare directives and leaves normal text alone", () => {
    expect(toInlineTermLinks("대표적으로 @@term 배열의 인덱스 접근")).toBe("대표적으로 배열의 인덱스 접근");
    expect(toInlineTermLinks("[스레드](term:os.thread)는 그대로")).toBe("[스레드](term:os.thread)는 그대로");
  });
});
