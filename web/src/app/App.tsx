import { useState, type ReactElement, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Provider as JotaiProvider } from "jotai";
import { BrowserRouter, Navigate, Route, Routes, useLocation } from "react-router-dom";
import { ScrollToTopOnNavigate } from "@/src/shared/lib/router";
import { ThemeSync } from "@/src/shared/lib/ThemeSync";
import { AppShell } from "./AppShell";
import { UIRoot } from "./UIRoot";
import {
  CardsPage, CheckPage, HistoryPage, InquiryPage, InstantPage, LecturePage, MapPage, NewChapterPage, NotFoundPage, PromptSettingsPage, SettingsPage, StartPage, TodayPage,
} from "@/src/pages";

function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { staleTime: 30_000, gcTime: 300_000, refetchOnWindowFocus: false, retry: 1 },
      mutations: { retry: false },
    },
  });
}

/**
 * Query client, atoms and the UI kit root (@forgen-team/ui) — shared by the standalone app and the Witems mount.
 * Witems passes inShell and its portal (ctx.portal) so overlays and toasts open inside the Shadow DOM.
 */
export function AppProviders({ children, inShell = false, portal = null }: { children: ReactNode; inShell?: boolean; portal?: HTMLElement | null }) {
  const [queryClient] = useState(createQueryClient);
  return (
    <JotaiProvider>
      <QueryClientProvider client={queryClient}>
        <UIRoot inShell={inShell} portal={portal}>{children}</UIRoot>
      </QueryClientProvider>
    </JotaiProvider>
  );
}

/** 없앤 주소(재설계 §2): /search?q= → /history?q=(쿼리 그대로) · /goals → /map?panel=goals. 옛 링크를 살리려 replace 로 옮긴다. */
function SearchRedirect() {
  const { search } = useLocation();
  return <Navigate to={`/history${search}`} replace />;
}

/** The screens under a shell (AppShell standalone, WitemsShell inside Witems). */
export function AppRoutes({ shell }: { shell: ReactElement }) {
  return (
    <Routes>
      <Route element={shell}>
        <Route index element={<TodayPage />} />
        <Route path="map" element={<MapPage />} />
        <Route path="goals" element={<Navigate to="/map?panel=goals" replace />} />
        <Route path="history" element={<HistoryPage />} />
        <Route path="search" element={<SearchRedirect />} />
        <Route path="chapters/new" element={<NewChapterPage />} />
        <Route path="lectures/:id" element={<LecturePage />} />
        <Route path="inquiries/:id" element={<InquiryPage />} />
        <Route path="check/:id" element={<CheckPage />} />
        <Route path="instant/:jobId" element={<InstantPage />} />
        <Route path="cards" element={<CardsPage />} />
        <Route path="start" element={<StartPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="settings/prompts" element={<PromptSettingsPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}

export function App() {
  return (
    <AppProviders>
      <ThemeSync />
      <BrowserRouter>
        <ScrollToTopOnNavigate />
        <AppRoutes shell={<AppShell />} />
      </BrowserRouter>
    </AppProviders>
  );
}
