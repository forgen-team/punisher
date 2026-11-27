import eslint from "@eslint/js";
import tseslint from "typescript-eslint";
import reactPlugin from "eslint-plugin-react";
import reactHooksPlugin from "eslint-plugin-react-hooks";
import unicornPlugin from "eslint-plugin-unicorn";
import forgen from "@forgen-team/eslint-plugin";

// Ported from @witems/config/eslint/flat.js (same rules, browser-only app).
export default [
  { ignores: ["node_modules/**", "dist/**", "**/*.d.ts"] },
  eslint.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
      parser: tseslint.parser,
      parserOptions: { ecmaFeatures: { jsx: true } },
      globals: { window: "readonly", document: "readonly", navigator: "readonly" },
    },
    plugins: {
      "@typescript-eslint": tseslint.plugin,
      react: reactPlugin,
      "react-hooks": reactHooksPlugin,
      unicorn: unicornPlugin,
    },
    settings: { react: { version: "detect" } },
    rules: {
      "react/react-in-jsx-scope": "off",
      "react/prop-types": "off",
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "@typescript-eslint/no-unused-vars": ["warn", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
      "no-console": ["warn", { allow: ["warn", "error", "info"] }],
      "unicorn/no-array-callback-reference": "error",
      "unicorn/prefer-array-some": "error",
      "react/function-component-definition": ["error", { namedComponents: "function-declaration" }],
      "unicorn/text-encoding-identifier-case": "error",
    },
  },
  // forgen 디자인 규칙(@forgen-team/eslint-plugin) — 이관 중이라 지금은 warn(기존 위반으로 CI 를 막지 않는다). 다 고치면 error 로.
  {
    files: ["src/**/*.{ts,tsx}"],
    plugins: { "@forgen-team": forgen },
    rules: Object.fromEntries(Object.keys(forgen.rules).map((id) => [`@forgen-team/${id}`, "warn"])),
  },
];
