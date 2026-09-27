/** @type {import("tailwindcss").Config} */
module.exports = {
  content: ["./web/templates/**/*.html"],
  safelist: [
    "cais-password-wrap",
    "cais-password-toggle",
    "cais-chat-scroll-down",
    "cais-thinking",
    "cais-thinking-dots",
    "cais-select-search",
    "cais-select-search-native",
    "cais-select-search-trigger",
    "cais-select-search-panel",
    "cais-select-search-input",
    "cais-select-search-list",
    "cais-select-search-option",
    "cais-select-search-label",
    "cais-select-search-chevron",
    "is-selected",
    "is-highlighted",
    "is-hidden",
    "amarra-rise",
    "amarra-rise-delay-1",
    "amarra-rise-delay-2",
    "amarra-rise-delay-3",
  ],
  theme: {
    extend: {
      // Serene Hearth — warm paper, terracotta and sage. Never pure black or
      // pure white: contrast is softened to keep the reading calm.
      colors: {
        linen: "#FAF7F2",
        parchment: "#F3EDE4",
        sand: "#E9DFD3",
        espresso: "#2B2623",
        umber: "#6E655F",
        clay: "#A39990",
        terracotta: "#D97746",
        "terracotta-soft": "#E08D64",
        "terracotta-deep": "#994619",
        sage: "#5A7865",
        "sage-mist": "#8FA89B",
        gold: "#C99A45",
      },
      fontFamily: {
        sans: ['"Plus Jakarta Sans"', "system-ui", "sans-serif"],
        serif: ["Literata", "Georgia", "serif"],
        display: ["Literata", "Georgia", "serif"],
        mono: ["ui-monospace", "SFMono-Regular", "Menlo", "monospace"],
      },
      borderRadius: {
        sm: "0.25rem",
        DEFAULT: "0.5rem",
        md: "0.75rem",
        lg: "1rem",
        xl: "1.5rem",
      },
      boxShadow: {
        soft: "0px 8px 24px -4px rgba(43, 38, 35, 0.05)",
        float: "0px 16px 36px -6px rgba(43, 38, 35, 0.08)",
        "2xs": "0 1px 2px 0 rgb(0 0 0 / 0.05)",
        xs: "0 1px 2px 0 rgb(0 0 0 / 0.05)",
      },
      keyframes: {
        "hearth-fade": {
          from: { opacity: "0", transform: "translateY(0.75rem)" },
          to: { opacity: "1", transform: "none" },
        },
        "hearth-breathe": {
          "0%, 100%": { opacity: "0.45", transform: "scale(1)" },
          "50%": { opacity: "0.85", transform: "scale(1.06)" },
        },
      },
      animation: {
        "hearth-fade": "hearth-fade 0.7s cubic-bezier(0.22, 1, 0.36, 1) both",
        "hearth-breathe": "hearth-breathe 6s ease-in-out infinite",
      },
    },
  },
  plugins: [],
};
