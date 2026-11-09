import type { Config } from "tailwindcss";
import sharedPreset from "./tailwind.preset.js";

// Colors, type scale and spacing come from the design-token preset (ported from @witems/config).
const config: Config = {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  presets: [sharedPreset],
  plugins: [],
};

export default config;
