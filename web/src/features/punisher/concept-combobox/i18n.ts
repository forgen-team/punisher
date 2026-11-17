import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    selected: "선택한 개념",
    remove: (name: string) => `${name} 제거`,
    full: "최대 개수를 골랐어요",
    loading: "개념 불러오는 중…",
    search: "개념 이름 또는 ID 검색",
    options: (label: string) => `${label} 후보`,
  },
  en: {
    selected: "Selected concepts",
    remove: (name: string) => `Remove ${name}`,
    full: "You've picked the maximum",
    loading: "Loading concepts…",
    search: "Search by concept name or ID",
    options: (label: string) => `${label} options`,
  },
});
