import type { Config } from "tailwindcss";
import { compat } from "@forgen-team/tailwind-preset";

// 색·글자·간격 클래스는 @forgen-team/tailwind-preset(이관 기간용 compat). 값은 @forgen-team/tokens 의 CSS 변수.
const config: Config = {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  presets: [compat],
  plugins: [],
};

export default config;
