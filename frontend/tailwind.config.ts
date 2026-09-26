import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        background: "var(--background)",
        foreground: "var(--foreground)",
        // Airline brand: deep navy with a gold accent.
        navy: {
          50: "#EEF3FA",
          100: "#D9E3F2",
          200: "#B3C7E4",
          300: "#86A4D0",
          400: "#5A80BA",
          500: "#3A62A0",
          600: "#274C85",
          700: "#1B3A6B",
          800: "#122A52",
          900: "#0B1D3A",
          950: "#071327",
        },
        gold: {
          50: "#FDF7E7",
          100: "#FAECC4",
          200: "#F4D98A",
          300: "#EEC454",
          400: "#E4AE2B",
          500: "#CF9416",
          600: "#A87311",
          700: "#7E5510",
        },
      },
      fontFamily: {
        sans: ["var(--font-geist-sans)", "system-ui", "-apple-system", "sans-serif"],
        mono: ["var(--font-geist-mono)", "ui-monospace", "monospace"],
      },
      boxShadow: {
        card: "0 1px 2px rgba(11, 29, 58, 0.06), 0 1px 3px rgba(11, 29, 58, 0.08)",
        lift: "0 10px 30px -12px rgba(11, 29, 58, 0.35)",
      },
    },
  },
  plugins: [],
};
export default config;
