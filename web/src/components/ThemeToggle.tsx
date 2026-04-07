import { useTheme } from "../context/ThemeContext";

export default function ThemeToggle() {
  const { mode, cycle } = useTheme();

  const label = mode === "system" ? "Auto" : mode === "light" ? "Light" : "Dark";
  const title = `Theme: ${label} (click to cycle)`;

  return (
    <button
      onClick={cycle}
      title={title}
      aria-label={title}
      className="inline-flex items-center justify-center w-7 h-7 rounded border border-[var(--color-border)] text-[var(--color-fg-subtle)] hover:text-[var(--color-fg)] hover:border-[var(--color-fg-subtle)] transition-colors"
    >
      {mode === "system" && (
        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth={1.8} viewBox="0 0 24 24">
          <rect x="3" y="4" width="18" height="12" rx="1.5" />
          <path strokeLinecap="round" d="M8 20h8M12 16v4" />
        </svg>
      )}
      {mode === "light" && (
        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth={1.8} viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="4" />
          <path strokeLinecap="round" d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
        </svg>
      )}
      {mode === "dark" && (
        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth={1.8} viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z" />
        </svg>
      )}
    </button>
  );
}
