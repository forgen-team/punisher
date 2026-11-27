import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Concept } from "@/src/shared/api/punisher";

const concept = (over: Partial<Concept>): Concept => ({
  id: "x", name: "x", area: "os", depth: 1, summary: "", practicalCases: null, docLinks: null,
  status: "active", origin: "seed", objectiveCount: 1, userState: "not_started", updatedAt: "", ...over,
});

const concepts: Concept[] = [
  concept({ id: "os.process", area: "os" }),
  concept({ id: "os.thread", area: "os" }),
  concept({ id: "os.deep", area: "os", depth: 2 }),
  concept({ id: "db.index", area: "database" }),
  concept({ id: "net.tcp", area: "network-web", userState: "known" }),
];

const api = {
  listConcepts: vi.fn(async () => concepts),
  markKnown: vi.fn<(id: string, options?: { queueVerify?: boolean }) => Promise<unknown>>(async () => ({})),
};

vi.mock("@/src/shared/api/punisher", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/src/shared/api/punisher")>();
  return { ...actual, punisherApi: new Proxy(actual.punisherApi, { get: (target, key) => (key in api ? api[key as keyof typeof api] : Reflect.get(target, key)) }) };
});

Object.defineProperty(window, "scrollTo", { value: () => undefined, writable: true });

const { KnownAreasSheet } = await import("../KnownAreasSheet");

function renderSheet(onClose = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <KnownAreasSheet open onClose={onClose} />
    </QueryClientProvider>,
  );
  return onClose;
}

beforeEach(() => {
  api.markKnown.mockClear();
});

describe("이미 아는 개념 표시 시트", () => {
  it("offers one chip per area, the button waits for a pick, and an all-known area is disabled", async () => {
    renderSheet();
    const os = await screen.findByRole("checkbox", { name: "운영체제" });
    expect(screen.getByRole("checkbox", { name: "데이터베이스" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: /네트워크\/웹/ })).toBeDisabled();
    expect(screen.getByRole("button", { name: "이미 앎으로 표시" })).toBeDisabled();
    fireEvent.click(os);
    expect(screen.getByRole("button", { name: "이미 앎으로 표시" })).toBeEnabled();
  });

  it("marks only the basic (depth 1) concepts of the chosen areas as known, then closes", async () => {
    const onClose = renderSheet();
    fireEvent.click(await screen.findByRole("checkbox", { name: "운영체제" }));
    fireEvent.click(screen.getByRole("button", { name: "이미 앎으로 표시" }));
    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(api.markKnown.mock.calls.map(([id]) => id).sort()).toEqual(["os.process", "os.thread"]);
  });
});
