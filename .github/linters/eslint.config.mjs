import js from "@eslint/js";
import globals from "globals";

export default [
  js.configs.recommended,
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        // foedus cross-file globals
        initDashboardFeatures: "readonly",
        // 3rd-party
        L: "readonly",
        google: "readonly",
        bootstrap: "readonly",
      },
    },
    rules: {
      "no-empty": ["error", { allowEmptyCatch: true }],
      "no-unused-vars": ["error", { args: "none", caughtErrors: "none" }],
      "no-useless-escape": "off",
      "no-self-assign": "off",
    },
  },
];
