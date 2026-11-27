import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { LearningSettings, LearningSettingsPatch, LearningSettingsResponse, LevelSuggestion } from "@/src/shared/api/punisher";

const settings: LearningSettings = {
  exampleLanguage: "typescript", dailyMinutes: 30, masteryTarget: 4, reviewIntervals: [1, 3, 7], reviewMinutes: 10,
  reviewGrader: "self", pretestEnabled: true, labOs: "macos", adaptiveIntervals: false, locale: "ko",
  readerLevel: "practitioner", areaFamiliarity: {},
};

let suggestion: LevelSuggestion | null = null;
const api = {
  getLearningSettings: vi.fn(async (): Promise<LearningSettingsResponse> => ({ settings, exampleLanguages: [], levelSuggestion: suggestion })),
  saveLearningSettings: vi.fn(async (patch: LearningSettingsPatch): Promise<LearningSettingsResponse> => ({
    settings: { ...settings, ...patch }, exampleLanguages: [], levelSuggestion: null,
  })),
  dismissLevelSuggestion: vi.fn(async () => undefined),
};

vi.mock("@/src/shared/api/punisher", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/src/shared/api/punisher")>();
  return { ...actual, punisherApi: new Proxy(actual.punisherApi, { get: (target, key) => (key in api ? api[key as keyof typeof api] : Reflect.get(target, key)) }) };
});

const { LevelSuggestionCard } = await import("../LevelSuggestionCard");

function renderCard() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <LevelSuggestionCard />
    </QueryClientProvider>,
  );
}

describe("수준 보정 카드", () => {
  beforeEach(() => {
    suggestion = { to: "novice", reason: "easier_often" };
  });

  it("is hidden without a suggestion", async () => {
    suggestion = null;
    renderCard();
    await waitFor(() => expect(api.getLearningSettings).toHaveBeenCalled());
    expect(screen.queryByText("설명을 더 풀어 쓸까요?")).toBeNull();
  });

  it("바꾸기 saves the suggested reader level", async () => {
    renderCard();
    expect(await screen.findByText("설명을 더 풀어 쓸까요?")).toBeInTheDocument();
    expect(screen.getByText("바꿀 수준: 코드를 조금 써 봤어요")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "바꾸기" }));
    await waitFor(() => expect(api.saveLearningSettings).toHaveBeenCalledWith({ readerLevel: "novice" }));
    await waitFor(() => expect(screen.queryByText("설명을 더 풀어 쓸까요?")).toBeNull());
  });

  it("닫기 dismisses the suggestion", async () => {
    renderCard();
    fireEvent.click(await screen.findByRole("button", { name: "닫기" }));
    await waitFor(() => expect(api.dismissLevelSuggestion).toHaveBeenCalled());
    expect(api.saveLearningSettings).not.toHaveBeenCalled();
    await waitFor(() => expect(screen.queryByText("설명을 더 풀어 쓸까요?")).toBeNull());
  });
});
