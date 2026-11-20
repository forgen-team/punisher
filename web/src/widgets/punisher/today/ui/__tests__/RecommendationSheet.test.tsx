import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import type { Recommendation } from "@/src/shared/api/punisher";

const api = {
  dismissConcept: vi.fn(async (): Promise<void> => {
    throw new Error("boom");
  }),
};

vi.mock("@/src/shared/api/punisher", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/src/shared/api/punisher")>();
  return { ...actual, punisherApi: new Proxy(actual.punisherApi, { get: (target, key) => (key in api ? api[key as keyof typeof api] : Reflect.get(target, key)) }) };
});

const { RecommendationSheet } = await import("../RecommendationRow");

const rec = (conceptId: string): Recommendation => ({
  kind: "new_goal", conceptId, conceptName: conceptId, title: "", area: "network", depth: 1, estimatedMinutes: 10, reason: "", practicalSituation: "",
});

function Sheet({ item, client }: { item: Recommendation; client: QueryClient }) {
  return (
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <RecommendationSheet item={item} starting={false} onClose={() => undefined} onStart={() => undefined} />
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("추천 상세 시트", () => {
  it("clears the previous dismiss error when another recommendation opens", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
    const { rerender } = render(<Sheet item={rec("net.tcp")} client={client} />);
    fireEvent.click(screen.getByRole("button", { name: "필요 없음" }));
    await waitFor(() => expect(api.dismissConcept).toHaveBeenCalledWith("net.tcp"));
    const errorText = await screen.findByText(/boom/);
    expect(errorText).toBeInTheDocument();

    rerender(<Sheet item={rec("net.udp")} client={client} />);
    await waitFor(() => expect(screen.queryByText(/boom/)).toBeNull());
  });
});
