import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Provider as JotaiProvider } from "jotai";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import { ThemeSync } from "@/src/shared/lib/ThemeSync";
import { AppShell } from "./AppShell";
import {
  HistoryPage, InstantPage, LecturePage, MapPage, NotFoundPage, PromptSettingsPage, SettingsPage, TodayPage,
} from "@/src/pages";

function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { staleTime: 30_000, gcTime: 300_000, refetchOnWindowFocus: false, retry: 1 },
      mutations: { retry: false },
    },
  });
}

export function App() {
  const [queryClient] = useState(createQueryClient);
  return (
    <JotaiProvider>
      <QueryClientProvider client={queryClient}>
        <ThemeSync />
        <BrowserRouter>
          <Routes>
            <Route element={<AppShell />}>
              <Route index element={<TodayPage />} />
              <Route path="map" element={<MapPage />} />
              <Route path="goals" element={<MapPage />} />
              <Route path="history" element={<HistoryPage />} />
              <Route path="lectures/:id" element={<LecturePage />} />
              <Route path="instant/:jobId" element={<InstantPage />} />
              <Route path="settings" element={<SettingsPage />} />
              <Route path="settings/prompts" element={<PromptSettingsPage />} />
              <Route path="*" element={<NotFoundPage />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </JotaiProvider>
  );
}
